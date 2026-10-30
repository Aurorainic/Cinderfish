package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// relationFlags reads the four UserDetailed follow-relation fields.
func relationFlags(t *testing.T, body map[string]any) (isFollowing, isFollowed, fromYou, toYou bool) {
	t.Helper()
	for _, key := range []string{
		"isFollowing", "isFollowed",
		"hasPendingFollowRequestFromYou", "hasPendingFollowRequestToYou",
	} {
		if _, ok := body[key].(bool); !ok {
			t.Fatalf("%s = %#v (%T), want a boolean; body %v", key, body[key], body[key], body)
		}
	}
	return body["isFollowing"].(bool), body["isFollowed"].(bool),
		body["hasPendingFollowRequestFromYou"].(bool), body["hasPendingFollowRequestToYou"].(bool)
}

func TestUsersShowRelationFlagsAnonymous(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, body := post(t, f.ts, "/api/users/show", map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show: status %d, body %v", resp.StatusCode, body)
	}
	isFollowing, isFollowed, fromYou, toYou := relationFlags(t, body)
	if isFollowing || isFollowed || fromYou || toYou {
		t.Errorf("anonymous flags = %v/%v/%v/%v, want all false", isFollowing, isFollowed, fromYou, toYou)
	}
}

// A bad token must not turn the optional authentication into a 401.
func TestUsersShowRelationFlagsBadToken(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, body := postWithToken(t, f.ts, "/api/users/show", "not-a-real-token", map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show with bad token: status %d, body %v", resp.StatusCode, body)
	}
	isFollowing, isFollowed, fromYou, toYou := relationFlags(t, body)
	if isFollowing || isFollowed || fromYou || toYou {
		t.Errorf("bad-token flags = %v/%v/%v/%v, want all false", isFollowing, isFollowed, fromYou, toYou)
	}
}

func TestUsersShowRelationFlagsBetweenUsers(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	// alice follows bob; nothing points the other way yet.
	resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: status %d, body %v", resp.StatusCode, out)
	}

	resp, out = postWithToken(t, f.ts, "/api/users/show", aliceToken, map[string]any{"username": "bob"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show bob: status %d, body %v", resp.StatusCode, out)
	}
	isFollowing, isFollowed, fromYou, toYou := relationFlags(t, out)
	if !isFollowing {
		t.Errorf("isFollowing = false, want true (alice follows bob)")
	}
	if isFollowed || fromYou || toYou {
		t.Errorf("isFollowed/fromYou/toYou = %v/%v/%v, want all false", isFollowed, fromYou, toYou)
	}

	// The reverse direction: bob viewing alice sees only isFollowed.
	resp, out = postWithToken(t, f.ts, "/api/users/show", bobToken, map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show alice: status %d, body %v", resp.StatusCode, out)
	}
	isFollowing, isFollowed, fromYou, toYou = relationFlags(t, out)
	if !isFollowed {
		t.Errorf("isFollowed = false, want true (bob is followed by alice)")
	}
	if isFollowing || fromYou || toYou {
		t.Errorf("isFollowing/fromYou/toYou = %v/%v/%v, want all false", isFollowing, fromYou, toYou)
	}

	// Self-view never reports a relationship.
	resp, out = postWithToken(t, f.ts, "/api/users/show", aliceToken, map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show self: status %d, body %v", resp.StatusCode, out)
	}
	isFollowing, isFollowed, fromYou, toYou = relationFlags(t, out)
	if isFollowing || isFollowed || fromYou || toYou {
		t.Errorf("self flags = %v/%v/%v/%v, want all false", isFollowing, isFollowed, fromYou, toYou)
	}
}

// Pending follow requests have no API to create them yet, so the rows are
// inserted directly; the pool is exposed for exactly this reason.
func TestUsersShowPendingFollowRequestFlags(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	if _, err := f.srv.TestPool().Exec(context.Background(), `
INSERT INTO follow_requests (id, follower_id, followee_id) VALUES ('fr-alice-bob', $1, $2)
	`, aliceID, bobID); err != nil {
		t.Fatalf("insert follow request: %v", err)
	}

	// alice asks to follow bob.
	resp, out := postWithToken(t, f.ts, "/api/users/show", aliceToken, map[string]any{"username": "bob"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show bob: status %d, body %v", resp.StatusCode, out)
	}
	_, _, fromYou, toYou := relationFlags(t, out)
	if !fromYou {
		t.Errorf("hasPendingFollowRequestFromYou = false, want true")
	}
	if toYou {
		t.Errorf("hasPendingFollowRequestToYou = true, want false")
	}

	// bob sees the request waiting on him.
	resp, out = postWithToken(t, f.ts, "/api/users/show", bobToken, map[string]any{"username": "alice"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show alice: status %d, body %v", resp.StatusCode, out)
	}
	_, _, fromYou, toYou = relationFlags(t, out)
	if !toYou {
		t.Errorf("hasPendingFollowRequestToYou = false, want true")
	}
	if fromYou {
		t.Errorf("hasPendingFollowRequestFromYou = true, want false")
	}
}
