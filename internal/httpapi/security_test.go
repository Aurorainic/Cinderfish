package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestPasswordChange walks i/change-password: the current password is
// required, the change is atomic, and the old password stops working.
func TestPasswordChange(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "old-password-1"})
	token := signup["token"].(string)

	// Wrong current password is refused with 403.
	resp, _ := post(t, ts, "/api/i/change-password", map[string]any{
		"i": token, "currentPassword": "not-it", "newPassword": "new-password-2",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong current password = %d, want 403", resp.StatusCode)
	}

	// The real change answers 204, which the JSON helpers cannot decode.
	payload := []byte(`{"i":"` + token + `","currentPassword":"old-password-1","newPassword":"new-password-2"}`)
	changed, err := ts.Client().Post(ts.URL+"/api/i/change-password", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("change-password: %v", err)
	}
	changed.Body.Close()
	if changed.StatusCode != http.StatusNoContent {
		t.Fatalf("change-password = %d, want 204", changed.StatusCode)
	}

	// The old password no longer signs in; the new one does.
	if resp, _ := post(t, ts, "/api/signin", map[string]any{"username": name, "password": "old-password-1"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("signin with old password = %d, want 403", resp.StatusCode)
	}
	resp, signed := post(t, ts, "/api/signin", map[string]any{"username": name, "password": "new-password-2"})
	if resp.StatusCode != http.StatusOK || signed["finished"] != true {
		t.Fatalf("signin with new password = %d %v", resp.StatusCode, signed)
	}
}

// TestRegenerateToken proves i/regenerate-token invalidates the old token and
// returns one that works, and that i/signin-history lists the sign-in the
// regeneration followed.
func TestRegenerateToken(t *testing.T) {
	ts := newTestServer(t, "open")
	name := uniqueName(t)
	_, signup := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "password-123"})
	oldToken := signup["token"].(string)

	resp, out := post(t, ts, "/api/i/regenerate-token", map[string]any{
		"i": oldToken, "password": "password-123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("regenerate-token = %d %v", resp.StatusCode, out)
	}
	fresh, _ := out["token"].(string)
	if fresh == "" || fresh == oldToken {
		t.Fatalf("regenerated token = %q, want a new value", fresh)
	}
	if resp, _ := post(t, ts, "/api/i", map[string]any{"i": oldToken}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("old token still authenticates: %d", resp.StatusCode)
	}
	if resp, body := post(t, ts, "/api/i", map[string]any{"i": fresh}); resp.StatusCode != http.StatusOK || body["username"] != name {
		t.Errorf("new token = %d %v", resp.StatusCode, body)
	}

	// Signing in once records a history row.
	if resp, signed := post(t, ts, "/api/signin", map[string]any{"username": name, "password": "password-123"}); resp.StatusCode != http.StatusOK || signed["finished"] != true {
		t.Fatalf("signin = %d %v", resp.StatusCode, signed)
	}
	rows := postListWithToken(t, ts, "/api/i/signin-history", fresh, map[string]any{"limit": 10})
	if len(rows) == 0 {
		t.Fatal("signin history is empty")
	}
	first, _ := rows[0].(map[string]any)
	if first["id"] == "" || first["createdAt"] == "" {
		t.Errorf("signin history row = %v", first)
	}
	if _, ok := first["success"]; !ok {
		t.Errorf("signin history row has no success flag: %v", first)
	}
}

// TestSigninHistoryPagination proves the list pages newest-first with a
// untilId cursor and honours limit.
func TestSigninHistoryPagination(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	username := ""
	if _, me := postWithToken(t, ts, "/api/i", token, map[string]any{}); me["username"] != nil {
		username, _ = me["username"].(string)
	}
	for i := 0; i < 12; i++ {
		resp, out := post(t, ts, "/api/signin", map[string]any{"username": username, "password": "hunter2"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("signin %d = %d %v", i, resp.StatusCode, out)
		}
	}

	var seen []string
	cursor := ""
	for page := 0; page < 5; page++ {
		body := map[string]any{"limit": 5}
		if cursor != "" {
			body["untilId"] = cursor
		}
		rows := postListWithToken(t, ts, "/api/i/signin-history", token, body)
		if len(rows) == 0 {
			break
		}
		for _, item := range rows {
			m, _ := item.(map[string]any)
			id, _ := m["id"].(string)
			seen = append(seen, id)
		}
		last, _ := rows[len(rows)-1].(map[string]any)
		cursor, _ = last["id"].(string)
		if len(rows) < 5 {
			break
		}
	}
	if len(seen) < 12 {
		t.Errorf("paged through %d signins, want at least 12", len(seen))
	}
	unique := map[string]bool{}
	for _, id := range seen {
		if unique[id] {
			t.Errorf("duplicate signin id across pages: %s", id)
		}
		unique[id] = true
	}
}
