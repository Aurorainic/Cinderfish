package httpapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestIKnownAsLocalAlias proves i/known-as stores the alias list, publishes
// it on the actor document and returns it from users/show. A local account
// named by bare username is enough for the shape; the remote path is
// covered by the two-instance move test.
func TestIKnownAsLocalAlias(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")
	bobURI := "https://fish.example/users/" + bobID

	resp, out := postWithToken(t, f.ts, "/api/i/known-as", aliceToken, map[string]any{
		"alsoKnownAs": []string{"@bob"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/known-as = %d %v", resp.StatusCode, out)
	}
	aliases, _ := out["alsoKnownAs"].([]any)
	if len(aliases) != 1 || aliases[0] != bobURI {
		t.Fatalf("response alsoKnownAs = %v, want [%s]", out["alsoKnownAs"], bobURI)
	}

	// The actor document is what a peer validates a Move against.
	actorResp, body := getPlain(t, f.ts, "/users/"+aliceID)
	if actorResp.StatusCode != http.StatusOK {
		t.Fatalf("actor = %d %s", actorResp.StatusCode, body)
	}
	if !strings.Contains(string(body), bobURI) {
		t.Errorf("actor document has no alsoKnownAs entry %s: %s", bobURI, body)
	}

	// users/show carries the same list.
	_, show := postWithToken(t, f.ts, "/api/users/show", "", map[string]any{"userId": aliceID})
	shown, _ := show["alsoKnownAs"].([]any)
	if len(shown) != 1 || shown[0] != bobURI {
		t.Errorf("users/show alsoKnownAs = %v, want [%s]", show["alsoKnownAs"], bobURI)
	}

	// Naming yourself is refused.
	selfResp, self := postWithToken(t, f.ts, "/api/i/known-as", aliceToken, map[string]any{
		"alsoKnownAs": []string{"@alice"},
	})
	if selfResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("self alias = %d %v, want 400", selfResp.StatusCode, self)
	}
	if self["error"] == nil || self["error"].(map[string]any)["code"] != "FORBIDDEN_TO_SET_YOURSELF" {
		t.Errorf("self alias error = %v, want FORBIDDEN_TO_SET_YOURSELF", self["error"])
	}
}

// TestIMoveRefusesLocalAndAdminTarget covers the two checks that need no
// network: a local account is not a move target, and an administrator may
// not migrate at all (Misskey's rule, kept by Firefish).
func TestIMoveRefusesLocalAndAdminTarget(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	f.registerUser(t, "bob")

	resp, out := postWithToken(t, f.ts, "/api/i/move", aliceToken, map[string]any{"moveToAccount": "bob"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("local move target = %d %v, want 400", resp.StatusCode, out)
	}
	if out["error"].(map[string]any)["code"] != "NOT_REMOTE" {
		t.Errorf("local move error = %v, want NOT_REMOTE", out["error"])
	}

	if _, err := f.pool.Exec(context.Background(), `UPDATE users SET is_admin = TRUE WHERE id = $1`, aliceID); err != nil {
		t.Fatalf("promote: %v", err)
	}
	adminResp, admin := postWithToken(t, f.ts, "/api/i/move", aliceToken, map[string]any{"moveToAccount": "@bob@beta.example"})
	if adminResp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin move = %d %v, want 403", adminResp.StatusCode, admin)
	}
	if admin["error"].(map[string]any)["code"] != "NOT_ADMIN_FORBIDDEN" {
		t.Errorf("admin move error = %v, want NOT_ADMIN_FORBIDDEN", admin["error"])
	}
}

// TestTwoInstanceMove exercises the whole outbound move: the target must
// publish the mover as an alias before the move is accepted, and afterwards
// the mover's actor document advertises where it went.
func TestTwoInstanceMove(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	bobActor := "https://beta.example/users/" + bobID

	// Without the alias on the target the move is refused.
	refused, body := postWithToken(t, a.ts, "/api/i/move", aliceToken, map[string]any{
		"moveToAccount": "@betabob@beta.example",
	})
	if refused.StatusCode != http.StatusBadRequest {
		t.Fatalf("move without alias = %d %v, want 400", refused.StatusCode, body)
	}
	if body["error"].(map[string]any)["code"] != "REMOTE_ACCOUNT_FORBIDS" {
		t.Errorf("move without alias error = %v, want REMOTE_ACCOUNT_FORBIDS", body["error"])
	}

	// Beta declares alpha's account as an alias of the destination.
	if resp, out := postWithToken(t, b.ts, "/api/i/known-as", bobToken, map[string]any{
		"alsoKnownAs": []string{"@alphaalice@alpha.example"},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("beta i/known-as = %d %v", resp.StatusCode, out)
	}

	resp, out := postWithToken(t, a.ts, "/api/i/move", aliceToken, map[string]any{
		"moveToAccount": "@betabob@beta.example",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/move = %d %v", resp.StatusCode, out)
	}
	moved, _ := out["movedToUri"].(map[string]any)
	if moved == nil || moved["username"] != "betabob" || moved["host"] != "beta.example" {
		t.Fatalf("movedToUri = %v, want betabob@beta.example", out["movedToUri"])
	}

	// The actor document now carries movedTo, so peers can redirect.
	actorResp, actorBody := getPlain(t, a.ts, "/users/"+aliceID)
	if actorResp.StatusCode != http.StatusOK {
		t.Fatalf("actor = %d %s", actorResp.StatusCode, actorBody)
	}
	if !strings.Contains(string(actorBody), `"movedTo":"`+bobActor+`"`) {
		t.Errorf("actor document has no movedTo %s: %s", bobActor, actorBody)
	}

	// A second move is refused: an account moves once.
	again, againBody := postWithToken(t, a.ts, "/api/i/move", aliceToken, map[string]any{
		"moveToAccount": "@betabob@beta.example",
	})
	if again.StatusCode != http.StatusBadRequest {
		t.Fatalf("second move = %d %v, want 400", again.StatusCode, againBody)
	}
	if againBody["error"].(map[string]any)["code"] != "ALREADY_MOVED" {
		t.Errorf("second move error = %v, want ALREADY_MOVED", againBody["error"])
	}
}

// TestInboundMove proves the receiving side: a Move is refused until the
// target lists the mover as an alias, and accepted afterwards.
func TestInboundMove(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	// Beta has to reach alpha to resolve the mover and itself to read the
	// target actor document on the public domain, the way a real instance
	// would.
	b.srv.SetFederationHTTPClient(&http.Client{
		Transport: &hostRewriteTransport{hosts: map[string]string{
			"alpha.example": a.ts.URL,
			"beta.example":  b.ts.URL,
		}},
		Timeout: 10 * timeSecond,
	})

	_, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	aliceActor := "https://alpha.example/users/" + aliceID
	bobActor := "https://beta.example/users/" + bobID

	actorInfo(t, a, aliceID)
	kp, err := a.srv.FederationLocalKey(context.Background(), aliceID)
	if err != nil {
		t.Fatalf("alpha key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse alpha key: %v", err)
	}
	keyID := "https://alpha.example/users/" + aliceID + "#main-key"
	move := federation.Object{
		"@context": federation.Context,
		"id":       aliceActor + "#moves/1",
		"type":     "Move",
		"actor":    aliceActor,
		"object":   aliceActor,
		"target":   bobActor,
	}

	if resp, body := postSigned(t, b.ts, "/inbox", move, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("inbox move = %d %s, want 202", resp.StatusCode, body)
	}
	var moved string
	err = b.srv.TestPool().QueryRow(context.Background(),
		`SELECT COALESCE(moved_to_uri, '') FROM users WHERE uri = $1`, aliceActor).Scan(&moved)
	if err != nil {
		t.Fatalf("read mover: %v", err)
	}
	if moved != "" {
		t.Fatalf("mover marked moved without a target alias: %q", moved)
	}

	// Bob declares the alias; the resent move is then accepted.
	if resp, out := postWithToken(t, b.ts, "/api/i/known-as", bobToken, map[string]any{
		"alsoKnownAs": []string{"@alphaalice@alpha.example"},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("beta i/known-as = %d %v", resp.StatusCode, out)
	}
	move["id"] = aliceActor + "#moves/2"
	if resp, body := postSigned(t, b.ts, "/inbox", move, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("inbox move retry = %d %s", resp.StatusCode, body)
	}
	if err := b.srv.TestPool().QueryRow(context.Background(),
		`SELECT COALESCE(moved_to_uri, '') FROM users WHERE uri = $1`, aliceActor).Scan(&moved); err != nil {
		t.Fatalf("read mover again: %v", err)
	}
	if moved != bobActor {
		t.Fatalf("moved_to_uri = %q, want %q", moved, bobActor)
	}
}

// hostRewriteTransport routes several federation domains to their httptest
// servers, so one instance can be tested talking to a peer and to itself.
type hostRewriteTransport struct {
	hosts map[string]string
}

func (h *hostRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, ok := h.hosts[req.URL.Host]
	if !ok {
		return http.DefaultTransport.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = strings.TrimPrefix(target, "http://")
	// The signed Host header must stay the domain, not the test address.
	clone.Host = req.URL.Host
	return http.DefaultTransport.RoundTrip(clone)
}
