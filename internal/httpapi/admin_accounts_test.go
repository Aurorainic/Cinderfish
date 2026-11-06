package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// adminTestServer builds an instance with the given registration mode and
// an administrator inserted directly, because admin mode has no signup
// path by design.
func adminTestServer(t *testing.T, mode string) (*httptest.Server, *httpapi.Server) {
	t.Helper()
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: mode, FederationMode: "none",
		Media: config.Media{Backend: "local"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	// The binary always attaches the federation surface, even with federation
	// mode "none": handlers such as following/create use the store for local
	// rows too. Test servers must mirror that.
	srv.AttachFederation()
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("local storage: %v", err)
	}
	srv.SetStorage(storage)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, token, is_admin)
VALUES ('root1', 'root', 'root', 'x', 'admintoken000000', TRUE)`); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	return ts, srv
}

func postRawWithToken(t *testing.T, ts *httptest.Server, path, token string, body any) (*http.Response, any) {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	var decoded any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return resp, decoded
}

func TestAdminCreatedAccountUnderAdminRegistration(t *testing.T) {
	ts, _ := adminTestServer(t, "admin")

	// Self-service signup is closed in admin mode.
	if resp, _ := post(t, ts, "/api/signup", map[string]any{
		"username": "selfserve", "password": "password123",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("signup in admin mode = %d, want 400", resp.StatusCode)
	}

	// The administrator can create one directly.
	resp, raw := postRawWithToken(t, ts, "/api/admin/accounts/create", "admintoken000000",
		map[string]any{"username": "created", "password": "password123"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin create = %d, body %v", resp.StatusCode, raw)
	}
	created, _ := raw.(map[string]any)
	if created["username"] != "created" {
		t.Fatalf("created user = %v", created)
	}
	createdID, _ := created["id"].(string)

	// And the new account can sign in.
	resp, body := post(t, ts, "/api/signin", map[string]any{"username": "created", "password": "password123"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signin as created = %d, body %v", resp.StatusCode, body)
	}
	createdToken, _ := body["i"].(string)

	// show-users lists local accounts with the administrator first.
	resp, list := postRawWithToken(t, ts, "/api/admin/show-users", "admintoken000000", map[string]any{"limit": 10})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show-users = %d, body %v", resp.StatusCode, list)
	}
	users, ok := list.([]any)
	if !ok || len(users) != 2 {
		t.Fatalf("show-users = %v, want two users", list)
	}
	if users[0].(map[string]any)["username"] != "root" {
		t.Errorf("first user = %v, want the administrator", users[0])
	}

	// A non-administrator cannot create accounts.
	if resp, _ := postWithToken(t, ts, "/api/admin/accounts/create", createdToken,
		map[string]any{"username": "nope", "password": "password123"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin create = %d, want 403", resp.StatusCode)
	}

	// Suspension blocks both signin and the existing token.
	if resp, _ := postWithToken(t, ts, "/api/admin/suspend-user", "admintoken000000",
		map[string]any{"userId": createdID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("suspend = %d", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/signin", map[string]any{"username": "created", "password": "password123"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("signin while suspended = %d, want 403", resp.StatusCode)
	}
	if resp, _ := postWithToken(t, ts, "/api/i", createdToken, map[string]any{}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("i while suspended = %d, want 401", resp.StatusCode)
	}

	// The administrator cannot suspend itself.
	if resp, _ := postWithToken(t, ts, "/api/admin/suspend-user", "admintoken000000",
		map[string]any{"userId": "root1"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self-suspend = %d, want 400", resp.StatusCode)
	}

	// Unsuspension restores access.
	if resp, _ := postWithToken(t, ts, "/api/admin/unsuspend-user", "admintoken000000",
		map[string]any{"userId": createdID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("unsuspend = %d", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/signin", map[string]any{"username": "created", "password": "password123"}); resp.StatusCode != http.StatusOK {
		t.Errorf("signin after unsuspend = %d, want 200", resp.StatusCode)
	}
}

func TestAdminCreateRejectsDuplicate(t *testing.T) {
	ts, _ := adminTestServer(t, "admin")
	body := map[string]any{"username": "dup", "password": "password123"}
	if resp, _ := postWithToken(t, ts, "/api/admin/accounts/create", "admintoken000000", body); resp.StatusCode != http.StatusOK {
		t.Fatalf("first create = %d", resp.StatusCode)
	}
	if resp, _ := postWithToken(t, ts, "/api/admin/accounts/create", "admintoken000000", body); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("duplicate create = %d, want 400", resp.StatusCode)
	}
}
