package httpapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboundReplyToLocalNote proves a reply from a remote author reaches the
// local note's instance without any follow in that direction: the delivery
// adds the reply target's author, and the inbox resolves the local note id
// from its ActivityPub URL.
func TestInboundReplyToLocalNote(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID

	// bob follows alice so he can see (and have a local copy of) her note.
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
		"id":       bobActor + "#follows/reply",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActor,
	}
	if resp, out := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != 200 && resp.StatusCode != 202 {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)
	b.srv.StartFederationWorkers(ctx)

	aliceNote := createNote(t, a.ts, aliceToken, map[string]any{"text": "reply to me", "visibility": "public"})

	// Wait until beta holds a copy of alice's note.
	var copyID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		if err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'alpha.example' AND text = 'reply to me' AND deleted_at IS NULL`).Scan(&copyID); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("alice's note never reached beta")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// bob replies to the copy. No one on alpha follows bob.
	if resp, out := postWithToken(t, b.ts, "/api/notes/create", bobToken, map[string]any{
		"text": "answering from beta", "replyId": copyID, "visibility": "public",
	}); resp.StatusCode != 200 {
		t.Fatalf("reply = %d %v", resp.StatusCode, out)
	}

	// alpha stores it as a reply to alice's own note.
	deadline = time.Now().Add(8 * time.Second)
	for {
		var replyID string
		err := a.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'beta.example' AND text = 'answering from beta'
			   AND reply_id = $1 AND deleted_at IS NULL`, aliceNote).Scan(&replyID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bob's reply never reached alpha as a reply to alice's note")
		}
		time.Sleep(200 * time.Millisecond)
	}

	notes := postListWithToken(t, a.ts, "/api/i/notifications", aliceToken, nil)
	found := false
	for _, item := range notes {
		n, _ := item.(map[string]any)
		if n["type"] != "reply" {
			continue
		}
		found = true
		// The notifier on alpha is the remote copy of bob, not beta's row.
		if user, _ := n["user"].(map[string]any); user["username"] != "betabob" || user["host"] != "beta.example" {
			t.Errorf("reply notification user = %v, want betabob@beta.example", n["user"])
		}
	}
	if !found {
		t.Errorf("alice got no reply notification: %v", notes)
	}
}
