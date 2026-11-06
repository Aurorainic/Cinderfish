package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAPShowResolvesRemoteObjects proves ap/show: a URL on another instance
// resolves to a stored note or user, packed in the normal shapes.
func TestAPShowResolvesRemoteObjects(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)

	viewerToken, _ := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	noteID := createNote(t, b.ts, bobToken, map[string]any{"text": "hello from beta"})

	// A note URL is fetched, stored and packed.
	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/notes/" + noteID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show note = %d %v", resp.StatusCode, out)
	}
	if out["type"] != "Note" {
		t.Fatalf("type = %v, want Note", out["type"])
	}
	note, _ := out["object"].(map[string]any)
	if note == nil || note["id"] == "" {
		t.Fatalf("packed note = %v", out["object"])
	}
	if note["text"] != "hello from beta" {
		t.Errorf("note text = %v", note["text"])
	}

	// An actor URL resolves to a user.
	resp, out = postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/users/" + bobID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show user = %d %v", resp.StatusCode, out)
	}
	if out["type"] != "User" {
		t.Fatalf("type = %v, want User", out["type"])
	}
	user, _ := out["object"].(map[string]any)
	if user == nil || user["username"] != "betabob" {
		t.Fatalf("packed user = %v", out["object"])
	}
}

// TestAPShowRequiresCredential keeps the lookup authenticated.
func TestAPShowRequiresCredential(t *testing.T) {
	a := newInstance(t, "alpha")
	resp, _ := post(t, a.ts, "/api/ap/show", map[string]any{"uri": "https://beta.example/notes/x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous ap/show = %d, want 401", resp.StatusCode)
	}
}

// TestAPShowRejectsNonEntity proves an ActivityPub collection (or any object
// that is neither a note nor an actor) is refused instead of stored.
func TestAPShowRejectsNonEntity(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	viewerToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/users/" + bobID + "/followers",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("collection = %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "NO_SUCH_OBJECT" {
		t.Errorf("code = %v, want NO_SUCH_OBJECT", e["code"])
	}
}

// TestAPShowRejectsForeignNote proves a note whose id lives on one host but
// which claims an author on another is refused.
func TestAPShowRejectsForeignNote(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	viewerToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write([]byte(`{
			"@context": "https://www.w3.org/ns/activitystreams",
			"id": "https://evil.example/notes/1",
			"type": "Note",
			"attributedTo": "https://beta.example/users/` + bobID + `",
			"content": "foreign",
			"to": ["https://www.w3.org/ns/activitystreams#Public"]
		}`))
	}))
	t.Cleanup(fake.Close)
	a.srv.SetFederationHTTPClient(&http.Client{
		Transport: &hostRewriteTransport{hosts: map[string]string{
			"evil.example": fake.URL,
			"beta.example": b.ts.URL,
		}},
		Timeout: 10 * timeSecond,
	})
	actorInfo(t, b, bobID)

	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://evil.example/notes/1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign note = %d %v, want 400", resp.StatusCode, out)
	}
}
