package httpapi_test

import (
	"net/http"
	"testing"
)

// TestReactionCreateRequiresVisibility proves notes/reactions/create refuses a
// note the caller cannot see, matching the reaction list's rule.
func TestReactionCreateRequiresVisibility(t *testing.T) {
	f := newFederationServer(t, "public")
	ts := f.ts
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")

	followersNote := createNote(t, ts, aliceToken, map[string]any{"text": "followers only", "visibility": "followers"})
	publicNote := createNote(t, ts, aliceToken, map[string]any{"text": "public", "visibility": "public"})
	specifiedNote := createNote(t, ts, aliceToken, map[string]any{
		"text": "just for bob", "visibility": "specified", "visibleUserIds": []string{bobID},
	})

	react := func(token, noteID string) (int, map[string]any) {
		resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
			"i": token, "noteId": noteID, "reaction": "👍",
		})
		return resp.StatusCode, out
	}

	// A stranger cannot react to followers-only or specified notes.
	if status, out := react(bobToken, followersNote); status != http.StatusNotFound {
		t.Errorf("stranger on followers note = %d %v, want 404", status, out)
	}
	if status, _ := react(carolToken, specifiedNote); status != http.StatusNotFound {
		t.Errorf("stranger on specified note = %d, want 404", status)
	}

	// A follower can.
	if resp, out := postWithToken(t, ts, "/api/following/create", carolToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, out)
	}
	if status, out := react(carolToken, followersNote); status != http.StatusOK {
		t.Errorf("follower on followers note = %d %v, want 200", status, out)
	}

	// The named user can, and the author can always.
	if status, out := react(bobToken, specifiedNote); status != http.StatusOK {
		t.Errorf("named user on specified note = %d %v, want 200", status, out)
	}
	if status, out := react(aliceToken, followersNote); status != http.StatusOK {
		t.Errorf("author on own followers note = %d %v, want 200", status, out)
	}

	// Public notes stay open to anyone signed in.
	if status, out := react(bobToken, publicNote); status != http.StatusOK {
		t.Errorf("public note = %d %v, want 200", status, out)
	}
}
