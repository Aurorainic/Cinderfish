package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestLockedFollowAcrossInstances drives the full remote path: a locked local
// account receives a signed Follow, stores a request, approves it through the
// API, and the peer records the following after the outbound Accept.
func TestLockedFollowAcrossInstances(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	if resp, out := postWithToken(t, a.ts, "/api/i/update", aliceToken, map[string]any{"isLocked": true}); resp.StatusCode != http.StatusOK {
		t.Fatalf("lock alice: %d %v", resp.StatusCode, out)
	}

	// bob sends a signed Follow to alpha's inbox.
	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	followID := bobActor + "#follows/locked"
	follow := federation.Object{
		"@context": federation.Context,
		"id":       followID,
		"type":     "Follow",
		"actor":    bobActor,
		"object":   "https://alpha.example/users/" + aliceID,
	}
	if resp, out := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}

	// alpha queued a request instead of a following.
	requests := postListWithToken(t, a.ts, "/api/following/requests/list", aliceToken, nil)
	if len(requests) != 1 {
		t.Fatalf("requests = %v, want bob's", requests)
	}
	item, _ := requests[0].(map[string]any)
	follower, _ := item["follower"].(map[string]any)
	remoteBobID, _ := follower["id"].(string)
	if remoteBobID == "" {
		t.Fatalf("request has no follower: %v", item)
	}
	if followers := followListItems(t, a.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID}); len(followers) != 0 {
		t.Errorf("followers before accept = %v, want none", followers)
	}

	// alice approves; alpha must send an Accept and beta must record it.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)
	if resp, out := postWithToken(t, a.ts, "/api/following/requests/accept", aliceToken, map[string]any{"userId": remoteBobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("accept = %d %v", resp.StatusCode, out)
	}

	deadline := time.Now().Add(8 * time.Second)
	for {
		var count int
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM followings f JOIN users u ON u.id = f.followee_id
			 WHERE f.follower_id = $1 AND u.host = 'alpha.example'`, bobID).Scan(&count)
		if err == nil && count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("beta did not record the accepted follow (last error %v)", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// beta sees alice in bob's following list through the API as well.
	list := postListWithToken(t, b.ts, "/api/users/following", "", map[string]any{"userId": bobID})
	if got := followIDs(t, list, "followeeId"); len(got) != 1 {
		t.Errorf("bob following = %v, want alice's remote copy", got)
	}

	// The Accept carried the original Follow id so the peer could match it.
	var delivered int
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM deliver_queue WHERE delivered_at IS NOT NULL
		   AND payload->>'type' = 'Accept' AND inbox_url LIKE 'https://beta.example/%'`).Scan(&delivered); err != nil {
		t.Fatalf("delivery queue: %v", err)
	}
	if delivered == 0 {
		t.Errorf("no delivered Accept row for beta")
	}
}
