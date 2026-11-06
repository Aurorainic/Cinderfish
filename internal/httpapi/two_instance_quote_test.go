package httpapi_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceQuoteReachesOriginalAuthor proves a quote of a remote note
// keeps its text, is delivered to the original author's instance, and is
// stored there as a renote of the author's own note (quoteUrl points at the
// original ActivityPub id, not the local copy).
func TestTwoInstanceQuoteReachesOriginalAuthor(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID

	// alice follows bob so his note arrives as a copy on alpha.
	aliceKeyID, _, _ := actorInfo(t, a, aliceID)
	aliceKP, err := a.srv.FederationLocalKey(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("alpha key: %v", err)
	}
	alicePriv, err := federation.ParsePrivateKeyPEM(aliceKP.PrivatePEM)
	if err != nil {
		t.Fatalf("parse alpha key: %v", err)
	}
	_, _, bobActor := actorInfo(t, b, bobID)
	follow := federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#follows/quote",
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	}
	if resp, out := postSigned(t, b.ts, "/inbox", follow, aliceKeyID, alicePriv); resp.StatusCode != 202 {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)
	b.srv.StartFederationWorkers(ctx)

	bobNoteLocal := createNote(t, b.ts, bobToken, map[string]any{"text": "quote me", "visibility": "public"})

	var copyID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		if err := a.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'beta.example' AND text = 'quote me' AND deleted_at IS NULL`).Scan(&copyID); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bob's note never reached alpha")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// alice quotes it with her own text.
	if resp, out := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "worth reading", "renoteId": copyID,
	}); resp.StatusCode != 200 {
		t.Fatalf("quote = %d %v", resp.StatusCode, out)
	}

	deadline = time.Now().Add(8 * time.Second)
	for {
		var text string
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT text FROM notes WHERE user_host = 'alpha.example' AND renote_id = $1 AND deleted_at IS NULL`,
			bobNoteLocal).Scan(&text)
		if err == nil {
			if !strings.Contains(text, "worth reading") {
				t.Errorf("stored quote text = %q", text)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the quote never reached the original author's instance")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
