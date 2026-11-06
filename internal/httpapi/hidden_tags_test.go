package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestHiddenTags covers admin/update-meta's hiddenTags: the list is normalised
// (trim, strip #, lowercase, dedupe), read back through admin/meta, and the
// trend stops ranking the hidden tag while the public statistics still answer.
func TestHiddenTags(t *testing.T) {
	ts, _ := adminTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	createNote(t, ts, aliceToken, map[string]any{"text": "alpha #w63visible"})
	createNote(t, ts, bobToken, map[string]any{"text": "beta #w63hidden"})

	trendTags := func() []string {
		out := []string{}
		for _, item := range postListWithToken(t, ts, "/api/hashtags/trend", "", nil) {
			entry, _ := item.(map[string]any)
			tag, _ := entry["tag"].(string)
			out = append(out, tag)
		}
		return out
	}
	if got := trendTags(); len(got) != 2 {
		t.Fatalf("trend before hiding = %v, want two tags", got)
	}

	resp, meta := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"hiddenTags": []string{"#W63Hidden", "w63hidden", "   "},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta hiddenTags = %d %v", resp.StatusCode, meta)
	}
	hidden, _ := meta["hiddenTags"].([]any)
	if len(hidden) != 1 || hidden[0] != "w63hidden" {
		t.Fatalf("hiddenTags = %v, want [w63hidden]", meta["hiddenTags"])
	}

	// admin/meta reads the same list back.
	resp, admin := postWithToken(t, ts, "/api/admin/meta", "admintoken000000", map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin/meta = %d %v", resp.StatusCode, admin)
	}
	if back, _ := admin["hiddenTags"].([]any); len(back) != 1 || back[0] != "w63hidden" {
		t.Errorf("admin/meta hiddenTags = %v", admin["hiddenTags"])
	}

	// The trend no longer ranks the hidden tag.
	if got := trendTags(); len(got) != 1 || got[0] != "w63visible" {
		t.Errorf("trend after hiding = %v, want [w63visible]", got)
	}

	// Its public statistics still answer: hiding only affects the ranking.
	if resp, show := post(t, ts, "/api/hashtags/show", map[string]any{"tag": "w63hidden"}); resp.StatusCode != http.StatusOK || show["mentionedUsersCount"] != float64(1) {
		t.Errorf("hashtags/show for a hidden tag = %d %v", resp.StatusCode, show)
	}

	// An over-long tag is refused.
	if resp, _ := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"hiddenTags": []string{strings.Repeat("x", 101)},
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("over-long hidden tag = %d, want 400", resp.StatusCode)
	}

	// Clearing the list brings the tag back to the trend.
	if resp, _ := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"hiddenTags": []string{},
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("clear hiddenTags failed")
	}
	if got := trendTags(); len(got) != 2 {
		t.Errorf("trend after clearing = %v, want two tags", got)
	}
}
