package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// promoteAdmin flips the admin flag directly, since the project has no
// admin-invite endpoint yet.
func promoteAdmin(t *testing.T, f *fedInstance, userID string) {
	t.Helper()
	if _, err := f.srv.TestPool().Exec(context.Background(),
		"UPDATE users SET is_admin = TRUE WHERE id = $1", userID); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
}

func TestAdminCanSwitchFederationMode(t *testing.T) {
	f := newFederationServer(t, "public")
	token, id := f.registerUser(t, "alice")
	promoteAdmin(t, f, id)

	// A non-admin cannot change the mode.
	otherToken, _ := f.registerUser(t, "bob")
	resp, out := postWithToken(t, f.ts, "/api/admin/federation/mode", otherToken, map[string]any{"mode": "none"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin status = %d, want 403 (body %v)", resp.StatusCode, out)
	}

	// The admin can, and the change takes effect on the very next
	// request rather than at the next restart.
	resp, out = postWithToken(t, f.ts, "/api/admin/federation/mode", token, map[string]any{"mode": "none"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch mode: %d %v", resp.StatusCode, out)
	}
	if resp2, _ := f.get(t, "/users/"+id, "application/activity+json"); resp2.StatusCode != http.StatusForbidden {
		t.Errorf("actor after switching to none: status %d, want 403", resp2.StatusCode)
	}

	// An invalid mode is a parameter error, not a silent no-op.
	resp, out = postWithToken(t, f.ts, "/api/admin/federation/mode", token, map[string]any{"mode": "sometimes"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid mode status = %d, want 400 (%v)", resp.StatusCode, out)
	}

	// Switch back so the instance federates again.
	if resp, out := postWithToken(t, f.ts, "/api/admin/federation/mode", token, map[string]any{"mode": "public"}); resp.StatusCode != http.StatusOK {
		t.Fatalf("restore mode: %d %v", resp.StatusCode, out)
	}
	if resp2, _ := f.get(t, "/users/"+id, "application/activity+json"); resp2.StatusCode != http.StatusOK {
		t.Errorf("actor after restoring public: status %d", resp2.StatusCode)
	}
}

func TestAllowlistGatesFederation(t *testing.T) {
	f := newFederationServer(t, "allowlist")
	token, id := f.registerUser(t, "alice")
	promoteAdmin(t, f, id)

	// In allowlist mode the instance's own host is always allowed, but a
	// remote host is not until it is listed.
	resp, _ := f.get(t, "/users/"+id, "application/activity+json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("own actor in allowlist mode: status %d", resp.StatusCode)
	}

	resp, out := postWithToken(t, f.ts, "/api/admin/federation/allowlist", token,
		map[string]any{"host": "trusted.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add host: %d %v", resp.StatusCode, out)
	}
	hosts, _ := out["hosts"].([]any)
	if len(hosts) != 1 || hosts[0] != "trusted.example" {
		t.Errorf("hosts = %v", out["hosts"])
	}

	// The policy now permits the host and its subdomains, not a lookalike.
	policy, _, _, _ := f.srv.Federation()
	ctx := context.Background()
	if policy.CheckAllowed(ctx, "trusted.example") != nil {
		t.Error("listed host is not allowed")
	}
	if policy.CheckAllowed(ctx, "sub.trusted.example") != nil {
		t.Error("subdomain of a listed host is not allowed")
	}
	if policy.CheckAllowed(ctx, "nottrusted.example") == nil {
		t.Error("a lookalike host is allowed")
	}

	resp, out = postWithToken(t, f.ts, "/api/admin/federation/allowlist", token,
		map[string]any{"host": "trusted.example", "remove": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove host: %d %v", resp.StatusCode, out)
	}
	if hosts, _ := out["hosts"].([]any); len(hosts) != 0 {
		t.Errorf("hosts after removal = %v", hosts)
	}
}
