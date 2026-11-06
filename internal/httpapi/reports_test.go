package httpapi_test

import (
	"bytes"
	"encoding/json"
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

// TestAbuseReportPagination covers the admin report list cursor and the state
// filter: reports page newest first, untilId continues the page, and resolving
// one moves it between the two state filters.
func TestAbuseReportPagination(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, target := post(t, ts, "/api/signup", map[string]any{"username": "target", "password": "password123"})
	targetID, _ := target["id"].(string)

	reporters := []string{"reporter1", "reporter2", "reporter3"}
	for _, name := range reporters {
		_, user := post(t, ts, "/api/signup", map[string]any{"username": name, "password": "password123"})
		token, _ := user["token"].(string)
		// users/report-abuse answers 204, which the JSON helpers cannot decode.
		payload, _ := json.Marshal(map[string]any{"i": token, "userId": targetID, "comment": "w63 report from " + name})
		resp, err := ts.Client().Post(ts.URL+"/api/users/report-abuse", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("report by %s: %v", name, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("report by %s: %d, want 204", name, resp.StatusCode)
		}
	}

	list := func(params map[string]any) []any {
		resp, raw := postRawWithToken(t, ts, "/api/admin/abuse-user-reports", "admintoken000000", params)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("reports %v: %d %v", params, resp.StatusCode, raw)
		}
		arr, _ := raw.([]any)
		return arr
	}

	all := list(map[string]any{})
	if len(all) != 3 {
		t.Fatalf("reports = %d, want 3", len(all))
	}
	// Newest first: reporter3's report is the first row.
	first, _ := all[0].(map[string]any)
	reporter, _ := first["reporter"].(map[string]any)
	if reporter["username"] != "reporter3" {
		t.Errorf("first reporter = %v, want reporter3", reporter["username"])
	}

	// limit=2 then continue from the second row.
	page := list(map[string]any{"limit": 2})
	if len(page) != 2 {
		t.Fatalf("limit 2 = %d rows, want 2", len(page))
	}
	second, _ := page[1].(map[string]any)
	secondID, _ := second["id"].(string)
	rest := list(map[string]any{"untilId": secondID})
	if len(rest) != 1 {
		t.Fatalf("untilId page = %d rows, want 1", len(rest))
	}
	oldest, _ := rest[0].(map[string]any)
	if oldReporter, _ := oldest["reporter"].(map[string]any); oldReporter["username"] != "reporter1" {
		t.Errorf("oldest reporter = %v, want reporter1", oldReporter["username"])
	}

	// Resolve the newest report and check both state filters.
	newestID, _ := first["id"].(string)
	if status := postNoContentWithToken(t, ts, "/api/admin/resolve-abuse-user-report", "admintoken000000", map[string]any{"reportId": newestID}); status != http.StatusNoContent {
		t.Fatalf("resolve: status %d, want 204", status)
	}
	if got := list(map[string]any{"state": "resolved"}); len(got) != 1 {
		t.Errorf("resolved reports = %d, want 1", len(got))
	}
	if got := list(map[string]any{"state": "unresolved"}); len(got) != 2 {
		t.Errorf("unresolved reports = %d, want 2", len(got))
	}
	if resp, _ := postRawWithToken(t, ts, "/api/admin/abuse-user-reports", "admintoken000000", map[string]any{"state": "sideways"}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid state status = %d, want 400", resp.StatusCode)
	}
}
