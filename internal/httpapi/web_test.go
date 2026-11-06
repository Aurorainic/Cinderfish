package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/config"
)

func webServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><div id=app>cinderfish-shell</div>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('cinderfish')"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "invite", FederationMode: "public",
		WebDir: dir,
		Media:  config.Media{Backend: "local"},
	}
	srv := NewServer(cfg)
	if srv.web == nil {
		t.Fatal("static site was not attached")
	}
	return srv, dir
}

func TestWebServesRoot(t *testing.T) {
	srv, _ := webServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cinderfish-shell") {
		t.Errorf("body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("index Cache-Control = %q, want no-cache", got)
	}
}

func TestWebMissingAssetIsNotFound(t *testing.T) {
	srv, _ := webServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/index-stale.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "cinderfish-shell") {
		t.Error("missing asset was answered with index.html")
	}
}

func TestWebFallsBackToIndexForClientRoutes(t *testing.T) {
	srv, _ := webServer(t)
	// Deep links that the client owns: the settings children and the admin
	// sections are query/segment-driven routes, so a reload on one has to
	// serve the shell and let vue-router rebuild the same state.
	for _, path := range []string{
		"/timeline",
		"/settings",
		"/settings/appearance",
		"/settings/account",
		"/admin",
		"/admin?tab=users",
		"/@" + "alice",
		"/@" + "alice?tab=followers",
		"/notes/abc123",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), "cinderfish-shell") {
			t.Errorf("%s: did not fall back to index.html", path)
		}
	}
}

func TestWebServesHashedAssets(t *testing.T) {
	srv, _ := webServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cinderfish") {
		t.Errorf("body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", got)
	}
}

func TestWebDoesNotSwallowProtocolPaths(t *testing.T) {
	srv, _ := webServer(t)
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/nope", nil),
		httptest.NewRequest(http.MethodPost, "/api/meta", nil),
		httptest.NewRequest(http.MethodGet, "/.well-known/webfinger?resource=acct:a@b", nil),
		httptest.NewRequest(http.MethodGet, "/users/abc", nil),
	} {
		req.Header.Set("Accept", "text/html")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "cinderfish-shell") {
			t.Errorf("%s %s was answered by the SPA", req.Method, req.URL.Path)
		}
	}
}

func TestWebYieldsToActivityPubForJSON(t *testing.T) {
	srv, _ := webServer(t)
	req := httptest.NewRequest(http.MethodGet, "/notes/abc123", nil)
	req.Header.Set("Accept", "application/activity+json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "cinderfish-shell") {
		t.Fatal("ActivityPub request was answered with the SPA")
	}
}

func TestWebHeadRequest(t *testing.T) {
	srv, _ := webServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD body = %q, want empty", rec.Body.String())
	}
}

func TestWebDisabledWithoutBuild(t *testing.T) {
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "invite", FederationMode: "public",
		WebDir: filepath.Join(t.TempDir(), "missing"),
	}
	srv := NewServer(cfg)
	if srv.web != nil {
		t.Fatal("static site attached from a missing directory")
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 JSON", rec.Code)
	}
}

func TestPathClean(t *testing.T) {
	cases := map[string]string{
		"":                  "/",
		"/":                 "/",
		"/a/b":              "/a/b",
		"/a/../b":           "/b",
		"/../../etc/passwd": "/etc/passwd",
		"//a///b":           "/a/b",
	}
	for in, want := range cases {
		if got := pathClean(in); got != want {
			t.Errorf("pathClean(%q) = %q, want %q", in, got, want)
		}
	}
}
