package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
)

// The last administrator cannot delete themselves: the guard is checked before
// any row is touched. This needs an instance with exactly one admin, so the
// pool is used directly instead of adminTestServer (which leaves other rows in
// the shared test database untouched only for the tables it truncates).
func TestDeleteLastAdminRefused(t *testing.T) {
	pool := testPool(t)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "none",
		Media: config.Media{Backend: "local"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, token, is_admin)
VALUES ('root1', 'root', 'root', $1, 'admintoken000000', TRUE)`, hash); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	resp, raw := postRaw(t, ts, "/api/i/delete-account", map[string]any{
		"i": "admintoken000000", "password": "password123",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("last admin delete status = %d, want 400 (body %v)", resp.StatusCode, raw)
	}
	body, _ := raw.(map[string]any)
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "LAST_ADMIN" {
		t.Errorf("error code = %v, want LAST_ADMIN", errObj["code"])
	}

	// The account is still there and still usable.
	resp, _ = postWithToken(t, ts, "/api/i", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin after refused delete = %d, want 200", resp.StatusCode)
	}
}
