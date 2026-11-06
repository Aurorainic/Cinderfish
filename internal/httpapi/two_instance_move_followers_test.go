package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboundMoveMigratesLocalFollowers proves a remote Move moves the local
// users who followed the old account over to the new one: the old following
// disappears, a request is queued for the remote target, and the follower
// gets a Follow delivered.
func TestInboundMoveMigratesLocalFollowers(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	daveToken, daveID := b.signup(t, "betadave")
	bobActor := "https://beta.example/users/" + bobID
	daveActor := "https://beta.example/users/" + daveID

	// Alpha learns both remote accounts, so the following row and the
	// migration use remote rows (host set), as they would in production.
	for _, uri := range []string{bobActor, daveActor} {
		if resp, out := postWithToken(t, a.ts, "/api/ap/show", aliceToken, map[string]any{"uri": uri}); resp.StatusCode != http.StatusOK {
			t.Fatalf("ap/show %s = %d %v", uri, resp.StatusCode, out)
		}
	}
	var remoteBobID, remoteDaveID string
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT id FROM users WHERE host = 'beta.example' AND username_lower = 'betabob'`).Scan(&remoteBobID); err != nil {
		t.Fatalf("remote bob: %v", err)
	}
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT id FROM users WHERE host = 'beta.example' AND username_lower = 'betadave'`).Scan(&remoteDaveID); err != nil {
		t.Fatalf("remote dave: %v", err)
	}

	// Alice followed the old account locally.
	if _, err := a.srv.TestPool().Exec(context.Background(),
		`INSERT INTO followings (id, follower_id, followee_id) VALUES ('mv-local', $1, $2) ON CONFLICT DO NOTHING`,
		aliceID, remoteBobID); err != nil {
		t.Fatalf("seed following: %v", err)
	}

	// Dave, the move target, lists bob as an alias.
	if resp, out := postWithToken(t, b.ts, "/api/i/known-as", daveToken, map[string]any{
		"alsoKnownAs": []string{"@betabob@beta.example"},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("beta known-as = %d %v", resp.StatusCode, out)
	}

	keyID, _, bobActorRef := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	move := federation.Object{
		"@context": federation.Context,
		"id":       bobActorRef + "#moves/followers",
		"type":     "Move",
		"actor":    bobActorRef,
		"object":   bobActorRef,
		"target":   daveActor,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", move, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Move = %d %s", resp.StatusCode, body)
	}

	// The old following is gone.
	var count int
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM followings WHERE follower_id = $1 AND followee_id = $2`, aliceID, remoteBobID).Scan(&count); err != nil {
		t.Fatalf("old following: %v", err)
	}
	if count != 0 {
		t.Errorf("old following still present")
	}

	// A request to the new remote account is pending and a Follow is queued.
	sent := postListWithToken(t, a.ts, "/api/following/requests/sent", aliceToken, nil)
	if len(sent) != 1 {
		t.Fatalf("sent requests after move = %v, want the new follow", sent)
	}
	if item, _ := sent[0].(map[string]any); item["followee"] == nil {
		t.Fatalf("sent request has no followee: %v", item)
	}
	var queued int
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM deliver_queue WHERE payload->>'type' = 'Follow' AND inbox_url LIKE 'https://beta.example/%'`).Scan(&queued); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if queued == 0 {
		t.Errorf("no Follow queued to the move target")
	}
}
