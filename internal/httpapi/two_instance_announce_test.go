package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestTwoInstanceAnnounceReachesOriginalAuthor proves a pure renote of a
// remote note is delivered to the original author's instance even when that
// author does not follow the renoter (Misskey's direct recipe).
func TestTwoInstanceAnnounceReachesOriginalAuthor(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID

	// alice follows bob so beta's note reaches alpha as a copy.
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
		"id":       aliceActor + "#follows/announce",
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	}
	if resp, out := postSigned(t, b.ts, "/inbox", follow, aliceKeyID, alicePriv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)
	b.srv.StartFederationWorkers(ctx)

	bobNoteLocal := createNote(t, b.ts, bobToken, map[string]any{"text": "announce me", "visibility": "public"})

	// The copy on alpha.
	var copyID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		err := a.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'beta.example' AND text = 'announce me' AND deleted_at IS NULL`).Scan(&copyID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bob's note never reached alpha")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// alice pure-renotes the copy.
	// No text at all: a text pointer, even empty, would make it a quote.
	if resp, out := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"renoteId": copyID,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("renote = %d %v", resp.StatusCode, out)
	}

	// beta must store the Announce as a renote of bob's own note.
	deadline = time.Now().Add(8 * time.Second)
	for {
		var renoteID string
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT renote_id FROM notes WHERE user_host = 'alpha.example' AND renote_id = $1 AND deleted_at IS NULL`,
			bobNoteLocal).Scan(&renoteID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			rows, _ := a.srv.TestPool().Query(context.Background(),
				`SELECT payload->>'type', inbox_url, attempts, COALESCE(last_error,'') FROM deliver_queue`)
			for rows.Next() {
				var typ, inbox, errText string
				var attempts int
				_ = rows.Scan(&typ, &inbox, &attempts, &errText)
				t.Logf("queued %s -> %s attempts=%d err=%s", typ, inbox, attempts, errText)
			}
			rows.Close()
			t.Fatal("the Announce never reached the original author's instance")
		}
		time.Sleep(200 * time.Millisecond)
	}

	var delivered int
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM deliver_queue WHERE delivered_at IS NOT NULL AND inbox_url LIKE 'https://beta.example/%'`).Scan(&delivered); err != nil {
		t.Fatalf("delivery queue: %v", err)
	}
	if delivered == 0 {
		t.Error("alpha has no delivered queue row for beta")
	}
}
