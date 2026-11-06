package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestChartsUserNotes derives the per-user notes chart from stored rows:
// buckets are UTC days oldest first, with created/deleted counts and the
// normal/reply/renote/withFile split.
func TestChartsUserNotes(t *testing.T) {
	ts, pool := newTestServerWithPool(t, "open")
	ctx := context.Background()
	aliceID, token := createUser(t, ts, "alice")

	normal := createNote(t, ts, token, map[string]any{"text": "normal"})
	reply := createNote(t, ts, token, map[string]any{"text": "reply", "replyId": normal})
	renote := createNote(t, ts, token, map[string]any{"renoteId": normal})
	deleted := createNote(t, ts, token, map[string]any{"text": "gone"})

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	backdate := func(id string, at time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE notes SET created_at = $2 WHERE id = $1`, id, at); err != nil {
			t.Fatalf("backdate %s: %v", id, err)
		}
	}
	backdate(normal, today.Add(-48*time.Hour).Add(12*time.Hour))
	backdate(reply, today.Add(-24*time.Hour).Add(12*time.Hour))
	backdate(renote, today.Add(30*time.Minute))
	backdate(deleted, today.Add(45*time.Minute))

	if resp, body := postWithToken(t, ts, "/api/notes/delete", token, map[string]any{"noteId": deleted}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: status %d body %v", resp.StatusCode, body)
	}

	resp, out := postWithToken(t, ts, "/api/charts/user/notes", token, map[string]any{
		"span": "day", "limit": 5, "userId": aliceID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("charts/user/notes: status %d body %v", resp.StatusCode, out)
	}
	diffs, _ := out["diffs"].(map[string]any)
	ints := func(key string) []int {
		t.Helper()
		raw, _ := out[key].([]any)
		if raw == nil {
			raw, _ = diffs[key].([]any)
		}
		got := make([]int, len(raw))
		for i, v := range raw {
			got[i] = int(v.(float64))
		}
		return got
	}
	normalSeries := ints("normal")
	replySeries := ints("reply")
	renoteSeries := ints("renote")
	withFileSeries := ints("withFile")
	inc := ints("inc")
	dec := ints("dec")
	total := ints("total")

	for name, series := range map[string][]int{
		"normal": normalSeries, "reply": replySeries, "renote": renoteSeries,
		"withFile": withFileSeries, "inc": inc, "dec": dec, "total": total,
	} {
		if len(series) != 5 {
			t.Fatalf("%s has %d buckets, want 5", name, len(series))
		}
	}
	// Today's normal note is the one deleted below: creation counts in the
	// diffs, deletion counts in dec, the same way Misskey separates them.
	if normalSeries[2] != 1 || normalSeries[3] != 0 || normalSeries[4] != 1 {
		t.Errorf("normal = %v, want one two days back and one today", normalSeries)
	}
	if replySeries[3] != 1 {
		t.Errorf("reply = %v, want one note yesterday", replySeries)
	}
	if renoteSeries[4] != 1 {
		t.Errorf("renote = %v, want one note today", renoteSeries)
	}
	if withFileSeries[4] != 0 {
		t.Errorf("withFile = %v, want zeros without uploads", withFileSeries)
	}
	// Today: a renote and a now-deleted note were created, the latter counted
	// in dec.
	if inc[4] != 2 || dec[4] != 1 {
		t.Errorf("today inc/dec = %d/%d, want 2/1", inc[4], dec[4])
	}
	// Total is cumulative and subtracts the deletion.
	if total[2] != 1 || total[3] != 2 || total[4] != 3 {
		t.Errorf("total = %v, want [.., 1, 2, 3, 3]", total)
	}

	// The hour span returns one bucket per hour.
	_, hourly := postWithToken(t, ts, "/api/charts/user/notes", token, map[string]any{
		"span": "hour", "limit": 3, "userId": aliceID,
	})
	hourlyDiffs, _ := hourly["diffs"].(map[string]any)
	if got := hourlyDiffs["normal"].([]any); len(got) != 3 {
		t.Fatalf("hourly normal has %d buckets, want 3", len(got))
	}

	// Invalid parameters are rejected.
	if resp, _ := postWithToken(t, ts, "/api/charts/user/notes", token, map[string]any{
		"span": "week", "userId": aliceID,
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("span=week status = %d, want 400", resp.StatusCode)
	}
	if resp, _ := postWithToken(t, ts, "/api/charts/user/notes", token, map[string]any{
		"span": "day",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing userId status = %d, want 400", resp.StatusCode)
	}
}
