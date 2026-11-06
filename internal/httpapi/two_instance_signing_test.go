package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceAuthorSigning proves deliveries are signed with the author's
// key rather than the instance's first account: the other instance verifies
// the signature against the actor it names, so a delivery signed by anyone else
// would be rejected and the note would never be stored.
func TestTwoInstanceAuthorSigning(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	// An earlier local administrator exists, so the instance fallback key is
	// not Alice's.
	_, adminID := a.signup(t, "alphaadmin")
	if _, err := a.srv.TestPool().Exec(context.Background(),
		"UPDATE users SET is_admin = TRUE WHERE id = $1", adminID); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID
	_, _, bobActor := actorInfo(t, b, bobID)

	// Bob follows Alice so the note has a remote audience.
	keyID, _, _ := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	follow := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#follows/signing",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActor,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, body)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)

	if resp, created := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "signed by alice", "visibility": "public",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, created)
	}

	// The queue records the key that will sign the delivery.
	deadline := time.Now().Add(8 * time.Second)
	wantKey := "https://alpha.example/users/" + aliceID + "#main-key"
	for {
		var key string
		err := a.srv.TestPool().QueryRow(context.Background(),
			`SELECT COALESCE(key_id, '') FROM deliver_queue ORDER BY id DESC LIMIT 1`).Scan(&key)
		if err == nil && key != "" {
			if key != wantKey {
				t.Fatalf("delivery signed with %q, want the author's key %q", key, wantKey)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no delivery was queued")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// And the peer accepted it, which only happens when the signature matches
	// the actor's published key.
	deadline = time.Now().Add(8 * time.Second)
	for {
		var stored int
		if err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM notes WHERE user_host = 'alpha.example' AND text = 'signed by alice'`).Scan(&stored); err != nil {
			t.Fatalf("count: %v", err)
		}
		if stored == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("beta rejected the delivery signed by the author")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
