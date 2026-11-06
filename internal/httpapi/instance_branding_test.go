package httpapi_test

import (
	"net/http"
	"testing"
)

// TestInstanceBranding covers admin/update-meta's iconUrl, bannerUrl and
// themeColor: the public meta and admin/meta both carry them, clearing works,
// and an invalid theme colour is refused.
func TestInstanceBranding(t *testing.T) {
	ts, _ := adminTestServer(t, "open")

	resp, updated := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"iconUrl":    "https://cdn.example/icon.png",
		"bannerUrl":  "https://cdn.example/banner.png",
		"themeColor": "#12ab34",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-meta branding = %d %v", resp.StatusCode, updated)
	}
	if updated["iconUrl"] != "https://cdn.example/icon.png" || updated["bannerUrl"] != "https://cdn.example/banner.png" || updated["themeColor"] != "#12ab34" {
		t.Errorf("updated branding = %v/%v/%v", updated["iconUrl"], updated["bannerUrl"], updated["themeColor"])
	}

	// The public meta exposes the same values so the client can render them.
	_, meta := post(t, ts, "/api/meta", map[string]any{})
	if meta["iconUrl"] != "https://cdn.example/icon.png" || meta["bannerUrl"] != "https://cdn.example/banner.png" || meta["themeColor"] != "#12ab34" {
		t.Errorf("public meta = %v/%v/%v", meta["iconUrl"], meta["bannerUrl"], meta["themeColor"])
	}

	// Clearing all three works.
	resp, cleared := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"iconUrl": nil, "bannerUrl": nil, "themeColor": nil,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear branding = %d %v", resp.StatusCode, cleared)
	}
	if cleared["iconUrl"] != nil || cleared["bannerUrl"] != nil || cleared["themeColor"] != nil {
		t.Errorf("cleared branding = %v/%v/%v", cleared["iconUrl"], cleared["bannerUrl"], cleared["themeColor"])
	}

	// A malformed theme colour is refused rather than stored.
	if resp, _ := postWithToken(t, ts, "/api/admin/update-meta", "admintoken000000", map[string]any{
		"themeColor": "not-a-colour",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad themeColor = %d, want 400", resp.StatusCode)
	}
}
