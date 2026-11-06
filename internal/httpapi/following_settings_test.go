package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestFollowingListAndUpdate covers following/list and following/update: the
// switches are stored per follow and the notification filter reads them.
func TestFollowingListAndUpdate(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")

	if resp, body := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, body)
	}

	list := postListWithToken(t, f.ts, "/api/following/list", aliceToken, nil)
	if len(list) != 1 {
		t.Fatalf("following/list = %v, want one row", list)
	}
	row, _ := list[0].(map[string]any)
	followee, _ := row["followee"].(map[string]any)
	if followee["id"] != bobID {
		t.Errorf("followee = %v", row["followee"])
	}
	if _, has := row["withReplies"]; has {
		t.Errorf("withReplies serialized for its default: %v", row)
	}

	// Turning the switches on.
	if resp, body := postWithToken(t, f.ts, "/api/following/update", aliceToken, map[string]any{
		"userId": bobID, "notify": "normal", "withReplies": false,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("following/update = %d %v", resp.StatusCode, body)
	}
	list = postListWithToken(t, f.ts, "/api/following/list", aliceToken, nil)
	row, _ = list[0].(map[string]any)
	if row["notify"] != "normal" || row["withReplies"] != false {
		t.Errorf("stored switches = %v", row)
	}

	// The notification filter keeps only follows that notify.
	notifying := postListWithToken(t, f.ts, "/api/following/list", aliceToken, map[string]any{"notification": true})
	if len(notifying) != 1 {
		t.Fatalf("notification list = %v, want one", notifying)
	}

	// update-all turns notifications back off everywhere.
	// update-all answers 204 with no body, so it bypasses the JSON helpers.
	updateAllBody := []byte(`{"i":"` + aliceToken + `","notify":"none"}`)
	updateAllResp, err := f.ts.Client().Post(f.ts.URL+"/api/following/update-all", "application/json", bytes.NewReader(updateAllBody))
	if err != nil {
		t.Fatalf("update-all: %v", err)
	}
	updateAllResp.Body.Close()
	if updateAllResp.StatusCode != http.StatusNoContent {
		t.Fatalf("update-all = %d, want 204", updateAllResp.StatusCode)
	}
	notifying = postListWithToken(t, f.ts, "/api/following/list", aliceToken, map[string]any{"notification": true})
	if len(notifying) != 0 {
		t.Errorf("notification list after update-all = %v, want none", notifying)
	}

	// Updating a follow that does not exist is refused.
	resp, body := postWithToken(t, f.ts, "/api/following/update", aliceToken, map[string]any{
		"userId": bobID, "notify": "normal",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-update = %d %v", resp.StatusCode, body)
	}
	_, carolID := f.registerUser(t, "carol")
	resp, body = postWithToken(t, f.ts, "/api/following/update", aliceToken, map[string]any{
		"userId": carolID, "withReplies": true,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("update non-follow = %d %v, want 400", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "NOT_FOLLOWING" {
		t.Errorf("error = %v, want NOT_FOLLOWING", body["error"])
	}
}

// TestFollowingInvalidate covers the forced unfollow: the followee removes a
// follower, which following/delete cannot express.
func TestFollowingInvalidate(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	if resp, body := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, body)
	}
	_, store, _, _ := f.srv.Federation()
	if !store.HasFollowing(t.Context(), bobID, aliceID) {
		t.Fatal("follow not recorded")
	}

	// Alice removes bob as a follower.
	if resp, body := postWithToken(t, f.ts, "/api/following/invalidate", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("invalidate = %d %v", resp.StatusCode, body)
	}
	if store.HasFollowing(t.Context(), bobID, aliceID) {
		t.Error("follow survived invalidate")
	}

	// Removing a non-follower is refused.
	resp, body := postWithToken(t, f.ts, "/api/following/invalidate", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalidate again = %d %v, want 400", resp.StatusCode, body)
	}
}
