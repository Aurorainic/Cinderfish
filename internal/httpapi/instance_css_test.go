package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestInstanceCustomCSS pins the instance stylesheet: admin/update-meta stores
// it, both meta endpoints publish it, the 100 KiB bound is enforced on bytes,
// and an empty string clears it.
func TestInstanceCustomCSS(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	css := "body { background: #101010; }"
	resp, updated := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"customCss": css,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta customCss = %d %v", resp.StatusCode, updated)
	}
	if updated["customCss"] != css {
		t.Errorf("admin update returned customCss %v, want the stylesheet", updated["customCss"])
	}

	// The public meta carries it, so the client can inject it before sign-in.
	_, meta := post(t, ts, "/api/meta", map[string]any{})
	if meta["customCss"] != css {
		t.Errorf("public meta customCss = %v, want %q", meta["customCss"], css)
	}
	_, adminMeta := postWithToken(t, ts, "/api/admin/meta", "admintoken000000", map[string]any{})
	if adminMeta["customCss"] != css {
		t.Errorf("admin/meta customCss = %v, want %q", adminMeta["customCss"], css)
	}

	// Exactly 100 KiB is accepted; one byte more is refused and the stored
	// stylesheet is left untouched.
	exact := strings.Repeat("a", 100*1024)
	if resp, out := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"customCss": exact,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("exactly 100 KiB = %d %v", resp.StatusCode, out)
	}
	resp, refused := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"customCss": exact + "b",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("100 KiB + 1 = %d %v, want 400", resp.StatusCode, refused)
	}
	if e, _ := refused["error"].(map[string]any); e["code"] != "INVALID_PARAM" {
		t.Errorf("oversize error = %v, want INVALID_PARAM", refused["error"])
	}
	_, meta = post(t, ts, "/api/meta", map[string]any{})
	if meta["customCss"] != exact {
		t.Errorf("meta after refused update changed length to %d", len(meta["customCss"].(string)))
	}

	// An empty string is the documented clear.
	if resp, cleared := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"customCss": "",
	}); resp.StatusCode != http.StatusOK || cleared["customCss"] != "" {
		t.Fatalf("clear customCss = %d %v", resp.StatusCode, cleared["customCss"])
	}
	_, meta = post(t, ts, "/api/meta", map[string]any{})
	if meta["customCss"] != "" {
		t.Errorf("meta after clear customCss = %v, want empty", meta["customCss"])
	}

	// Absent customCss leaves the stored value alone (other fields update).
	if resp, out := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"customCss": "p { color: red; }",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("restore customCss = %d %v", resp.StatusCode, out)
	}
	if resp, out := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"name": "Fish",
	}); resp.StatusCode != http.StatusOK || out["customCss"] != "p { color: red; }" {
		t.Errorf("unrelated update customCss = %v, want retained", out["customCss"])
	}
}
