package httpapi_test

import (
	"net/http"
	"testing"
)

// notificationOfType returns the first notification whose type matches.
func notificationOfType(items []any, kind string) map[string]any {
	for _, item := range items {
		n, _ := item.(map[string]any)
		if n["type"] == kind {
			return n
		}
	}
	return nil
}

// TestLockedFollowRequestFlow drives the whole locked-account path through
// following/create rather than inserting the request row directly: the follow
// queues a request, the target is notified, accept turns it into a following,
// and unlocking makes later follows immediate again.
func TestLockedFollowRequestFlow(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")
	daveToken, _ := f.registerUser(t, "dave")

	// alice locks the account.
	if resp, out := postWithToken(t, f.ts, "/api/i/update", aliceToken, map[string]any{"isLocked": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("lock alice: %d %v", resp.StatusCode, out)
	}
	if _, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": aliceID}); show["isLocked"] != true {
		t.Errorf("alice isLocked = %v, want true", show["isLocked"])
	}

	// bob's follow does not create a following; it queues a request.
	if resp, out := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("bob follow: %d %v", resp.StatusCode, out)
	}
	if followers := followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID}); len(followers) != 0 {
		t.Errorf("alice followers after locked follow = %v, want none", followers)
	}
	if following := followListItems(t, f.ts, "/api/users/following", bobToken, map[string]any{"userId": bobID}); len(following) != 0 {
		t.Errorf("bob following after locked follow = %v, want none", following)
	}
	if incoming := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, nil); len(incoming) != 1 {
		t.Fatalf("alice requests = %v, want bob's", incoming)
	}
	if sent := postListWithToken(t, f.ts, "/api/following/requests/sent", bobToken, nil); len(sent) != 1 {
		t.Errorf("bob sent requests = %v, want one", sent)
	}

	// A second follow while the request waits is refused, as upstream.
	resp, errBody := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("second follow = %d %v, want 400", resp.StatusCode, errBody)
	}
	if e, _ := errBody["error"].(map[string]any); e["code"] != "ALREADY_FOLLOWING" {
		t.Errorf("second follow code = %v, want ALREADY_FOLLOWING", e["code"])
	}

	// alice is told a request is waiting, with the requester attached.
	aliceNotes := postListWithToken(t, f.ts, "/api/i/notifications", aliceToken, nil)
	request := notificationOfType(aliceNotes, "receiveFollowRequest")
	if request == nil {
		t.Fatalf("alice notifications = %v, want receiveFollowRequest", aliceNotes)
	}
	if user, _ := request["user"].(map[string]any); user["id"] != bobID {
		t.Errorf("request notifier = %v, want bob", request["user"])
	}
	if aliceNotes := postListWithToken(t, f.ts, "/api/i/notifications", aliceToken, nil); notificationOfType(aliceNotes, "follow") != nil {
		t.Errorf("locked queue produced a follow notification as well")
	}

	// A local direct follow still notifies the followee.
	if resp, out := postWithToken(t, f.ts, "/api/following/create", carolToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("carol follows bob: %d %v", resp.StatusCode, out)
	}
	bobNotes := postListWithToken(t, f.ts, "/api/i/notifications", bobToken, nil)
	if follow := notificationOfType(bobNotes, "follow"); follow == nil {
		t.Errorf("bob notifications = %v, want a follow notification", bobNotes)
	}

	// Accept converts the queued request into a following.
	if resp, out := postWithToken(t, f.ts, "/api/following/requests/accept", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("accept bob: %d %v", resp.StatusCode, out)
	}
	followers := followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID})
	if len(followers) != 1 || followers[0]["followerId"] != bobID {
		t.Errorf("alice followers after accept = %v, want bob", followers)
	}
	if left := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, nil); len(left) != 0 {
		t.Errorf("requests after accept = %v, want none", left)
	}

	// Unlocking makes new follows immediate again: dave gets no request row.
	if resp, out := postWithToken(t, f.ts, "/api/i/update", aliceToken, map[string]any{"isLocked": false}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unlock alice: %d %v", resp.StatusCode, out)
	}
	if resp, out := postWithToken(t, f.ts, "/api/following/create", daveToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("dave follows unlocked alice: %d %v", resp.StatusCode, out)
	}
	if left := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, nil); len(left) != 0 {
		t.Errorf("requests after unlock = %v, want none", left)
	}
	followers = followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID})
	if len(followers) != 2 {
		t.Errorf("alice followers after unlock = %v, want bob and dave", followers)
	}
	if got := notificationOfType(postListWithToken(t, f.ts, "/api/i/notifications", aliceToken, nil), "follow"); got == nil {
		t.Errorf("alice got no follow notification for dave")
	}
}
