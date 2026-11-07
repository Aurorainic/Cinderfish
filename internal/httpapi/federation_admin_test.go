package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// getAPIWithToken issues a GET with a bearer token and decodes a JSON object.
func getAPIWithToken(t *testing.T, ts *httptest.Server, path, token string) (*http.Response, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return resp, decoded
}

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
	// GET reports the live switch, not the startup configuration.
	if resp, out := getAPIWithToken(t, f.ts, "/api/admin/federation/mode", token); resp.StatusCode != http.StatusOK || out["mode"] != "none" {
		t.Errorf("get mode = %d %v, want 200 {mode:none}", resp.StatusCode, out)
	}
	if resp, _ := getAPIWithToken(t, f.ts, "/api/admin/federation/mode", otherToken); resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin get mode = %d, want 403", resp.StatusCode)
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

// The visitor homepage reads meta.federation, so it has to follow the runtime
// switch rather than the startup configuration: an administrator switching to
// the allowlist must be reflected for anonymous clients immediately.
func TestMetaReportsLiveFederationMode(t *testing.T) {
	f := newFederationServer(t, "public")

	if _, meta := post(t, f.ts, "/api/meta", map[string]any{"detail": true}); meta["federation"] != "all" {
		t.Fatalf("meta.federation = %v, want all before the switch", meta["federation"])
	}

	for _, tc := range []struct{ mode, want string }{
		{"allowlist", "specified"},
		{"none", "none"},
	} {
		if _, err := f.pool.Exec(context.Background(), `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb($1::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
		`, tc.mode); err != nil {
			t.Fatalf("switch mode to %s: %v", tc.mode, err)
		}
		_, meta := post(t, f.ts, "/api/meta", map[string]any{"detail": true})
		if meta["federation"] != tc.want {
			t.Errorf("meta.federation after %s = %v, want %s", tc.mode, meta["federation"], tc.want)
		}
		features, _ := meta["features"].(map[string]any)
		if want := tc.mode != "none"; features["globalTimeline"] != want {
			t.Errorf("features.globalTimeline after %s = %v, want %v", tc.mode, features["globalTimeline"], want)
		}
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

// TestAllowlistGatesInboundInbox closes the loop the policy test leaves open:
// in allowlist mode an inbound activity from an unlisted host cannot even be
// verified, and listing the host lets the same Follow through.
func TestAllowlistGatesInboundInbox(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	// Flip alpha to allowlist after startup; the policy reads the live meta.
	if _, err := a.srv.TestPool().Exec(context.Background(), `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb('allowlist'::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatalf("seed allowlist mode: %v", err)
	}

	adminToken, aliceID := a.signup(t, "alphaalice")
	if _, err := a.srv.TestPool().Exec(context.Background(),
		"UPDATE users SET is_admin = TRUE WHERE id = $1", aliceID); err != nil {
		t.Fatalf("promote: %v", err)
	}
	_, bobID := b.signup(t, "betabob")

	_, _, aliceActor := actorInfo(t, a, aliceID)
	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse beta key: %v", err)
	}

	follow := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#follows/1",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActor,
	}
	// Unlisted host: the signature cannot be verified, so the activity is
	// refused before it reaches the dispatcher.
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("inbound from unlisted host = %d %s, want 401", resp.StatusCode, body)
	}
	if a.srv.HasFollowingByActorURI(context.Background(), bobActor, aliceID) {
		t.Fatal("unlisted follow was recorded")
	}

	// Listing the host changes the answer for the same activity.
	resp, out := postWithToken(t, a.ts, "/api/admin/federation/allowlist", adminToken,
		map[string]any{"host": "beta.example"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("allow host: %d %v", resp.StatusCode, out)
	}
	follow["id"] = bobActor + "#follows/2"
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("inbound after allowlisting = %d %s, want 202", resp.StatusCode, body)
	}
	if !a.srv.HasFollowingByActorURI(context.Background(), bobActor, aliceID) {
		t.Error("allowed follow was not recorded")
	}
}
