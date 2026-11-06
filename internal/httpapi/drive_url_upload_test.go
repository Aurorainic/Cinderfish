package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestDriveUploadFromURLRejectsUnsafeTargets covers the SSRF guard on
// drive/files/upload-from-url: only http(s) to a public address is fetched, and
// every refusal is a 400 with an explanatory message.
func TestDriveUploadFromURLRejectsUnsafeTargets(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t)+"a")

	cases := []struct {
		name string
		url  string
		want string
	}{
		{"file scheme", "file:///etc/passwd", "url must be an http(s) URL"},
		{"ftp scheme", "ftp://example.com/x", "url must be an http(s) URL"},
		{"no host", "http://", "url must be an http(s) URL"},
		{"loopback", "http://127.0.0.1/file", "private address"},
		{"localhost", "http://localhost/file", "private address"},
		{"private range", "http://192.168.1.1/file", "private address"},
		{"unresolvable", "http://w63-no-such-host.invalid/file", "could not resolve the host"},
	}
	for _, tc := range cases {
		resp, body := postWithToken(t, ts, "/api/drive/files/upload-from-url", token, map[string]any{"url": tc.url})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (%v)", tc.name, resp.StatusCode, body)
			continue
		}
		// invalidParamError puts the reason in `info`, not in the message.
		errObj, _ := body["error"].(map[string]any)
		info, _ := errObj["info"].(string)
		if !strings.Contains(info, tc.want) {
			t.Errorf("%s: info = %q, want it to contain %q", tc.name, info, tc.want)
		}
	}

	// A missing url and an anonymous caller are their own branches.
	if resp, _ := postWithToken(t, ts, "/api/drive/files/upload-from-url", token, map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing url = %d, want 400", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/drive/files/upload-from-url", map[string]any{"url": "http://example.com/x"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", resp.StatusCode)
	}

	// A folder that is not the caller's is refused before any fetch happens.
	if resp, body := postWithToken(t, ts, "/api/drive/files/upload-from-url", token, map[string]any{
		"url": "http://example.com/x", "folderId": "00mw000000000000000",
	}); resp.StatusCode == http.StatusOK {
		t.Errorf("foreign folder accepted: %d %v", resp.StatusCode, body)
	}
}
