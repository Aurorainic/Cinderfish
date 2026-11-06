package httpapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboundFlagCreatesReport proves a remote instance can report one of our
// accounts: the Flag lands in the admin abuse report list as a normal report,
// with the reporting actor as its reporter.
func TestInboundFlagCreatesReport(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	// Alpha must reach beta to verify the signature.
	routeTo(t, a, b)

	adminToken, adminID := a.signup(t, "alphaadmin")
	_, bobID := b.signup(t, "betabob")
	if _, err := a.srv.TestPool().Exec(context.Background(),
		`UPDATE users SET is_admin = TRUE WHERE id = $1`, adminID); err != nil {
		t.Fatalf("promote: %v", err)
	}

	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse beta key: %v", err)
	}

	target := "https://alpha.example/users/" + adminID
	flag := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#flags/1",
		"type":     "Flag",
		"actor":    bobActor,
		"object":   []any{target},
		"content":  "spam from this account",
	}
	if resp, body := postSigned(t, a.ts, "/inbox", flag, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Flag = %d %s", resp.StatusCode, body)
	}

	reports := postListWithToken(t, a.ts, "/api/admin/abuse-user-reports", adminToken, nil)
	if len(reports) != 1 {
		t.Fatalf("reports = %v, want one", reports)
	}
	report, _ := reports[0].(map[string]any)
	comment, _ := report["comment"].(string)
	if !strings.Contains(comment, "spam from this account") {
		t.Errorf("comment = %q, want the Flag content", comment)
	}
	if !strings.Contains(comment, target) {
		t.Errorf("comment = %q, want the target URI", comment)
	}
	reporter, _ := report["reporter"].(map[string]any)
	if reporter["username"] != "betabob" {
		t.Errorf("reporter = %v, want betabob", report["reporter"])
	}
	if report["resolved"] != false {
		t.Errorf("resolved = %v, want false", report["resolved"])
	}
}
