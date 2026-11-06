package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboxBehindReverseProxy verifies the deployment topology in the Caddyfile:
// a reverse proxy terminates public traffic and forwards to the app on
// 127.0.0.1, preserving the Host header. The inbox signs and verifies against
// whatever Host the peer connected to, so a proxied request must be accepted
// exactly like a direct one.
func TestInboxBehindReverseProxy(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, b, a)
	// The Accept leg runs in the other direction; without it alpha cannot
	// resolve beta's key when the proxy test's delivery status is checked.
	routeTo(t, a, b)

	_, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceKeyID, _, aliceActor := actorInfo(t, a, aliceID)
	_, _, bobActor := actorInfo(t, b, bobID)

	kp, err := a.srv.FederationLocalKey(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("alpha key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse alpha key: %v", err)
	}

	// The proxy forwards to beta's test server but leaves the Host header
	// alone, which is what Caddy's reverse_proxy does by default.
	target, err := url.Parse(b.ts.URL)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	front := httptest.NewServer(proxy)
	defer front.Close()

	follow := federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#follows/" + bobID,
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	}
	body, err := json.Marshal(follow)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, front.URL+"/inbox", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	// The peer addresses the public domain, not the proxy's loopback address.
	req.Host = "beta.example"
	req.Header.Set("Content-Type", "application/activity+json")
	if err := federation.SignRequest(req, body, aliceKeyID, priv); err != nil {
		t.Fatalf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("proxied inbox status = %d, want 202", resp.StatusCode)
	}
	if !b.srv.HasFollowingByActorURI(context.Background(), aliceActor, bobID) {
		t.Fatal("beta did not record the follow through the proxy")
	}

	// A signature made for the public Host must not verify when the request
	// reaches the app with a different Host: this is the replay protection the
	// signature binding provides, and the reason the app trusts r.Host.
	body2, _ := json.Marshal(federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#follows/" + bobID + "-again",
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	})
	req2, _ := http.NewRequest(http.MethodPost, b.ts.URL+"/inbox", bytes.NewReader(body2))
	req2.Host = "other.example"
	req2.Header.Set("Content-Type", "application/activity+json")
	if err := federation.SignRequest(req2, body2, aliceKeyID, priv); err != nil {
		t.Fatalf("sign second: %v", err)
	}
	// Re-sign for beta.example but send with other.example: the Host header is
	// the only difference, so verification must fail.
	req2.Host = "beta.example"
	if err := federation.SignRequest(req2, body2, aliceKeyID, priv); err != nil {
		t.Fatalf("re-sign: %v", err)
	}
	req2.Host = "other.example"
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("post second: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("host-mismatch status = %d, want 401", resp2.StatusCode)
	}
}
