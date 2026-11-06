package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestFederationInstances covers federation/instances: hosts are aggregated
// from stored users and notes, ranked by user count, filterable by host, paged,
// and an empty instance answers an empty array. Guests are refused.
func TestFederationInstances(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	// Empty first: nothing has been federated with yet.
	if got := postListWithToken(t, f.ts, "/api/federation/instances", token, map[string]any{}); len(got) != 0 {
		t.Fatalf("instances on an empty instance = %v, want none", got)
	}

	// Two remote users on a.example, one on b.example, plus a note each.
	for _, row := range []struct{ id, user, host string }{
		{"w63ra1", "ra1", "a.example"},
		{"w63ra2", "ra2", "a.example"},
		{"w63rb1", "rb1", "b.example"},
	} {
		if _, err := f.pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri)
VALUES ($1,$2,$2,$3,'https://' || $3 || '/users/' || $2)`, row.id, row.user, row.host); err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}
	for _, row := range []struct{ id, user string }{
		{"w63n1", "w63ra1"}, {"w63n2", "w63ra2"}, {"w63n3", "w63rb1"},
	} {
		if _, err := f.pool.Exec(context.Background(), `
INSERT INTO notes (id, user_id, user_host, visibility, text)
VALUES ($1,$2,(SELECT host FROM users WHERE id=$2),'public','remote note')`, row.id, row.user); err != nil {
			t.Fatalf("insert note %s: %v", row.id, err)
		}
	}

	list := func(params map[string]any) []any {
		return postListWithToken(t, f.ts, "/api/federation/instances", token, params)
	}

	// Ranked by user count, then host.
	all := list(map[string]any{})
	if len(all) != 2 {
		t.Fatalf("instances = %v, want two hosts", all)
	}
	first, _ := all[0].(map[string]any)
	if first["host"] != "a.example" || first["usersCount"] != float64(2) || first["notesCount"] != float64(2) {
		t.Errorf("first instance = %v", first)
	}
	second, _ := all[1].(map[string]any)
	if second["host"] != "b.example" || second["usersCount"] != float64(1) {
		t.Errorf("second instance = %v", second)
	}

	// The host filter keeps one row (case-insensitively).
	filtered := list(map[string]any{"host": "B.EXAMPLE"})
	if len(filtered) != 1 || filtered[0].(map[string]any)["host"] != "b.example" {
		t.Errorf("host filter = %v", filtered)
	}

	// Paging: limit 1 gives the first host, offset 1 the second.
	if page := list(map[string]any{"limit": 1}); len(page) != 1 || page[0].(map[string]any)["host"] != "a.example" {
		t.Errorf("limit page = %v", page)
	}
	if page := list(map[string]any{"limit": 1, "offset": 1}); len(page) != 1 || page[0].(map[string]any)["host"] != "b.example" {
		t.Errorf("offset page = %v", page)
	}

	// Guests cannot read the list.
	if resp, _ := post(t, f.ts, "/api/federation/instances", map[string]any{}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}
}
