package httpapi_test

import (
	"net/http"
	"testing"
)

func TestInstanceProfileLinksAndRules(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, raw := postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"tosUrl":           "https://example.com/tos",
		"privacyPolicyUrl": "https://example.com/privacy",
		"repositoryUrl":    "https://example.com/repo",
		"feedbackUrl":      "https://example.com/feedback",
		"inquiryUrl":       "https://example.com/contact",
		"impressumUrl":     "https://example.com/impressum",
		"serverRules":      []string{"第一条规则", " 第二条规则 ", ""},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta: status %d, body %v", resp.StatusCode, raw)
	}
	out, _ := raw.(map[string]any)
	if out["tosUrl"] != "https://example.com/tos" {
		t.Errorf("admin tosUrl = %v", out["tosUrl"])
	}
	rules, _ := out["serverRules"].([]any)
	if len(rules) != 2 || rules[0] != "第一条规则" || rules[1] != "第二条规则" {
		t.Errorf("serverRules = %v", out["serverRules"])
	}
	if out["privacyPolicyUrl"] != "https://example.com/privacy" {
		t.Errorf("privacyPolicyUrl = %v", out["privacyPolicyUrl"])
	}
	if out["impressumUrl"] != "https://example.com/impressum" {
		t.Errorf("impressumUrl = %v", out["impressumUrl"])
	}

	// The public meta carries the same values.
	resp, meta := post(t, ts, "/api/meta", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("meta: status %d", resp.StatusCode)
	}
	if meta["tosUrl"] != "https://example.com/tos" || meta["privacyPolicyUrl"] != "https://example.com/privacy" {
		t.Errorf("public meta urls = %v / %v", meta["tosUrl"], meta["privacyPolicyUrl"])
	}
	publicRules, _ := meta["serverRules"].([]any)
	if len(publicRules) != 2 {
		t.Errorf("public serverRules = %v", meta["serverRules"])
	}

	// Bad values are refused.
	resp, _ = postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"tosUrl": "javascript:alert(1)",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("javascript tosUrl status = %d, want 400", resp.StatusCode)
	}
	tooMany := make([]string, 21)
	for i := range tooMany {
		tooMany[i] = "r"
	}
	resp, _ = postRawWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"serverRules": tooMany,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("21 rules status = %d, want 400", resp.StatusCode)
	}
}

func TestStatsCounts(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	_, alice := post(t, ts, "/api/signup", map[string]any{"username": "alice", "password": "password123"})
	aliceToken, _ := alice["token"].(string)
	resp, out := postWithToken(t, ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "stats check", "visibility": "public",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/create: status %d, body %v", resp.StatusCode, out)
	}

	resp, stats := post(t, ts, "/api/stats", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats: status %d, body %v", resp.StatusCode, stats)
	}
	// root + alice
	if stats["usersCount"] != float64(2) {
		t.Errorf("usersCount = %v, want 2", stats["usersCount"])
	}
	if stats["notesCount"] != float64(1) || stats["originalNotesCount"] != float64(1) {
		t.Errorf("notes counts = %v / %v", stats["notesCount"], stats["originalNotesCount"])
	}
	if stats["driveFiles"] != float64(0) {
		t.Errorf("driveFiles = %v, want 0", stats["driveFiles"])
	}
	if _, ok := stats["instancesCount"]; !ok {
		t.Errorf("instancesCount missing: %v", stats)
	}
}
