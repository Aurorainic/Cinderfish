package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"milnfish/internal/config"
	"milnfish/internal/httpapi"
	"milnfish/internal/media"
)

// wsURL converts an httptest server URL to its WebSocket form.
func wsURL(ts *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http") + path
}

// newStreamingServer starts a server with the Streaming endpoint wired.
func newStreamingServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: mode, FederationMode: "public",
		Media: config.Media{Backend: "local"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("local storage: %v", err)
	}
	srv.SetStorage(storage)
	srv.AttachStreaming(nil)

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// dial opens a streaming connection.
func dial(t *testing.T, ts *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	u := wsURL(ts, "/streaming")
	if token != "" {
		u += "?i=" + token
	}
	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial: %v (status %d)", err, status)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// sendFrame writes a client frame.
func sendFrame(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	if err := conn.WriteJSON(v); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// message is one server-to-client frame.
type message struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

// readWithTimeout waits for the next frame.
func readWithTimeout(t *testing.T, conn *websocket.Conn, d time.Duration) (message, bool) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(d))
	var m message
	if err := conn.ReadJSON(&m); err != nil {
		return message{}, false
	}
	return m, true
}

// waitForChannelMessage waits until a "channel" frame for the given
// channel id arrives, skipping unrelated frames.
func waitForChannelMessage(t *testing.T, conn *websocket.Conn, channelID string, d time.Duration) (map[string]any, bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		m, ok := readWithTimeout(t, conn, time.Until(deadline))
		if !ok {
			return nil, false
		}
		if m.Type != "channel" {
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(m.Body, &body); err != nil {
			continue
		}
		if body["id"] != channelID {
			continue
		}
		return body, true
	}
	return nil, false
}

// signup creates an account and returns its native token.
func signup(t *testing.T, ts *httptest.Server, username string) (id, token string) {
	t.Helper()
	resp, out := post(t, ts, "/api/signup", map[string]any{
		"username": username,
		"password": "correct horse battery staple",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup: status %d body %v", resp.StatusCode, out)
	}
	user, _ := out["id"].(string)
	if user == "" {
		t.Fatalf("signup response has no id: %v", out)
	}
	// The native token is issued at signup time under "token" when the
	// instance exposes it; otherwise read it from the database.
	tok, _ := out["token"].(string)
	if tok == "" {
		tok = nativeToken(t, username)
	}
	return user, tok
}

// nativeToken reads a user's native token from the database.
func nativeToken(t *testing.T, username string) string {
	t.Helper()
	pool := testPool(t)
	var tok string
	if err := pool.QueryRow(t.Context(),
		"SELECT COALESCE(token, '') FROM users WHERE username_lower = lower($1)", username).Scan(&tok); err != nil {
		t.Fatalf("read token: %v", err)
	}
	if tok == "" {
		t.Fatalf("user %s has no native token", username)
	}
	return tok
}
