package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestAdminForwardAbuseReport proves report forwarding end to end: an
// administrator on alpha reports a beta account, forwards it, and the Flag
// arrives on beta as a normal abuse report. Forwarding twice must not send
// the Flag again.
func TestAdminForwardAbuseReport(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	adminToken, adminID := a.signup(t, "alphaadmin")
	_, bobID := b.signup(t, "betabob")
	if _, err := a.srv.TestPool().Exec(context.Background(),
		`UPDATE users SET is_admin = TRUE WHERE id = $1`, adminID); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Bob follows alpha's administrator so alpha stores him as a remote actor
	// with a reachable inbox; without that there is nobody to forward to.
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
		"id":       bobActor + "#follows/forward",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   "https://alpha.example/users/" + adminID,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, body)
	}

	var remoteBobID string
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT id FROM users WHERE uri = $1`, bobActor).Scan(&remoteBobID); err != nil {
		t.Fatalf("remote bob on alpha: %v", err)
	}

	if status := postNoContentWithToken(t, a.ts, "/api/users/report-abuse", adminToken, map[string]any{
		"userId": remoteBobID, "comment": "harassing alpha users",
	}); status != http.StatusNoContent {
		t.Fatalf("report-abuse = %d, want 204", status)
	}
	reports := postListWithToken(t, a.ts, "/api/admin/abuse-user-reports", adminToken, nil)
	if len(reports) != 1 {
		t.Fatalf("reports = %v, want one", reports)
	}
	report := reports[0].(map[string]any)
	if report["forwarded"] != false {
		t.Errorf("forwarded = %v before forwarding, want false", report["forwarded"])
	}
	reportID, _ := report["id"].(string)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)

	if status := postNoContentWithToken(t, a.ts, "/api/admin/forward-abuse-user-report", adminToken, map[string]any{
		"reportId": reportID,
	}); status != http.StatusNoContent {
		t.Fatalf("forward = %d, want 204", status)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		var stored int
		if err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT count(*) FROM abuse_reports WHERE target_user_id = $1 AND comment LIKE '%harassing alpha users%'`,
			bobID).Scan(&stored); err != nil {
			t.Fatalf("count beta reports: %v", err)
		}
		if stored == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the Flag never arrived on beta")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// The shared test database also holds the report beta created from the
	// Flag, so the row is selected by id instead of position.
	reports = postListWithToken(t, a.ts, "/api/admin/abuse-user-reports", adminToken, nil)
	if forwarded := reportByID(t, reports, reportID)["forwarded"]; forwarded != true {
		t.Errorf("forwarded = %v after forwarding, want true", forwarded)
	}

	// A second forward is a no-op: beta keeps exactly one report.
	if status := postNoContentWithToken(t, a.ts, "/api/admin/forward-abuse-user-report", adminToken, map[string]any{
		"reportId": reportID,
	}); status != http.StatusNoContent {
		t.Fatalf("second forward = %d, want 204", status)
	}
	time.Sleep(500 * time.Millisecond)
	var after int
	if err := b.srv.TestPool().QueryRow(context.Background(),
		`SELECT count(*) FROM abuse_reports WHERE target_user_id = $1`, bobID).Scan(&after); err != nil {
		t.Fatalf("count beta reports: %v", err)
	}
	if after != 1 {
		t.Errorf("beta reports = %d after a repeated forward, want 1", after)
	}

	// Unknown report id.
	if resp, body := postWithToken(t, a.ts, "/api/admin/forward-abuse-user-report", adminToken, map[string]any{
		"reportId": "00000000000000000000000000",
	}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown report = %d %v, want 404", resp.StatusCode, body)
	}
	// Missing reportId.
	if resp, body := postWithToken(t, a.ts, "/api/admin/forward-abuse-user-report", adminToken, map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing reportId = %d %v, want 400", resp.StatusCode, body)
	}
	// A non-administrator is rejected.
	if resp, _ := postWithToken(t, a.ts, "/api/admin/forward-abuse-user-report", "not-a-token", map[string]any{
		"reportId": reportID,
	}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous forward = %d, want 401", resp.StatusCode)
	}
}

// TestAdminForwardLocalReportRejected checks that a report about a local
// account is refused with a reason before anything is queued.
func TestAdminForwardLocalReportRejected(t *testing.T) {
	f := newFederationServer(t, "public")
	ts, pool := f.ts, f.pool
	adminID, adminToken := createUser(t, ts, "forwardadmin")
	if _, err := pool.Exec(context.Background(),
		`UPDATE users SET is_admin = TRUE WHERE id = $1`, adminID); err != nil {
		t.Fatalf("promote: %v", err)
	}
	targetID, _ := createUser(t, ts, "forwardtarget")

	if status := postNoContentWithToken(t, ts, "/api/users/report-abuse", adminToken, map[string]any{
		"userId": targetID, "comment": "local only",
	}); status != http.StatusNoContent {
		t.Fatalf("report-abuse = %d, want 204", status)
	}
	reports := postListWithToken(t, ts, "/api/admin/abuse-user-reports", adminToken, nil)
	if len(reports) != 1 {
		t.Fatalf("reports = %v, want one", reports)
	}
	id, _ := reports[0].(map[string]any)["id"].(string)

	resp, body := postWithToken(t, ts, "/api/admin/forward-abuse-user-report", adminToken, map[string]any{
		"reportId": id,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("forward of a local report = %d %v, want 400", resp.StatusCode, body)
	}
	errObject, _ := body["error"].(map[string]any)
	if code, _ := errObject["code"].(string); code != "CANNOT_FORWARD" {
		t.Fatalf("error = %v, want CANNOT_FORWARD", body)
	}

	// The refusal must not leave the report marked as forwarded.
	reports = postListWithToken(t, ts, "/api/admin/abuse-user-reports", adminToken, nil)
	if forwarded := reportByID(t, reports, id)["forwarded"]; forwarded != false {
		t.Errorf("forwarded = %v after a refused forward, want false", forwarded)
	}
}

// reportByID picks one report out of an admin list response.
func reportByID(t *testing.T, reports []any, id string) map[string]any {
	t.Helper()
	for _, item := range reports {
		row, _ := item.(map[string]any)
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("report %s not found in %v", id, reports)
	return nil
}
