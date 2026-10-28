package httpapi_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"milnfish/internal/config"
	"milnfish/internal/federation"
	"milnfish/internal/httpapi"
)

// instance is one federating MilnFish server in a two-instance test.
type instance struct {
	name   string
	srv    *httpapi.Server
	ts     *httptest.Server
	domain string
}

// newInstance starts an instance whose ActivityPub identity is its
// configured domain. The transport is redirected to the httptest server
// so the two instances really talk to each other over HTTP.
func newInstance(t *testing.T, name string) *instance {
	t.Helper()
	pool := testPool(t)
	domain := name + ".example"
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: domain,
		RegistrationMode: "open", FederationMode: "public",
		Media:      config.Media{Backend: "local"},
		Federation: config.Federation{PublicURL: "https://" + domain},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	srv.AttachStreaming(t.Logf)
	srv.AttachFederation()

	if _, err := pool.Exec(context.Background(), `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb('public'::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`); err != nil {
		t.Fatalf("seed federation mode: %v", err)
	}

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return &instance{name: name, srv: srv, ts: ts, domain: domain}
}

// routeTo points this instance's federation client at another instance's
// httptest server, translating https://<other>.example to the real
// address while preserving the path.
func routeTo(t *testing.T, from, to *instance) {
	t.Helper()
	from.srv.SetFederationHTTPClient(&http.Client{
		Transport: &rewriteTransport{host: to.domain, target: to.ts.URL},
		Timeout:   10 * timeSecond,
	})
}

// rewriteTransport sends requests for host to target's address.
type rewriteTransport struct {
	host   string
	target string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != rt.host {
		return http.DefaultTransport.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = strings.TrimPrefix(rt.target, "http://")
	// The signed Host header must stay the domain, not the test address.
	clone.Host = rt.host
	return http.DefaultTransport.RoundTrip(clone)
}

// signup registers a user and returns the token and id.
func (in *instance) signup(t *testing.T, username string) (string, string) {
	t.Helper()
	resp, out := post(t, in.ts, "/api/signup", map[string]any{
		"username": username, "password": "correct horse battery staple",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s signup %s: %d %v", in.name, username, resp.StatusCode, out)
	}
	return out["token"].(string), out["id"].(string)
}

// getPlain performs a GET without Accept negotiation.
func getPlain(t *testing.T, ts *httptest.Server, path string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

// actorInfo reads an instance's actor document.
func actorInfo(t *testing.T, in *instance, userID string) (keyID string, pub *rsa.PublicKey, actorURI string) {
	t.Helper()
	resp, body := getPlain(t, in.ts, "/users/"+userID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s actor: %d %s", in.name, resp.StatusCode, body)
	}
	var person map[string]any
	if err := json.Unmarshal(body, &person); err != nil {
		t.Fatalf("decode actor: %v", err)
	}
	key, ok := person["publicKey"].(map[string]any)
	if !ok {
		t.Fatalf("%s actor has no publicKey", in.name)
	}
	pem, _ := key["publicKeyPem"].(string)
	pub, err := federation.ParsePublicKeyPEM(pem)
	if err != nil {
		t.Fatalf("parse public key: %v", err)
	}
	return key["id"].(string), pub, person["id"].(string)
}

// postSigned delivers a signed activity to an inbox path.
func postSigned(t *testing.T, target *httptest.Server, inboxPath string, activity federation.Object, keyID string, priv *rsa.PrivateKey) (*http.Response, []byte) {
	t.Helper()
	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatalf("encode activity: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, target.URL+inboxPath, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", "application/activity+json")
	if err := federation.SignRequest(req, body, keyID, priv); err != nil {
		t.Fatalf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST inbox: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp, respBody
}

func TestInboxRejectsUnsigned(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, err := http.Post(f.ts.URL+"/inbox", "application/activity+json",
		strings.NewReader(`{"type":"Create","actor":"https://a.example/users/1"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unsigned inbox status = %d, want 401", resp.StatusCode)
	}
}

func TestInboxRejectsActivityWithoutActor(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, err := http.Post(f.ts.URL+"/inbox", "application/activity+json", strings.NewReader("null"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		t.Errorf("status = %d, want a rejection", resp.StatusCode)
	}
}

func TestInboxRejectsTamperedBody(t *testing.T) {
	a := newInstance(t, "alpha")
	f := newFederationServer(t, "public")
	f.registerUser(t, "fishuser")
	_, aliceID := a.signup(t, "alphaalice")

	keyID, _, _ := actorInfo(t, a, aliceID)
	kp, err := federation.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	priv, _ := federation.ParsePrivateKeyPEM(kp.PrivatePEM)

	body := []byte(`{"type":"Follow","actor":"` + keyID + `"}`)
	tampered := append([]byte{}, body...)
	tampered[0] = '['

	req, err := http.NewRequest(http.MethodPost, f.ts.URL+"/inbox", bytes.NewReader(tampered))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", "application/activity+json")
	// Sign the original body, send the tampered one: the digest check
	// must catch it before the signature is even considered.
	if err := federation.SignRequest(req, body, keyID, priv); err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("tampered body status = %d, want 401", resp.StatusCode)
	}
}

// TestTwoInstanceFollow is the M5 smoke test: two MilnFish instances
// exchange a Follow over real HTTP with real RSA signatures. Beta
// resolves alpha's actor to verify the signature, exactly as a peer
// would, and ends up with the follow recorded.
func TestTwoInstanceFollow(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")

	_, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	aliceKeyID, _, aliceActor := actorInfo(t, a, aliceID)
	_, _, bobActor := actorInfo(t, b, bobID)

	// Beta must be able to reach alpha to fetch its actor document.
	routeTo(t, b, a)

	kp, err := a.srv.FederationLocalKey(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("alpha key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse alpha key: %v", err)
	}

	follow := federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#follows/" + bobID,
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	}

	resp, body := postSigned(t, b.ts, "/inbox", follow, aliceKeyID, priv)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow inbox status = %d body %s, want 202", resp.StatusCode, body)
	}

	if !b.srv.HasFollowingByActorURI(context.Background(), aliceActor, bobID) {
		t.Fatal("beta did not record the follow")
	}
}

// TestTwoInstanceFollowAccepted proves the return leg too: beta accepts
// the follow with a signed Accept, alpha verifies it against beta's
// published key and records the follow on its side.
func TestTwoInstanceFollowAccepted(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")

	_, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	_, _, aliceActor := actorInfo(t, a, aliceID)
	bobKeyID, _, bobActor := actorInfo(t, b, bobID)

	// Both directions: alpha must verify beta's Accept, and beta must
	// verify alpha's Follow.
	routeTo(t, b, a)
	routeTo(t, a, b)

	aliceKP, err := a.srv.FederationLocalKey(context.Background(), aliceID)
	if err != nil {
		t.Fatal(err)
	}
	alicePriv, _ := federation.ParsePrivateKeyPEM(aliceKP.PrivatePEM)

	follow := federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#follows/" + bobID,
		"type":     "Follow",
		"actor":    aliceActor,
		"object":   bobActor,
	}
	if resp, body := postSigned(t, b.ts, "/inbox", follow, "https://alpha.example/users/"+aliceID+"#main-key", alicePriv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow: %d %s", resp.StatusCode, body)
	}

	bobKP, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatal(err)
	}
	bobPriv, _ := federation.ParsePrivateKeyPEM(bobKP.PrivatePEM)

	accept := federation.Object{
		"@context": federation.Context,
		"id":       follow["id"].(string) + "#accept",
		"type":     "Accept",
		"actor":    bobActor,
		"object":   follow,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", accept, bobKeyID, bobPriv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("accept: %d %s", resp.StatusCode, body)
	}

	// alpha now has bob in alice's following list.
	if !a.srv.HasFollowingByActorURIIDs(context.Background(), aliceID, bobActor) {
		t.Fatal("alpha did not record the accepted follow")
	}
}

const timeSecond = 1000000000
