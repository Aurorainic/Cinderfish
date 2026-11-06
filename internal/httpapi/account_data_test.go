package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIExportContainsOwnDataOnly(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	createSearchNote(t, f, aliceToken, "alice export note", "public")
	createSearchNote(t, f, bobToken, "bob export note", "public")
	if resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: status %d body %v", resp.StatusCode, out)
	}
	// Alice favorites her own note.
	noteID := createSearchNote(t, f, aliceToken, "alice favorite target", "public")
	if resp, out := postWithToken(t, f.ts, "/api/notes/favorites/create", aliceToken, map[string]any{"noteId": noteID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("favorite: status %d body %v", resp.StatusCode, out)
	}

	req, err := http.NewRequest(http.MethodPost, f.ts.URL+"/api/i/export",
		strings.NewReader(`{"i":"`+aliceToken+`"}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("export status %d: %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Errorf("Content-Disposition = %q, want an attachment", got)
	}

	var doc struct {
		User       map[string]any   `json:"user"`
		Notes      []map[string]any `json:"notes"`
		Followings []map[string]any `json:"followings"`
		Favorites  []map[string]any `json:"favorites"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if doc.User["username"] != "alice" || doc.User["id"] != aliceID {
		t.Errorf("export user = %v, want alice/%s", doc.User, aliceID)
	}
	if len(doc.Notes) != 2 {
		t.Errorf("export notes = %d, want 2 (alice only)", len(doc.Notes))
	}
	for _, n := range doc.Notes {
		if strings.Contains(n["text"].(string), "bob") {
			t.Errorf("export leaked bob's note: %v", n)
		}
	}
	if len(doc.Followings) != 1 {
		t.Errorf("export followings = %d, want 1", len(doc.Followings))
	}
	if len(doc.Favorites) != 1 {
		t.Errorf("export favorites = %d, want 1", len(doc.Favorites))
	}
	if _, leaked := doc.User["token"]; leaked {
		t.Errorf("export leaked the token: %v", doc.User)
	}
}

func TestIDeleteAccount(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	// Wrong password does not delete.
	resp, out := postWithToken(t, f.ts, "/api/i/delete-account", token, map[string]any{"password": "wrong password"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong password status = %d, want 403 (body %v)", resp.StatusCode, out)
	}

	// Correct password deletes and invalidates the token.
	payload, _ := json.Marshal(map[string]any{"i": token, "password": "correct horse battery staple"})
	req, _ := http.NewRequest(http.MethodPost, f.ts.URL+"/api/i/delete-account", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.ts.Client().Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", resp.StatusCode)
	}

	if resp, _ := postWithToken(t, f.ts, "/api/i", token, map[string]any{}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("token after delete status = %d, want 401", resp.StatusCode)
	}
	if resp, _ := post(t, f.ts, "/api/users/show", map[string]any{"username": "alice"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("users/show after delete = %d, want 404", resp.StatusCode)
	}
}
