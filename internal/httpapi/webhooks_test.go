package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type delivered struct {
	headers http.Header
	body    map[string]any
}

func TestWebhookCRUDAndOwnership(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	resp, out := postWithToken(t, f.ts, "/api/i/webhooks/create", aliceToken, map[string]any{
		"name": "note hook", "url": "https://example.com/hook", "secret": "s3cret",
		"on": []string{"note", "reply"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d, body %v", resp.StatusCode, out)
	}
	id, _ := out["id"].(string)
	if id == "" || out["active"] != true {
		t.Fatalf("created webhook = %v", out)
	}

	resp, raw := postRawWithToken(t, f.ts, "/api/i/webhooks/list", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status %d, body %v", resp.StatusCode, raw)
	}
	if arr, _ := raw.([]any); len(arr) != 1 {
		t.Errorf("alice webhooks = %d, want 1", len(arr))
	}
	// Another account cannot see or touch it.
	resp, raw = postRawWithToken(t, f.ts, "/api/i/webhooks/show", bobToken, map[string]any{"webhookId": id})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("bob show status = %d, want 404", resp.StatusCode)
	}
	resp, raw = postRawWithToken(t, f.ts, "/api/i/webhooks/list", bobToken, map[string]any{})
	if arr, _ := raw.([]any); len(arr) != 0 {
		t.Errorf("bob webhooks = %d, want 0", len(arr))
	}

	// Update flips active off, keeping the other fields.
	resp, out = postWithToken(t, f.ts, "/api/i/webhooks/update", aliceToken, map[string]any{
		"webhookId": id, "name": "renamed", "url": "https://example.com/hook2",
		"secret": "other", "on": []string{"renote"}, "active": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: status %d, body %v", resp.StatusCode, out)
	}
	if out["name"] != "renamed" || out["active"] != false {
		t.Errorf("updated webhook = %v", out)
	}

	// Delete is idempotent and scoped to the owner.
	if status := postNoContentWithToken(t, f.ts, "/api/i/webhooks/delete", aliceToken, map[string]any{"webhookId": id}); status != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", status)
	}
	if status := postNoContentWithToken(t, f.ts, "/api/i/webhooks/delete", bobToken, map[string]any{"webhookId": id}); status != http.StatusNoContent {
		t.Errorf("second delete status = %d, want 204", status)
	}
	resp, raw = postRawWithToken(t, f.ts, "/api/i/webhooks/list", aliceToken, map[string]any{})
	if arr, _ := raw.([]any); len(arr) != 0 {
		t.Errorf("webhooks after delete = %d, want 0", len(arr))
	}
}

func TestWebhookValidation(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	cases := []map[string]any{
		{"name": "", "url": "https://example.com", "secret": "x", "on": []string{"note"}},
		{"name": "n", "url": "ftp://example.com", "secret": "x", "on": []string{"note"}},
		{"name": "n", "url": "https://example.com", "secret": "", "on": []string{"note"}},
		{"name": "n", "url": "https://example.com", "secret": "x", "on": []string{}},
		{"name": "n", "url": "https://example.com", "secret": "x", "on": []string{"bogus"}},
	}
	for i, body := range cases {
		resp, _ := postWithToken(t, f.ts, "/api/i/webhooks/create", token, body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("case %d status = %d, want 400", i, resp.StatusCode)
		}
	}
}

func TestWebhookDeliveryOnNote(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	ch := make(chan delivered, 4)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ch <- delivered{headers: r.Header.Clone(), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	// The parent is created before the hook exists, so only the two notes under
	// test produce deliveries.
	parent := createSearchNote(t, f, token, "parent note", "public")

	resp, out := postWithToken(t, f.ts, "/api/i/webhooks/create", token, map[string]any{
		"name": "delivery", "url": receiver.URL, "secret": "hook-secret",
		"on": []string{"note", "reply"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d, body %v", resp.StatusCode, out)
	}
	hookID, _ := out["id"].(string)

	createSearchNote(t, f, token, "webhook delivery note", "public")
	got := waitForDelivery(t, ch)
	if got.body["type"] != "note" {
		t.Errorf("event type = %v, want note", got.body["type"])
	}
	if got.body["hookId"] != hookID {
		t.Errorf("hookId = %v, want %s", got.body["hookId"], hookID)
	}
	if got.headers.Get("X-Firefish-Hook-Secret") != "hook-secret" {
		t.Errorf("secret header = %q", got.headers.Get("X-Firefish-Hook-Secret"))
	}
	body, _ := got.body["body"].(map[string]any)
	if body["text"] != "webhook delivery note" {
		t.Errorf("delivered note text = %v", body["text"])
	}

	// A reply fires the reply event.
	resp, out = postWithToken(t, f.ts, "/api/notes/create", token, map[string]any{
		"text": "reply body", "visibility": "public", "replyId": parent,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reply create: status %d body %v", resp.StatusCode, out)
	}
	got = waitForDelivery(t, ch)
	if got.body["type"] != "reply" {
		t.Errorf("reply event type = %v, want reply", got.body["type"])
	}

	// latestStatus is written back after delivery.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, out = postWithToken(t, f.ts, "/api/i/webhooks/show", token, map[string]any{"webhookId": hookID})
		if resp.StatusCode == http.StatusOK && out["latestStatus"] == float64(200) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("latestStatus was not recorded: %v", out)
}

func waitForDelivery(t *testing.T, ch <-chan delivered) delivered {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(4 * time.Second):
		t.Fatal("webhook was not delivered within 4s")
		return delivered{}
	}
}

// TestWebhookDeliveryFailureStatus covers the two failure outcomes: a receiver
// that answers 500 records that status, and an unreachable receiver records 1
// (the port's marker for a transport error).
func TestWebhookDeliveryFailureStatus(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()

	resp, out := postWithToken(t, f.ts, "/api/i/webhooks/create", token, map[string]any{
		"name": "failing", "url": failing.URL, "secret": "s", "on": []string{"note"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create failing hook: %d %v", resp.StatusCode, out)
	}
	hookID, _ := out["id"].(string)

	createSearchNote(t, f, token, "failure delivery", "public")
	if status := waitForLatestStatus(t, f, token, hookID, func(v float64) bool { return v == 500 }); status != 500 {
		t.Errorf("latestStatus after 500 = %v, want 500", status)
	}

	// A closed port refuses the connection; the marker is 1.
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	resp, out = postWithToken(t, f.ts, "/api/i/webhooks/create", token, map[string]any{
		"name": "unreachable", "url": deadURL, "secret": "s", "on": []string{"note"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create unreachable hook: %d %v", resp.StatusCode, out)
	}
	deadID, _ := out["id"].(string)

	createSearchNote(t, f, token, "unreachable delivery", "public")
	if status := waitForLatestStatus(t, f, token, deadID, func(v float64) bool { return v == 1 }); status != 1 {
		t.Errorf("latestStatus after transport error = %v, want 1", status)
	}
}

// TestWebhookIgnoresUnproducedEvents documents the port's limit: only note,
// reply and renote events are produced, so a hook subscribed to follow hears
// nothing even when a follow happens.
func TestWebhookIgnoresUnproducedEvents(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	ch := make(chan delivered, 4)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ch <- delivered{headers: r.Header.Clone(), body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	resp, out := postWithToken(t, f.ts, "/api/i/webhooks/create", aliceToken, map[string]any{
		"name": "follow only", "url": receiver.URL, "secret": "s", "on": []string{"follow"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, out)
	}

	if resp, body := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: %d %v", resp.StatusCode, body)
	}
	select {
	case got := <-ch:
		t.Errorf("follow delivered to a webhook that should not hear it: %v", got.body)
	case <-time.After(700 * time.Millisecond):
	}
}

// waitForLatestStatus polls one webhook until its latestStatus satisfies done,
// returning the last value it saw.
func waitForLatestStatus(t *testing.T, f *fedInstance, token, hookID string, done func(float64) bool) float64 {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	var last any
	for time.Now().Before(deadline) {
		resp, out := postWithToken(t, f.ts, "/api/i/webhooks/show", token, map[string]any{"webhookId": hookID})
		if resp.StatusCode == http.StatusOK {
			last = out["latestStatus"]
			if v, ok := last.(float64); ok && done(v) {
				return v
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if v, ok := last.(float64); ok {
		return v
	}
	return 0
}
