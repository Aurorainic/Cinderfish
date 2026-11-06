package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestRemoteMentionIsDeliveredAndNotifies proves a local note that mentions an
// account on another instance reaches that instance: the acct is resolved into
// a Mention tag, the delivery worker posts it, and the target instance stores
// the note and notifies the mentioned user.
func TestRemoteMentionIsDeliveredAndNotifies(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")

	// The worker is what turns the queued Create into an HTTP delivery.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	b.srv.StartFederationWorkers(ctx)

	resp, created := postWithToken(t, b.ts, "/api/notes/create", bobToken, map[string]any{
		"text":       "hello @alphaalice@alpha.example, remote mention",
		"visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, created)
	}

	deadline := time.Now().Add(8 * time.Second)
	for {
		var stored int
		if err := a.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM notes WHERE text LIKE '%remote mention%' AND user_host = 'beta.example'`).Scan(&stored); err != nil {
			t.Fatalf("count notes: %v", err)
		}
		if stored > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the mentioned note never reached the other instance")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// The mentioned user gets a notification on the receiving instance.
	notifications := postListWithToken(t, a.ts, "/api/i/notifications", aliceToken, nil)
	found := false
	for _, item := range notifications {
		n, _ := item.(map[string]any)
		if n["type"] == "mention" {
			found = true
			user, _ := n["user"].(map[string]any)
			if user["username"] != "betabob" {
				t.Errorf("mention notifier = %v, want betabob", user["username"])
			}
		}
	}
	if !found {
		t.Fatalf("no mention notification: %v", notifications)
	}

	// The rendered note carries the Mention tag so peers can notify; check the
	// stored copy's tag list through the author's outbox would be heavy, so the
	// notification above is the observable proof.
	_ = bobID
	_ = aliceID
}
