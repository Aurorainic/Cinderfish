package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstancePollDelivery proves a local poll federates as a Question and
// the receiving instance stores it as a poll the UI can render.
func TestTwoInstancePollDelivery(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID

	keyID, _, bobActor := actorInfo(t, b, bobID)
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
		"id":       bobActor + "#follows/poll",
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

	resp, created := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text":       "which colour?",
		"visibility": "public",
		"poll": map[string]any{
			"choices":      []string{"red", "blue"},
			"multiple":     true,
			"expiredAfter": 3600000,
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, created)
	}

	var remoteID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'alpha.example' AND text = 'which colour?' AND deleted_at IS NULL`).Scan(&remoteID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the poll was never delivered to the other instance")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// The receiving instance stores rows in polls, so notes/show reports it.
	_, shown := postWithToken(t, b.ts, "/api/notes/show", bobToken, map[string]any{"noteId": remoteID})
	if shown["hasPoll"] != true {
		t.Fatalf("delivered note has no poll: %v", shown)
	}
	poll, _ := shown["poll"].(map[string]any)
	if poll["multiple"] != true {
		t.Errorf("multiple = %v, want true", poll["multiple"])
	}
	if expires, _ := poll["expiresAt"].(float64); expires <= float64(time.Now().UnixMilli()) {
		t.Errorf("expiresAt = %v, want a future timestamp", poll["expiresAt"])
	}
	choices, _ := poll["choices"].([]any)
	if len(choices) != 2 {
		t.Fatalf("choices = %v, want two", poll["choices"])
	}
	first, _ := choices[0].(map[string]any)
	second, _ := choices[1].(map[string]any)
	if first["text"] != "red" || second["text"] != "blue" {
		t.Errorf("choices = %v/%v, want red/blue", first["text"], second["text"])
	}

	// A local user may vote on the remote poll; the vote stays local.
	resp, voted := postWithToken(t, b.ts, "/api/notes/polls/vote", bobToken, map[string]any{
		"noteId": remoteID, "choice": 1,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vote = %d %v", resp.StatusCode, voted)
	}
	_, shown = postWithToken(t, b.ts, "/api/notes/show", bobToken, map[string]any{"noteId": remoteID})
	poll, _ = shown["poll"].(map[string]any)
	choices, _ = poll["choices"].([]any)
	second, _ = choices[1].(map[string]any)
	if second["votes"] != float64(1) || second["isVoted"] != true {
		t.Errorf("second choice after vote = %v, want one vote and isVoted", second)
	}
}
