package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestNotifyFollowNotifications proves following/update's notify switch is
// consumed: a "normal" follow turns every note of that account into a "note"
// notification, and a plain follow does not.
func TestNotifyFollowNotifications(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	carolToken, _ := f.registerUser(t, "carol")
	bobToken, bobID := f.registerUser(t, "bob")

	for _, token := range []string{aliceToken, carolToken} {
		if resp, body := postWithToken(t, f.ts, "/api/following/create", token, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow = %d %v", resp.StatusCode, body)
		}
	}
	if resp, body := postWithToken(t, f.ts, "/api/following/update", aliceToken, map[string]any{
		"userId": bobID, "notify": "normal",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("following/update = %d %v", resp.StatusCode, body)
	}

	noteID := createNote(t, f.ts, bobToken, map[string]any{"text": "a note worth notifying"})

	found := false
	for _, item := range postListWithToken(t, f.ts, "/api/i/notifications", aliceToken, nil) {
		n, _ := item.(map[string]any)
		if n["type"] != "note" {
			continue
		}
		found = true
		if n["noteId"] != noteID {
			t.Errorf("notification noteId = %v, want %s", n["noteId"], noteID)
		}
	}
	if !found {
		t.Error("alice got no note notification although the follow asks for them")
	}

	for _, item := range postListWithToken(t, f.ts, "/api/i/notifications", carolToken, nil) {
		if n, _ := item.(map[string]any); n["type"] == "note" {
			t.Errorf("carol got a note notification without asking: %v", n)
		}
	}
}

// TestLoadFollowsCarriesWithReplies proves the streaming viewer snapshot reads
// the per-follow switch, which is what the homeTimeline channel filters on.
func TestLoadFollowsCarriesWithReplies(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")

	if resp, body := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, body)
	}
	follows, err := f.srv.LoadFollows(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("LoadFollows: %v", err)
	}
	if !follows[bobID].WithReplies {
		t.Errorf("default WithReplies = false, want true")
	}

	if resp, body := postWithToken(t, f.ts, "/api/following/update", aliceToken, map[string]any{
		"userId": bobID, "withReplies": false,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("following/update = %d %v", resp.StatusCode, body)
	}
	follows, err = f.srv.LoadFollows(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("LoadFollows after update: %v", err)
	}
	if follows[bobID].WithReplies {
		t.Errorf("WithReplies = true after turning it off")
	}
}
