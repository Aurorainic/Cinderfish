package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceNoteUpdate proves an edit reaches the other instance: the
// stored copy takes the edited Markdown, verbatim, rather than rendered HTML.
func TestTwoInstanceNoteUpdate(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID
	_, _, bobActor := actorInfo(t, b, bobID)

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
		"id":       bobActor + "#follows/update",
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
		"text": "before the edit", "visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create = %d %v", resp.StatusCode, created)
	}
	obj, _ := created["createdNote"].(map[string]any)
	noteID, _ := obj["id"].(string)

	// Wait for the original note to arrive.
	deadline := time.Now().Add(8 * time.Second)
	for {
		var stored int
		if err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM notes WHERE user_host = 'alpha.example' AND text = 'before the edit'`).Scan(&stored); err != nil {
			t.Fatalf("count: %v", err)
		}
		if stored == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the original note never arrived")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Edit it, with Markdown that would look different after HTML rendering.
	const edited = "after **bold** edit\nsecond line"
	if resp, body := postWithToken(t, a.ts, "/api/notes/edit", aliceToken, map[string]any{
		"editId": noteID, "text": edited, "visibility": "public",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("edit = %d %v", resp.StatusCode, body)
	}

	deadline = time.Now().Add(8 * time.Second)
	for {
		var text string
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT text FROM notes WHERE user_host = 'alpha.example' AND ap_id IS NOT NULL ORDER BY created_at DESC LIMIT 1`).Scan(&text)
		if err == nil && text == edited {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the edit never arrived; stored text = %q, want %q", text, edited)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
