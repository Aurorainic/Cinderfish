package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestNotesPollsVote pins the vote endpoint: a first vote counts, a second in
// a single-choice poll is ALREADY_VOTED, a multiple-choice poll takes one vote
// per choice, and a non-poll note is NO_POLL.
func TestNotesPollsVote(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")

	single := createNote(t, f.ts, aliceToken, map[string]any{
		"text": "pick one",
		"poll": map[string]any{"choices": []string{"a", "b"}, "multiple": false},
	})
	multiple := createNote(t, f.ts, aliceToken, map[string]any{
		"text": "pick many",
		"poll": map[string]any{"choices": []string{"x", "y", "z"}, "multiple": true},
	})
	plain := createNote(t, f.ts, aliceToken, map[string]any{"text": "no poll here"})

	if resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": single, "choice": 0}); resp.StatusCode != http.StatusOK {
		t.Fatalf("first vote: %d %v", resp.StatusCode, out)
	}
	_, shown := post(t, f.ts, "/api/notes/show", map[string]any{"noteId": single})
	poll, _ := shown["poll"].(map[string]any)
	choices, _ := poll["choices"].([]any)
	first, _ := choices[0].(map[string]any)
	if first["votes"] != float64(1) {
		t.Errorf("choice 0 votes = %v, want 1", first["votes"])
	}
	if poll["votesCount"] != float64(1) {
		t.Errorf("votesCount = %v, want 1", poll["votesCount"])
	}

	resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": single, "choice": 1})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("second vote: %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "ALREADY_VOTED" {
		t.Errorf("second vote code = %v", e["code"])
	}

	// A multiple-choice poll accepts distinct choices but not the same twice.
	if resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": multiple, "choice": 0}); resp.StatusCode != http.StatusOK {
		t.Fatalf("multiple first: %d %v", resp.StatusCode, out)
	}
	if resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": multiple, "choice": 2}); resp.StatusCode != http.StatusOK {
		t.Fatalf("multiple second: %d %v", resp.StatusCode, out)
	}
	if resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": multiple, "choice": 2}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("multiple repeat: %d %v, want 400", resp.StatusCode, out)
	}
	_, shown = post(t, f.ts, "/api/notes/show", map[string]any{"noteId": multiple, "i": bobToken})
	poll, _ = shown["poll"].(map[string]any)
	choices, _ = poll["choices"].([]any)
	zero, _ := choices[0].(map[string]any)
	two, _ := choices[2].(map[string]any)
	if zero["isVoted"] != true || two["isVoted"] != true {
		t.Errorf("isVoted flags = %v / %v, want both true", zero["isVoted"], two["isVoted"])
	}

	// A note without a poll answers NO_POLL.
	resp, out = post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": carolToken, "noteId": plain, "choice": 0})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("vote on plain note: %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "NO_POLL" {
		t.Errorf("plain note code = %v", e["code"])
	}

	// Out-of-range choices are client errors, and an expired poll refuses.
	if resp, _ := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": carolToken, "noteId": single, "choice": 9}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("out of range = %d, want 400", resp.StatusCode)
	}
	expired := createNote(t, f.ts, aliceToken, map[string]any{
		"text": "expired",
		"poll": map[string]any{"choices": []string{"a", "b"}, "expiredAfter": 1},
	})
	// Backdate the poll so the expiry check has something to catch.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE polls SET expires_at = now() - interval '1 hour' WHERE id = $1`, expired); err != nil {
		t.Fatalf("expire poll: %v", err)
	}
	resp, out = post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": carolToken, "noteId": expired, "choice": 0})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expired vote: %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "ALREADY_EXPIRED" {
		t.Errorf("expired code = %v", e["code"])
	}

	// Auth and unknown notes.
	if resp, _ := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"noteId": single, "choice": 0}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous vote = %d, want 401", resp.StatusCode)
	}
	if resp, out := post(t, f.ts, "/api/notes/polls/vote", map[string]any{"i": bobToken, "noteId": "00000000000000000000", "choice": 0}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown note = %d %v, want 404", resp.StatusCode, out)
	} else if e, _ := out["error"].(map[string]any); e["id"] != "ecafbd2e-c283-4d6d-aecb-1a0a33b75396" {
		t.Errorf("unknown note error id = %v", e["id"])
	}
}
