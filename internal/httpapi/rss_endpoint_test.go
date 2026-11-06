package httpapi_test

import (
	"net/http"
	"testing"
)

// TestFetchRSSRejectsUnsafeTargets pins the SSRF guard on the feed fetch: the
// endpoint and every redirect go through the same check as drive uploads.
func TestFetchRSSRejectsUnsafeTargets(t *testing.T) {
	ts := newTestServer(t, "open")

	for _, target := range []string{
		"http://127.0.0.1:8080/feed.xml",
		"http://localhost/feed.xml",
		"ftp://example.com/feed.xml",
		"file:///etc/passwd",
		"",
	} {
		resp, body := post(t, ts, "/api/fetch-rss", map[string]any{"url": target})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%q status = %d, want 400 (%v)", target, resp.StatusCode, body)
		}
	}

	// The GET alias takes the URL from the query string.
	resp, _ := postRaw(t, ts, "/api/fetch-rss?url=http%3A%2F%2F127.0.0.1%2Ffeed.xml", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("GET loopback status = %d, want 400", resp.StatusCode)
	}
}
