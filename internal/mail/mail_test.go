package mail_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/mail"
)

// fakeSMTP accepts one message and reports what it received.
type fakeSMTP struct {
	addr  string
	lines chan string
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	lines := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		write := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }
		write("220 fake ESMTP\r\n")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					lines <- data.String()
					write("250 ok\r\n")
					continue
				}
				data.WriteString(line)
				continue
			}
			upper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				write("250-fake\r\n250 HELP\r\n")
			case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
				write("250 ok\r\n")
			case strings.HasPrefix(upper, "DATA"):
				inData = true
				write("354 go ahead\r\n")
			case strings.HasPrefix(upper, "QUIT"):
				write("221 bye\r\n")
				return
			default:
				write("250 ok\r\n")
			}
		}
	}()
	return &fakeSMTP{addr: ln.Addr().String(), lines: lines}
}

// TestSMTPSenderDelivers proves the SMTP path really speaks the protocol and
// that a header value cannot inject extra headers.
func TestSMTPSenderDelivers(t *testing.T) {
	srv := startFakeSMTP(t)
	host, port, err := net.SplitHostPort(srv.addr)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		t.Fatalf("port: %v", err)
	}

	sender := mail.New(mail.Settings{
		Host: host, Port: portNum, From: "no-reply@fish.example", TLS: "none",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sender.Send(ctx, mail.Message{
		To:      "alice@fish.example",
		Subject: "Verify\r\nBcc: victim@example.com",
		Body:    "click https://fish.example/verify?code=x\n",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case raw := <-srv.lines:
		if !strings.Contains(raw, "To: alice@fish.example") {
			t.Errorf("message has no To header: %s", raw)
		}
		if strings.Contains(strings.ToLower(raw), "\r\nbcc:") {
			t.Errorf("header injection was not sanitised: %s", raw)
		}
		if !strings.Contains(raw, "click https://fish.example/verify?code=x") {
			t.Errorf("body missing: %s", raw)
		}
		if !strings.Contains(raw, "Content-Type: text/plain") {
			t.Errorf("content type missing: %s", raw)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no message delivered")
	}
}

// TestSMTPDialFailure reports a dial error instead of hanging or panicking.
func TestSMTPDialFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // nothing listens here any more

	host, port, _ := net.SplitHostPort(addr)
	var portNum int
	fmt.Sscanf(port, "%d", &portNum)
	sender := mail.New(mail.Settings{Host: host, Port: portNum, TLS: "none"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = sender.Send(ctx, mail.Message{To: "x@example.com", Subject: "s", Body: "b"})
	if err == nil || !strings.Contains(err.Error(), "mail: dial") {
		t.Errorf("error = %v, want a dial error", err)
	}
}

// TestLogSenderWithoutHost documents the no-SMTP fallback.
func TestLogSenderWithoutHost(t *testing.T) {
	if _, ok := mail.New(mail.Settings{}).(mail.LogSender); !ok {
		t.Error("an empty host must select the log sender")
	}
}
