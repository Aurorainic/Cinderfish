package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// mailTestServer builds an instance with a live SMTP sink and a database.
func mailTestServer(t *testing.T, smtp *localSMTP) (*httptest.Server, *httpapi.Server) {
	t.Helper()
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		Media: config.Media{Backend: "local"},
		Mail:  config.Mail{Host: smtp.host, Port: smtp.port, TLS: "none", From: "no-reply@fish.example"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	srv.AttachFederation()
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	srv.SetStorage(storage)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, srv
}

// subscribeMail marks a profile verified and sets the notification reasons it
// wants mailed.
func subscribeMail(t *testing.T, srv *httpapi.Server, userID, address string, verified bool, types []string, announcements bool) {
	t.Helper()
	raw := `[]`
	if len(types) > 0 {
		raw = `["` + strings.Join(types, `","`) + `"]`
	}
	if _, err := srv.TestPool().Exec(context.Background(), `
UPDATE user_profiles
SET email = $2, email_verified = $3,
    settings = jsonb_set(
        jsonb_set(COALESCE(settings, '{}'::jsonb), '{emailNotificationTypes}', $4::jsonb),
        '{receiveAnnouncementEmail}', $5::jsonb)
WHERE user_id = $1
`, userID, address, verified, raw, announcements); err != nil {
		t.Fatalf("subscribe %s: %v", address, err)
	}
}

// noMail fails when a message arrives within the window.
func (s *localSMTP) noMail(t *testing.T, window time.Duration) {
	t.Helper()
	select {
	case msg := <-s.msgs:
		t.Fatalf("unexpected mail: %s", msg)
	case <-time.After(window):
	}
}

// TestNotificationMailReasons drives the four mailed notification reasons
// through the real endpoints and SMTP sink, plus the gates that must stay
// quiet.
func TestNotificationMailReasons(t *testing.T) {
	smtp := startLocalSMTP(t)
	ts, srv := mailTestServer(t, smtp)

	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password-123"})
	bobID, _ := bob["id"].(string)
	bobToken, _ := bob["token"].(string)
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password-123"})
	aliceToken, _ := alice["token"].(string)
	_, carol := post(t, ts, "/api/signup", map[string]any{"username": "carol", "password": "password-123"})
	carolID, _ := carol["id"].(string)
	carolToken, _ := carol["token"].(string)
	_, dave := post(t, ts, "/api/signup", map[string]any{"username": "dave", "password": "password-123"})
	daveID, _ := dave["id"].(string)

	subscribeMail(t, srv, bobID, "bob@example.com", true,
		[]string{"reply", "mention", "follow", "receiveFollowRequest"}, true)
	subscribeMail(t, srv, carolID, "carol@example.com", true, nil, false)
	subscribeMail(t, srv, daveID, "dave@example.com", false,
		[]string{"reply", "mention", "follow", "receiveFollowRequest"}, false)

	bobNote := createNote(t, ts, bobToken, map[string]any{"text": "bob's note"})

	// reply
	if resp, out := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "answering bob", "replyId": bobNote,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("reply: %d %v", resp.StatusCode, out)
	}
	reply := smtp.waitMail(t)
	if !strings.Contains(reply, "To: bob@example.com") || !strings.Contains(reply, "回复了你的帖子") || !strings.Contains(reply, "@alice") {
		t.Errorf("reply mail = %s", reply)
	}

	// mention
	if resp, out := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "@bob look at this",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("mention: %d %v", resp.StatusCode, out)
	}
	mention := smtp.waitMail(t)
	if !strings.Contains(mention, "To: bob@example.com") || !strings.Contains(mention, "提到了你") {
		t.Errorf("mention mail = %s", mention)
	}
	if !strings.Contains(mention, "/notes/") {
		t.Errorf("mention mail has no note link: %s", mention)
	}

	// follow
	if resp, out := postWithToken(t, ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow bob: %d %v", resp.StatusCode, out)
	}
	follow := smtp.waitMail(t)
	if !strings.Contains(follow, "To: bob@example.com") || !strings.Contains(follow, "@alice 关注了你") {
		t.Errorf("follow mail = %s", follow)
	}

	// receiveFollowRequest: bob locks, carol follows.
	if resp, out := postWithToken(t, ts, "/api/i/update", bobToken, map[string]any{"isLocked": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("lock bob: %d %v", resp.StatusCode, out)
	}
	if resp, out := postWithToken(t, ts, "/api/following/create", carolToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("carol follows locked bob: %d %v", resp.StatusCode, out)
	}
	request := smtp.waitMail(t)
	if !strings.Contains(request, "To: bob@example.com") || !strings.Contains(request, "@carol 请求关注你") {
		t.Errorf("request mail = %s", request)
	}

	// carol is verified but subscribed to nothing, so a mention sends no mail.
	if resp, out := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "@carol quiet please",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("mention carol: %d %v", resp.StatusCode, out)
	}
	smtp.noMail(t, 700*time.Millisecond)

	// dave subscribed but his address is not verified.
	if resp, out := post(t, ts, "/api/notes/create", map[string]any{
		"i": aliceToken, "text": "@dave not verified",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("mention dave: %d %v", resp.StatusCode, out)
	}
	smtp.noMail(t, 700*time.Millisecond)
}

// TestAnnouncementMailCoverage proves creating an announcement mails the
// verified subscribers and nobody else.
func TestAnnouncementMailCoverage(t *testing.T) {
	smtp := startLocalSMTP(t)
	ts, srv := mailTestServer(t, smtp)

	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password-123"})
	bobID, _ := bob["id"].(string)
	_, carol := post(t, ts, "/api/signup", map[string]any{"username": "carol", "password": "password-123"})
	carolID, _ := carol["id"].(string)
	subscribeMail(t, srv, bobID, "bob@example.com", true, nil, true)
	subscribeMail(t, srv, carolID, "carol@example.com", true, nil, false)

	if _, err := srv.TestPool().Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, token, is_admin)
VALUES ('root1', 'root', 'root', 'x', 'admintoken000000', TRUE)`); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	resp, out := postWithToken(t, ts, "/api/admin/announcements/create", "admintoken000000", map[string]any{
		"title": "维护通知", "text": "今晚重启。",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create announcement: %d %v", resp.StatusCode, out)
	}
	mail := smtp.waitMail(t)
	if !strings.Contains(mail, "To: bob@example.com") || !strings.Contains(mail, "维护通知") || !strings.Contains(mail, "今晚重启。") {
		t.Errorf("announcement mail = %s", mail)
	}
	smtp.noMail(t, 700*time.Millisecond)
}
