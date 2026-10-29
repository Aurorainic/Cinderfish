package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
)

// Benchmarks run against the real router and a real database, because the
// number this project tracks is "the same instance got faster or slower",
// not a synthetic handler figure. Run with:
//
//	CINDERFISH_TEST_DATABASE_URL=... go test ./internal/httpapi \
//	  -run '^$' -bench . -benchmem -benchtime 2s
func benchServer(b *testing.B) (*httptest.Server, string, string) {
	b.Helper()
	pool := testPool(b)
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		Media: config.Media{Backend: "local"},
	}
	srv := httpapi.NewServerWithDB(cfg, pool)
	srv.AttachFederation()
	ts := httptest.NewServer(srv)
	b.Cleanup(ts.Close)

	body := `{"username":"bench","password":"password123"}`
	resp, err := http.Post(ts.URL+"/api/signup", "application/json", bytes.NewBufferString(body))
	if err != nil {
		b.Fatalf("signup: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		b.Fatalf("decode signup: %v", err)
	}
	token, _ := out["token"].(string)
	id, _ := out["id"].(string)
	if token == "" || id == "" {
		b.Fatalf("signup returned no token/id: %v", out)
	}
	return ts, token, id
}

func benchPOST(b *testing.B, ts *httptest.Server, path, token string, body any) {
	b.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		b.Fatalf("encode: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(payload))
	if err != nil {
		b.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		b.Fatalf("%s: %v", path, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b.Fatalf("%s: status %d", path, resp.StatusCode)
	}
}

func seedNotes(b *testing.B, ts *httptest.Server, token string, n int) {
	b.Helper()
	for i := 0; i < n; i++ {
		benchPOST(b, ts, "/api/notes/create", token, map[string]any{
			"text": fmt.Sprintf("bench seed note %d with a little text", i),
		})
	}
}

func BenchmarkMeta(b *testing.B) {
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		Media: config.Media{Backend: "local"},
	}
	ts := httptest.NewServer(httpapi.NewServer(cfg))
	b.Cleanup(ts.Close)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchPOST(b, ts, "/api/meta", "", map[string]any{})
	}
}

func BenchmarkLocalTimeline(b *testing.B) {
	ts, token, _ := benchServer(b)
	seedNotes(b, ts, token, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchPOST(b, ts, "/api/notes/local-timeline", "", map[string]any{"limit": 20})
	}
}

func BenchmarkNotesCreate(b *testing.B) {
	ts, token, _ := benchServer(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchPOST(b, ts, "/api/notes/create", token, map[string]any{
			"text": fmt.Sprintf("benchmark note %d", i),
		})
	}
}

func BenchmarkNotifications(b *testing.B) {
	ts, token, _ := benchServer(b)
	seedNotes(b, ts, token, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchPOST(b, ts, "/api/notifications", token, map[string]any{"limit": 20})
	}
}

func BenchmarkActorJSON(b *testing.B) {
	ts, _, id := benchServer(b)
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/users/"+id, nil)
	if err != nil {
		b.Fatalf("request: %v", err)
	}
	req.Header.Set("Accept", "application/activity+json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := ts.Client().Do(req)
		if err != nil {
			b.Fatalf("actor: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("actor status = %d", resp.StatusCode)
		}
	}
}

func BenchmarkWebIndex(b *testing.B) {
	dir := b.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><div id=app></div>"), 0o644); err != nil {
		b.Fatal(err)
	}
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "open", FederationMode: "public",
		WebDir: dir,
		Media:  config.Media{Backend: "local"},
	}
	ts := httptest.NewServer(httpapi.NewServer(cfg))
	b.Cleanup(ts.Close)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := ts.Client().Get(ts.URL + "/timeline")
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("status = %d", resp.StatusCode)
		}
	}
}
