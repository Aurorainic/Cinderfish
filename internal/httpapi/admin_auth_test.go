package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// adminRequest performs one admin call, with an optional bearer token, and
// decodes the JSON envelope so a test can read the error code.
func adminRequest(t *testing.T, ts *httptest.Server, method, path, token string, body map[string]any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode %s body: %v", path, err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("new request %s: %v", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do %s: %v", path, err)
	}
	defer resp.Body.Close()

	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

// TestAdminEndpointsRequireAdministrator pins the server-side half of the admin
// gate. The web client also renders a permission card for a non-administrator,
// but that card is UX only: every admin endpoint has to refuse the request on
// its own, so hiding the navigation entry can never be the control.
//
// The endpoint list mirrors the `/api/admin/...` registrations in NewServer. A
// newly added admin endpoint belongs in this table; if it is registered without
// requireAdmin this test fails with a 200 for the plain account.
func TestAdminEndpointsRequireAdministrator(t *testing.T) {
	f := newFederationServer(t, "none")
	adminToken, adminID := f.registerUser(t, "adminone")
	promoteAdmin(t, f, adminID)
	plainToken, plainID := f.registerUser(t, "plainone")

	type call struct {
		method string
		path   string
		body   map[string]any
	}
	// Every body is one the handler rejects *after* authorization — an empty
	// username, an unknown mode, a missing host, self-suspension — so the
	// administrator leg proves the guard opened without mutating the shared
	// test database that the neighbouring tests run against.
	calls := []call{
		{http.MethodPost, "/api/admin/accounts/create", map[string]any{"username": "", "password": ""}},
		{http.MethodPost, "/api/admin/show-users", map[string]any{}},
		{http.MethodPost, "/api/admin/suspend-user", map[string]any{"userId": adminID}},
		{http.MethodPost, "/api/admin/unsuspend-user", map[string]any{"userId": plainID}},
		{http.MethodPost, "/api/admin/federation/mode", map[string]any{"mode": "sometimes"}},
		{http.MethodGet, "/api/admin/federation/allowlist", nil},
		{http.MethodPost, "/api/admin/federation/allowlist", map[string]any{"host": ""}},
	}

	for _, c := range calls {
		name := c.method + " " + c.path

		status, _ := adminRequest(t, f.ts, c.method, c.path, "", c.body)
		if status != http.StatusUnauthorized {
			t.Errorf("%s without credential = %d, want 401", name, status)
		}

		status, out := adminRequest(t, f.ts, c.method, c.path, plainToken, c.body)
		if status != http.StatusForbidden {
			t.Errorf("%s as non-admin = %d, want 403 (body %v)", name, status, out)
		}
		if errOut, ok := out["error"].(map[string]any); ok {
			if code, _ := errOut["code"].(string); code != "FORBIDDEN" {
				t.Errorf("%s as non-admin: code = %q, want FORBIDDEN", name, code)
			}
		} else {
			t.Errorf("%s as non-admin: body = %v, want an error envelope", name, out)
		}

		// The administrator clears the same guard, which proves the 403 above
		// came from the admin check and not from an unrelated parse failure.
		if status, _ := adminRequest(t, f.ts, c.method, c.path, adminToken, c.body); status == http.StatusUnauthorized || status == http.StatusForbidden {
			t.Errorf("%s as admin = %d, want the guard to let it through", name, status)
		}
	}
}
