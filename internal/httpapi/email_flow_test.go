package httpapi_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// localSMTP accepts connections and reports each delivered message body.
type localSMTP struct {
	host string
	port int
	msgs chan string
}

func startLocalSMTP(t *testing.T) *localSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	addr := ln.Addr().(*net.TCPAddr)
	s := &localSMTP{host: "127.0.0.1", port: addr.Port, msgs: make(chan string, 4)}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	return s
}

func (s *localSMTP) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(line string) { _, _ = w.WriteString(line); _ = w.Flush() }
	write("220 local ESMTP\r\n")
	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		upper := strings.ToUpper(line)
		if inData {
			if line == ".\r\n" {
				inData = false
				s.msgs <- data.String()
				write("250 ok\r\n")
				continue
			}
			data.WriteString(line)
			continue
		}
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250-local\r\n250 HELP\r\n")
		case strings.HasPrefix(upper, "MAIL FROM"):
			write("250 ok\r\n")
		case strings.HasPrefix(upper, "RCPT TO"):
			// One address is rejected so tests can prove a failed recipient
			// does not stop the others.
			if strings.Contains(upper, "FAIL@EXAMPLE.COM") {
				write("550 no such user\r\n")
			} else {
				write("250 ok\r\n")
			}
		case strings.HasPrefix(upper, "DATA"):
			inData = true
			write("354 go\r\n")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 bye\r\n")
			return
		default:
			write("250 ok\r\n")
		}
	}
}

// waitMail returns the next message, failing after a few seconds.
func (s *localSMTP) waitMail(t *testing.T) string {
	t.Helper()
	select {
	case msg := <-s.msgs:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("no mail arrived")
		return ""
	}
}

var emailLinkPattern = regexp.MustCompile(`https://fish\.example/(verify-email|reset-password)/([A-Za-z0-9_-]+)`)

// TestEmailVerificationAndResetFlow runs the real SMTP path end to end: the
// server sends a verification link over a live socket, redeeming it marks the
// address verified, and the password-reset link sets a new password.
func TestEmailVerificationAndResetFlow(t *testing.T) {
	smtpSrv := startLocalSMTP(t)
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		Media: config.Media{Backend: "local"},
		Mail:  config.Mail{Host: smtpSrv.host, Port: smtpSrv.port, TLS: "none", From: "no-reply@fish.example"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	srv.SetStorage(storage)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	_, signup := post(t, ts, "/api/signup", map[string]any{"username": "mailuser", "password": "password-123"})
	token := signup["token"].(string)

	// Changing the address sends a verification link to it.
	resp, updated := post(t, ts, "/api/i/update-email", map[string]any{
		"i": token, "password": "password-123", "email": "mailuser@example.com",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update-email = %d %v", resp.StatusCode, updated)
	}
	if updated["emailVerified"] != false {
		t.Errorf("emailVerified before redeeming = %v", updated["emailVerified"])
	}
	verifyMail := smtpSrv.waitMail(t)
	if !strings.Contains(verifyMail, "mailuser@example.com") || !strings.Contains(verifyMail, "/verify-email/") {
		t.Fatalf("verification mail = %s", verifyMail)
	}
	match := emailLinkPattern.FindStringSubmatch(verifyMail)
	if match == nil || match[1] != "verify-email" {
		t.Fatalf("no verification link in %s", verifyMail)
	}
	verifyCode := match[2]

	if status := postNoContentWithToken(t, ts, "/api/verify-email", token, map[string]any{"code": verifyCode}); status != http.StatusNoContent {
		t.Fatalf("verify-email = %d, want 204", status)
	}
	_, me := post(t, ts, "/api/i", map[string]any{"i": token})
	if me["emailVerified"] != true {
		t.Errorf("emailVerified after redeeming = %v", me["emailVerified"])
	}

	// Redeeming the same code again fails: challenges are single use.
	if status := postNoContentWithToken(t, ts, "/api/verify-email", token, map[string]any{"code": verifyCode}); status != http.StatusBadRequest {
		t.Errorf("reused code = %d, want 400", status)
	}

	// The reset link is sent to the verified address and sets a new password.
	if status := postNoContentWithToken(t, ts, "/api/request-reset-password", "", map[string]any{
		"username": "mailuser", "email": "mailuser@example.com",
	}); status != http.StatusNoContent {
		t.Fatalf("request-reset-password = %d, want 204", status)
	}
	resetMail := smtpSrv.waitMail(t)
	resetMatch := emailLinkPattern.FindStringSubmatch(resetMail)
	if resetMatch == nil || resetMatch[1] != "reset-password" {
		t.Fatalf("no reset link in %s", resetMail)
	}
	resetCode := resetMatch[2]

	if status := postNoContentWithToken(t, ts, "/api/reset-password", "", map[string]any{
		"token": resetCode, "password": "brand-new-password",
	}); status != http.StatusNoContent {
		t.Fatalf("reset-password = %d, want 204", status)
	}
	if resp, _ := post(t, ts, "/api/signin", map[string]any{"username": "mailuser", "password": "password-123"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("old password still signs in: %d", resp.StatusCode)
	}
	if resp, signed := post(t, ts, "/api/signin", map[string]any{"username": "mailuser", "password": "brand-new-password"}); resp.StatusCode != http.StatusOK || signed["finished"] != true {
		t.Errorf("new password signin = %d %v", resp.StatusCode, signed)
	}
}
