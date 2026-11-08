package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
)

// fedInstance is a Cinderfish instance with ActivityPub wired up.
type fedInstance struct {
	ts     *httptest.Server
	srv    *httpapi.Server
	pool   *pgxpool.Pool
	domain string
}

// newFederationServer builds an instance whose AP surface is attached.
// federationMode is the configured mode; the instance_meta row is seeded
// to match, which is what the policy reads.
func newFederationServer(t *testing.T, federationMode string) *fedInstance {
	t.Helper()
	pool := testPool(t)
	domain := "fish.example"
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: domain,
		RegistrationMode: "open", FederationMode: federationMode,
		Media: config.Media{Backend: "local"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	srv.AttachStreaming(t.Logf)
	srv.AttachFederation()

	if _, err := pool.Exec(context.Background(), `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb($1::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, federationMode); err != nil {
		t.Fatalf("seed federation mode: %v", err)
	}

	srv.AllowPrivateWebhookTargets()
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return &fedInstance{ts: ts, srv: srv, pool: pool, domain: domain}
}

// get fetches path with the given Accept header.
func (f *fedInstance) get(t *testing.T, path, accept string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, f.ts.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp, body
}

// registerUser signs up an account and returns its token and id.
func (f *fedInstance) registerUser(t *testing.T, username string) (token, id string) {
	t.Helper()
	resp, out := post(t, f.ts, "/api/signup", map[string]any{
		"username": username, "password": "correct horse battery staple",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup %s: status %d body %v", username, resp.StatusCode, out)
	}
	tok, _ := out["token"].(string)
	uid, _ := out["id"].(string)
	if tok == "" || uid == "" {
		t.Fatalf("signup returned no token or id: %v", out)
	}
	return tok, uid
}

func TestWebFingerResolvesLocalUser(t *testing.T) {
	f := newFederationServer(t, "public")
	_, id := f.registerUser(t, "alice")

	for _, resource := range []string{
		"acct:alice@fish.example",
		"alice@fish.example",
		"https://fish.example/users/" + id,
	} {
		resp, body := f.get(t, "/.well-known/webfinger?resource="+resource, "application/jrd+json")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("resource %q: status %d body %s", resource, resp.StatusCode, body)
		}
		if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/jrd+json") {
			t.Errorf("content type = %q", got)
		}
		var jrd struct {
			Subject string                             `json:"subject"`
			Links   []struct{ Rel, Type, Href string } `json:"links"`
		}
		if err := json.Unmarshal(body, &jrd); err != nil {
			t.Fatalf("decode jrd: %v", err)
		}
		if jrd.Subject != "acct:alice@fish.example" {
			t.Errorf("subject = %q", jrd.Subject)
		}
		found := false
		for _, l := range jrd.Links {
			if l.Rel == "self" && l.Type == "application/activity+json" && strings.HasSuffix(l.Href, "/users/"+id) {
				found = true
			}
		}
		if !found {
			t.Errorf("no self link to the actor: %v", jrd.Links)
		}
	}
}

func TestWebFingerRejectsForeignAndUnknown(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, _ := f.get(t, "/.well-known/webfinger?resource=acct:bob@other.example", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("foreign resource status = %d, want 422", resp.StatusCode)
	}

	resp, _ = f.get(t, "/.well-known/webfinger?resource=acct:nobody@fish.example", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want 404", resp.StatusCode)
	}

	resp, _ = f.get(t, "/.well-known/webfinger", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing resource status = %d, want 400", resp.StatusCode)
	}
}

func TestActorDocument(t *testing.T) {
	f := newFederationServer(t, "public")
	_, id := f.registerUser(t, "alice")

	resp, body := f.get(t, "/users/"+id, "application/activity+json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/activity+json") {
		t.Errorf("content type = %q", ct)
	}
	if got := resp.Header.Get("Vary"); got != "Accept" {
		t.Errorf("Vary = %q", got)
	}

	var person map[string]any
	if err := json.Unmarshal(body, &person); err != nil {
		t.Fatalf("decode: %v", err)
	}
	actor := "https://fish.example/users/" + id
	if person["id"] != actor {
		t.Errorf("id = %v", person["id"])
	}
	if person["preferredUsername"] != "alice" {
		t.Errorf("preferredUsername = %v", person["preferredUsername"])
	}
	if person["inbox"] != actor+"/inbox" {
		t.Errorf("inbox = %v", person["inbox"])
	}
	if person["sharedInbox"] != "https://fish.example/inbox" {
		t.Errorf("sharedInbox = %v", person["sharedInbox"])
	}
	key, ok := person["publicKey"].(map[string]any)
	if !ok {
		t.Fatalf("publicKey is %T", person["publicKey"])
	}
	if key["id"] != actor+"#main-key" {
		t.Errorf("key id = %v", key["id"])
	}
	if pem, _ := key["publicKeyPem"].(string); !strings.Contains(pem, "PUBLIC KEY") {
		t.Errorf("public key pem = %v", key["publicKeyPem"])
	}
}

func TestActorViaHandleRoute(t *testing.T) {
	f := newFederationServer(t, "public")
	f.registerUser(t, "alice")

	resp, body := f.get(t, "/@alice", "application/activity+json")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body %s", resp.StatusCode, body)
	}
	var person map[string]any
	_ = json.Unmarshal(body, &person)
	if person["preferredUsername"] != "alice" {
		t.Errorf("preferredUsername = %v", person["preferredUsername"])
	}

	// A path that is not an @handle is not an actor.
	resp, _ = f.get(t, "/not-a-handle", "application/activity+json")
	if resp.StatusCode == http.StatusOK {
		t.Error("a non-handle path answered 200")
	}
}

func TestFederationModeNoneBlocksAP(t *testing.T) {
	f := newFederationServer(t, "none")
	_, id := f.registerUser(t, "alice")

	paths := []string{
		"/users/" + id,
		"/users/" + id + "/outbox",
		"/users/" + id + "/followers",
		"/notes/whatever",
		"/.well-known/webfinger?resource=acct:alice@fish.example",
		"/.well-known/nodeinfo",
		"/nodeinfo/2.1",
	}
	for _, p := range paths {
		resp, _ := f.get(t, p, "application/activity+json")
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s in mode none: status %d, want 403", p, resp.StatusCode)
		}
	}
}

func TestNodeInfo(t *testing.T) {
	f := newFederationServer(t, "public")

	resp, body := f.get(t, "/.well-known/nodeinfo", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var links struct {
		Links []struct{ Rel, Href string } `json:"links"`
	}
	if err := json.Unmarshal(body, &links); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(links.Links) != 2 {
		t.Fatalf("links = %v", links.Links)
	}
	if links.Links[0].Rel != "http://nodeinfo.diaspora.software/ns/schema/2.1" {
		t.Errorf("first rel = %q", links.Links[0].Rel)
	}

	resp, body = f.get(t, "/nodeinfo/2.1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("2.1 status = %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "profile=\"http://nodeinfo.diaspora.software/ns/schema/2.1#\"") {
		t.Errorf("content type = %q", ct)
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["version"] != "2.1" {
		t.Errorf("version = %v", doc["version"])
	}
	software, _ := doc["software"].(map[string]any)
	if software["version"] != "0.1.5" {
		t.Errorf("NodeInfo software.version = %v, want 0.1.5", software["version"])
	}
	protocols, _ := doc["protocols"].([]any)
	if len(protocols) != 1 || protocols[0] != "activitypub" {
		t.Errorf("protocols = %v", doc["protocols"])
	}
}

func TestHostMeta(t *testing.T) {
	f := newFederationServer(t, "public")

	resp, body := f.get(t, "/.well-known/host-meta.json", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("json status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "webfinger?resource={uri}") {
		t.Errorf("json host-meta has no template: %s", body)
	}

	resp, body = f.get(t, "/.well-known/host-meta", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("xml status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "xrd-1.0") {
		t.Errorf("xml host-meta is not XRD: %s", body)
	}
}
