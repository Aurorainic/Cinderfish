package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func adminReports(t *testing.T, ts *httptest.Server, body map[string]any) []any {
	t.Helper()
	resp, raw := postRawWithToken(t, ts, "/api/admin/abuse-user-reports", "admintoken000000", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("abuse-user-reports: status %d, body %v", resp.StatusCode, raw)
	}
	arr, _ := raw.([]any)
	return arr
}

func TestReportAbuseLifecycle(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobID, _ := bob["id"].(string)

	if status := postNoContentWithToken(t, ts, "/api/users/report-abuse", aliceToken, map[string]any{
		"userId": bobID, "comment": "spam in replies",
	}); status != http.StatusNoContent {
		t.Fatalf("report-abuse status = %d, want 204", status)
	}

	list := adminReports(t, ts, map[string]any{"state": "unresolved"})
	if len(list) != 1 {
		t.Fatalf("unresolved reports = %d, want 1: %v", len(list), list)
	}
	rep, _ := list[0].(map[string]any)
	if rep["resolved"] != false || rep["comment"] != "spam in replies" {
		t.Errorf("report = %v", rep)
	}
	reporter, _ := rep["reporter"].(map[string]any)
	target, _ := rep["targetUser"].(map[string]any)
	if reporter["username"] != "alice" || target["username"] != "bob" {
		t.Errorf("reporter/target = %v/%v, want alice/bob", reporter["username"], target["username"])
	}

	id, _ := rep["id"].(string)
	if status := postNoContentWithToken(t, ts, "/api/admin/resolve-abuse-user-report", "admintoken000000", map[string]any{
		"reportId": id,
	}); status != http.StatusNoContent {
		t.Fatalf("resolve status = %d, want 204", status)
	}

	if got := len(adminReports(t, ts, map[string]any{"state": "unresolved"})); got != 0 {
		t.Errorf("unresolved after resolve = %d, want 0", got)
	}
	if got := len(adminReports(t, ts, map[string]any{"state": "resolved"})); got != 1 {
		t.Errorf("resolved reports = %d, want 1", got)
	}

	// Both parties are local, so a remote-only filter yields nothing.
	if got := len(adminReports(t, ts, map[string]any{"reporterOrigin": "remote"})); got != 0 {
		t.Errorf("remote reporter reports = %d, want 0", got)
	}
}

func TestReportAbuseValidationAndPermissions(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	aliceID, _ := alice["id"].(string)
	_, bob := post(t, ts, "/api/signup", map[string]any{"username": "bob", "password": "password123"})
	bobID, _ := bob["id"].(string)

	// Self-report is rejected.
	resp, _ := postWithToken(t, ts, "/api/users/report-abuse", aliceToken, map[string]any{
		"userId": aliceID, "comment": "me",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self report status = %d, want 400", resp.StatusCode)
	}
	// Empty comment is rejected.
	resp, _ = postWithToken(t, ts, "/api/users/report-abuse", aliceToken, map[string]any{
		"userId": bobID, "comment": "   ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty comment status = %d, want 400", resp.StatusCode)
	}
	// Anonymous cannot report.
	resp, _ = post(t, ts, "/api/users/report-abuse", map[string]any{"userId": bobID, "comment": "x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous report status = %d, want 401", resp.StatusCode)
	}
	// A normal account cannot list reports.
	resp, _ = postWithToken(t, ts, "/api/admin/abuse-user-reports", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-admin list status = %d, want 403", resp.StatusCode)
	}
	// Unknown state is rejected instead of silently ignored.
	resp, _ = postWithToken(t, ts, "/api/admin/abuse-user-reports", "admintoken000000", map[string]any{"state": "bogus"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad state status = %d, want 400", resp.StatusCode)
	}
}
