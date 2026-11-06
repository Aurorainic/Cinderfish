package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceRemoteNoteNotification proves the follower-notify switch is
// honoured for a note that arrives over federation, not only for local posts:
// beta stores alice's note and creates the "note" notification for bob.
func TestTwoInstanceRemoteNoteNotification(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)
	b.srv.StartFederationWorkers(ctx)

	// bob follows alice with a signed Follow; alpha accepts, which teaches
	// beta its own following row.
	keyID, _, bobActor := actorInfo(t, b, bobID)
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
		"id":       bobActor + "#follows/notify",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActor,
	}
	if resp, out := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}

	var remoteAliceID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM users WHERE host = 'alpha.example' AND username_lower = 'alphaalice'`).Scan(&remoteAliceID)
		if err == nil && remoteAliceID != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("beta never learned alice: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if resp, out := postWithToken(t, b.ts, "/api/following/update", bobToken, map[string]any{
		"userId": remoteAliceID, "notify": "normal",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("following/update = %d %v", resp.StatusCode, out)
	}

	if resp, out := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "notify the remote follower", "visibility": "public",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, out)
	}

	var copyID string
	deadline = time.Now().Add(10 * time.Second)
	for {
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'alpha.example' AND text = 'notify the remote follower' AND deleted_at IS NULL`).Scan(&copyID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("note never reached beta: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	deadline = time.Now().Add(6 * time.Second)
	for {
		var count int
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM notifications WHERE recipient_id = $1 AND type = 'note' AND note_id = $2`,
			bobID, copyID).Scan(&count)
		if err == nil && count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no note notification for bob (count=%d, err=%v)", count, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
