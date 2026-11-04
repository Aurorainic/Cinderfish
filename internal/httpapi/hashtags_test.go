package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestHashtagsTrend ranks tags by distinct local users over the window and
// ignores backdated and deleted notes.
func TestHashtagsTrend(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")
	carolToken, _ := f.registerUser(t, "carol")

	// #golang: two users. #cats: one user twice (still one user of reach).
	createNote(t, f.ts, aliceToken, map[string]any{"text": "learning #golang today"})
	createNote(t, f.ts, bobToken, map[string]any{"text": "#golang again"})
	createNote(t, f.ts, carolToken, map[string]any{"text": "#cats one"})
	createNote(t, f.ts, carolToken, map[string]any{"text": "#cats two"})

	// A note outside the window must not count, and a deleted one must not
	// either.
	old := createNote(t, f.ts, aliceToken, map[string]any{"text": "#golang old"})
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE notes SET created_at = now() - interval '30 days' WHERE id = $1`, old); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	removed := createNote(t, f.ts, bobToken, map[string]any{"text": "#golang deleted"})
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE notes SET deleted_at = now() WHERE id = $1`, removed); err != nil {
		t.Fatalf("delete: %v", err)
	}

	trends := postListWithToken(t, f.ts, "/api/hashtags/trend", "", nil)
	if len(trends) != 2 {
		t.Fatalf("trends = %v, want two tags", trends)
	}
	first, _ := trends[0].(map[string]any)
	second, _ := trends[1].(map[string]any)
	if first["tag"] != "golang" || first["usersCount"] != float64(2) {
		t.Errorf("first trend = %v, want golang with 2 users", first)
	}
	if second["tag"] != "cats" || second["usersCount"] != float64(1) {
		t.Errorf("second trend = %v, want cats with 1 user", second)
	}
	chart, _ := first["chart"].([]any)
	if len(chart) != 7 {
		t.Errorf("chart length = %d, want 7", len(chart))
	}
	if last, _ := chart[len(chart)-1].(float64); last != 2 {
		t.Errorf("today's bucket = %v, want 2 users", chart[len(chart)-1])
	}

	// No credential needed, as upstream.
	if resp, _ := post(t, f.ts, "/api/hashtags/trend", map[string]any{}); resp.StatusCode != http.StatusOK {
		t.Errorf("anonymous trend = %d, want 200", resp.StatusCode)
	}
}

// TestHashtagsTrendEmptyState proves a quiet instance answers an empty array
// instead of an error.
func TestHashtagsTrendEmptyState(t *testing.T) {
	ts := newTestServer(t, "open")
	trends := postListWithToken(t, ts, "/api/hashtags/trend", "", nil)
	if len(trends) != 0 {
		t.Errorf("empty instance trends = %v, want none", trends)
	}
}
