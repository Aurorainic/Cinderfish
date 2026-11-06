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

// TestHashtagsShowAndList covers hashtags/show (counts for one tag) and
// hashtags/list (the same aggregation, ranked).
func TestHashtagsShowAndList(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	createNote(t, f.ts, aliceToken, map[string]any{"text": "about #golang"})
	createNote(t, f.ts, bobToken, map[string]any{"text": "more #golang"})
	createNote(t, f.ts, bobToken, map[string]any{"text": "and #cats"})

	resp, show := post(t, f.ts, "/api/hashtags/show", map[string]any{"tag": "#golang"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hashtags/show = %d %v", resp.StatusCode, show)
	}
	if show["tag"] != "golang" {
		t.Errorf("tag = %v, want golang", show["tag"])
	}
	if show["mentionedUsersCount"] != float64(2) {
		t.Errorf("mentionedUsersCount = %v, want 2", show["mentionedUsersCount"])
	}
	if show["mentionedLocalUsersCount"] != float64(2) {
		t.Errorf("mentionedLocalUsersCount = %v, want 2", show["mentionedLocalUsersCount"])
	}
	if show["attachedUsersCount"] != float64(0) {
		t.Errorf("attachedUsersCount = %v, want 0", show["attachedUsersCount"])
	}

	// An unknown tag is a 404 with Misskey's code.
	resp, missing := post(t, f.ts, "/api/hashtags/show", map[string]any{"tag": "nothing-here"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown tag = %d %v, want 404", resp.StatusCode, missing)
	}
	if missing["error"].(map[string]any)["code"] != "NO_SUCH_HASHTAG" {
		t.Errorf("error = %v", missing["error"])
	}

	list := postListWithToken(t, f.ts, "/api/hashtags/list", "", map[string]any{"sort": "+mentionedUsers"})
	if len(list) != 2 {
		t.Fatalf("hashtags/list = %v, want two tags", list)
	}
	first, _ := list[0].(map[string]any)
	if first["tag"] != "golang" {
		t.Errorf("first tag = %v, want golang", first["tag"])
	}
	// Ascending order puts the less used tag first.
	list = postListWithToken(t, f.ts, "/api/hashtags/list", "", map[string]any{"sort": "-mentionedUsers"})
	first, _ = list[0].(map[string]any)
	if first["tag"] != "cats" {
		t.Errorf("ascending first tag = %v, want cats", first["tag"])
	}

	// Attached-only filters have no matches: Cinderfish does not model
	// profile tags.
	list = postListWithToken(t, f.ts, "/api/hashtags/list", "", map[string]any{
		"sort": "+mentionedUsers", "attachedToUserOnly": true,
	})
	if len(list) != 0 {
		t.Errorf("attached filter = %v, want none", list)
	}
}

// TestHashtagsUsersIsEmpty documents the honest answer: the endpoint lists
// profile tags, which Cinderfish does not store.
func TestHashtagsUsersIsEmpty(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	createNote(t, f.ts, aliceToken, map[string]any{"text": "about #golang"})

	list := postListWithToken(t, f.ts, "/api/hashtags/users", "", map[string]any{"tag": "golang", "sort": "+follower"})
	if len(list) != 0 {
		t.Errorf("hashtags/users = %v, want none", list)
	}
}

// TestHashtagsListSortModes covers every sort enum: the mentioned* variants
// order by their own counter, and the attached* variants fall back to the
// matching mentioned* count because profile tags are not modelled.
func TestHashtagsListSortModes(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	createNote(t, f.ts, aliceToken, map[string]any{"text": "#local first"})
	createNote(t, f.ts, bobToken, map[string]any{"text": "#local again"})
	createNote(t, f.ts, aliceToken, map[string]any{"text": "#mixed local"})

	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, host)
VALUES ('hremote1', 'r1', 'r1', 'x', 'remote.example'),
       ('hremote2', 'r2', 'r2', 'x', 'remote.example')`); err != nil {
		t.Fatalf("insert remote users: %v", err)
	}
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO notes (id, user_id, user_host, visibility, text, tags)
VALUES ('hnote1', 'hremote1', 'remote.example', 'public', 'remote only', '["remote"]'::jsonb),
       ('hnote2', 'hremote2', 'remote.example', 'public', 'mixed remote', '["mixed"]'::jsonb)`); err != nil {
		t.Fatalf("insert remote notes: %v", err)
	}

	order := func(sort string) []string {
		t.Helper()
		list := postListWithToken(t, f.ts, "/api/hashtags/list", "", map[string]any{"sort": sort})
		out := make([]string, 0, len(list))
		for _, item := range list {
			m, _ := item.(map[string]any)
			tag, _ := m["tag"].(string)
			out = append(out, tag)
		}
		return out
	}
	assertOrder := func(sort string, want []string) {
		t.Helper()
		got := order(sort)
		if len(got) != len(want) {
			t.Fatalf("%s = %v, want %v", sort, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s = %v, want %v", sort, got, want)
			}
		}
	}

	assertOrder("+mentionedUsers", []string{"local", "mixed", "remote"})
	assertOrder("-mentionedUsers", []string{"remote", "local", "mixed"})
	assertOrder("+mentionedLocalUsers", []string{"local", "mixed", "remote"})
	assertOrder("+mentionedRemoteUsers", []string{"mixed", "remote", "local"})
	assertOrder("-mentionedRemoteUsers", []string{"local", "mixed", "remote"})
	// attached* has no data and falls back to the mentioned counters.
	assertOrder("+attachedUsers", []string{"local", "mixed", "remote"})
	assertOrder("+attachedLocalUsers", []string{"local", "mixed", "remote"})
	assertOrder("+attachedRemoteUsers", []string{"mixed", "remote", "local"})
}
