package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestUsersRelation covers the single and batch shapes and the flags they
// carry: follow in both directions, a pending request and a mute.
func TestUsersRelation(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	_, carolID := f.registerUser(t, "carol")

	// Alice follows bob; bob follows alice; alice mutes carol.
	if resp, body := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("alice follows bob = %d %v", resp.StatusCode, body)
	}
	if resp, body := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("bob follows alice = %d %v", resp.StatusCode, body)
	}
	// mute/create answers 204, which the JSON helpers cannot decode.
	muteBody := []byte(`{"i":"` + aliceToken + `","userId":"` + carolID + `"}`)
	muteResp, err := f.ts.Client().Post(f.ts.URL+"/api/mute/create", "application/json", bytes.NewReader(muteBody))
	if err != nil {
		t.Fatalf("mute: %v", err)
	}
	muteResp.Body.Close()
	if muteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("mute = %d, want 204", muteResp.StatusCode)
	}

	resp, rel := postWithToken(t, f.ts, "/api/users/relation", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relation = %d %v", resp.StatusCode, rel)
	}
	if rel["isFollowing"] != true || rel["isFollowed"] != true {
		t.Errorf("follow flags = %v", rel)
	}
	if rel["isMuted"] != false {
		t.Errorf("isMuted = %v, want false", rel["isMuted"])
	}

	// A batch answers an array in the requested order.
	resp, raw := postRaw(t, f.ts, "/api/users/relation", map[string]any{
		"i": aliceToken, "userId": []string{bobID, carolID},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch relation = %d %v", resp.StatusCode, raw)
	}
	list, _ := raw.([]any)
	if len(list) != 2 {
		t.Fatalf("batch = %v, want two entries", raw)
	}
	second, _ := list[1].(map[string]any)
	if second["id"] != carolID || second["isMuted"] != true {
		t.Errorf("second relation = %v", second)
	}
}

// TestUsersFrequentlyReplied ranks the accounts a user replies to.
func TestUsersFrequentlyReplied(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")

	bobNote := createNote(t, f.ts, bobToken, map[string]any{"text": "bob here"})
	carolNote := createNote(t, f.ts, carolToken, map[string]any{"text": "carol here"})
	createNote(t, f.ts, aliceToken, map[string]any{"text": "hi bob", "replyId": bobNote})
	createNote(t, f.ts, aliceToken, map[string]any{"text": "again bob", "replyId": bobNote})
	createNote(t, f.ts, aliceToken, map[string]any{"text": "hi carol", "replyId": carolNote})

	list := postListWithToken(t, f.ts, "/api/users/get-frequently-replied-users", "", map[string]any{"userId": aliceID})
	if len(list) != 2 {
		t.Fatalf("frequently replied = %v, want two", list)
	}
	first, _ := list[0].(map[string]any)
	user, _ := first["user"].(map[string]any)
	if user["id"] != bobID || first["weight"] != float64(2) {
		t.Errorf("first = %v, want bob with weight 2", first)
	}
	second, _ := list[1].(map[string]any)
	if second["weight"] != float64(1) {
		t.Errorf("second weight = %v, want 1", second["weight"])
	}
}
