package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceNoteDelivery proves the outbound path end to end with real
// HTTP: a follower on the other instance receives a Create, the note is stored
// there, and the author's own instance recorded the outbound activity.
func TestTwoInstanceNoteDelivery(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID
	_, _, bobActor := actorInfo(t, b, bobID)

	// Bob follows Alice over the wire, so Alice has a remote follower.
	keyID, _, _ := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse beta key: %v", err)
	}
	follow := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#follows/w63",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActor,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, body)
	}

	// The worker is what turns the queued Create into an HTTP delivery.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)

	resp, created := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "delivered to beta", "visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, created)
	}

	deadline := time.Now().Add(8 * time.Second)
	for {
		var stored int
		if err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM notes WHERE user_host = 'alpha.example' AND text = 'delivered to beta' AND deleted_at IS NULL`).Scan(&stored); err != nil {
			t.Fatalf("count delivered notes: %v", err)
		}
		if stored == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the note was never delivered to the other instance")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// The delivery queue keeps the finished row, so the fan-out is traceable.
	var delivered int
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM deliver_queue WHERE delivered_at IS NOT NULL AND inbox_url LIKE 'https://beta.example/%'`).Scan(&delivered); err != nil {
		t.Fatalf("delivery queue: %v", err)
	}
	if delivered == 0 {
		t.Error("no delivered queue row for the beta inbox")
	}
}
