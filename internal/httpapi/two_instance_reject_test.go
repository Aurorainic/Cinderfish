package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboundRejectClearsPendingRequest proves a remote Reject withdraws the
// follow request we sent: requests/sent empties and no following appears.
func TestInboundRejectClearsPendingRequest(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID
	_, _, bobActor := actorInfo(t, b, bobID)

	// Alpha resolves bob's actor, which creates its remote copy of the
	// account (beta's own row is local and has no uri).
	if resp, out := postWithToken(t, a.ts, "/api/ap/show", aliceToken, map[string]any{"uri": bobActor}); resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show = %d %v", resp.StatusCode, out)
	}
	var remoteBobID string
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT id FROM users WHERE host = 'beta.example' AND username_lower = 'betabob'`).Scan(&remoteBobID); err != nil {
		t.Fatalf("remote bob: %v", err)
	}
	if remoteBobID == "" || remoteBobID == bobID {
		t.Fatalf("remote bob id = %q, want a separate remote row", remoteBobID)
	}

	// A follow we sent is waiting at the peer.
	if _, err := a.srv.TestPool().Exec(context.Background(), `
INSERT INTO follow_requests (id, follower_id, followee_id, uri) VALUES ('fr-reject', $1, $2, $3)
`, aliceID, remoteBobID, aliceActor+"#follows/"+remoteBobID); err != nil {
		t.Fatalf("insert request: %v", err)
	}
	sent := postListWithToken(t, a.ts, "/api/following/requests/sent", aliceToken, nil)
	if len(sent) != 1 {
		t.Fatalf("sent requests = %v, want one", sent)
	}

	keyID, _, _ := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	reject := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#rejects/1",
		"type":     "Reject",
		"actor":    bobActor,
		"object": federation.Object{
			"id":     aliceActor + "#follows/" + remoteBobID,
			"type":   "Follow",
			"actor":  aliceActor,
			"object": bobActor,
		},
	}
	if resp, body := postSigned(t, a.ts, "/inbox", reject, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Reject = %d %s", resp.StatusCode, body)
	}

	if left := postListWithToken(t, a.ts, "/api/following/requests/sent", aliceToken, nil); len(left) != 0 {
		t.Errorf("sent requests after Reject = %v, want none", left)
	}
	following := followListItems(t, a.ts, "/api/users/following", aliceToken, map[string]any{"userId": aliceID})
	if len(following) != 0 {
		t.Errorf("following after Reject = %v, want none", following)
	}
}
