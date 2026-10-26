package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"milnfish/internal/config"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		Host: "127.0.0.1", Port: 0, Domain: "fish.example",
		RegistrationMode: "invite", FederationMode: "public",
		Media: config.Media{Backend: "local"},
	}
	return NewServer(cfg)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("response is not JSON: %v (body %q)", err, rec.Body.String())
	}
	return v
}

func TestMetaDetailed(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/meta", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	m := decodeJSON(t, rec)

	if m["name"] != "MilnFish" {
		t.Errorf("meta.name = %v", m["name"])
	}
	// Misskey names the instance URL `uri`, not `instanceUrl`.
	if m["uri"] != "https://fish.example" {
		t.Errorf("meta.uri = %v, want https://fish.example", m["uri"])
	}
	if _, ok := m["instanceUrl"]; ok {
		t.Error("meta must not contain instanceUrl; Misskey uses uri")
	}
	if m["disableRegistration"] != true {
		t.Errorf("invite mode must disable open registration, got %v", m["disableRegistration"])
	}
	if m["federation"] != "all" {
		t.Errorf("meta.federation = %v, want all for public mode", m["federation"])
	}

	features, ok := m["features"].(map[string]any)
	if !ok {
		t.Fatal("detailed meta must include features")
	}
	for _, key := range []string{
		"localTimeline", "globalTimeline", "registration", "emailRequiredForSignup",
		"hcaptcha", "recaptcha", "turnstile", "objectStorage", "serviceWorker", "miauth",
	} {
		if _, ok := features[key]; !ok {
			t.Errorf("features missing %q", key)
		}
	}
	if features["miauth"] != true {
		t.Error("features.miauth must be true")
	}
}

func TestMetaLite(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/meta", bytes.NewBufferString(`{"detail":false}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	m := decodeJSON(t, rec)
	if _, ok := m["features"]; ok {
		t.Error("lite meta must not include features")
	}
	if m["uri"] != "https://fish.example" {
		t.Errorf("meta.uri = %v", m["uri"])
	}
}

func TestPing(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/ping", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	p := decodeJSON(t, rec)
	// Misskey's ping returns the server timestamp, not an echo.
	ts, ok := p["pong"].(float64)
	if !ok {
		t.Fatalf("ping.pong = %v, want a number", p["pong"])
	}
	if ts < 1_600_000_000_000 {
		t.Errorf("ping.pong = %v, want a millisecond timestamp", ts)
	}
}

func TestErrorEnvelope(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/nonexistent", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeJSON(t, rec)
	e, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("404 body missing error envelope: %v", body)
	}
	for _, key := range []string{"message", "code", "id", "kind"} {
		if _, ok := e[key]; !ok {
			t.Errorf("error envelope missing %q", key)
		}
	}
	if e["kind"] != "client" {
		t.Errorf("error.kind = %v, want client", e["kind"])
	}
}

func TestInvalidParamError(t *testing.T) {
	err := invalidParamError(map[string]any{"reason": "bad"})
	if err.Code != "INVALID_PARAM" {
		t.Errorf("code = %q, want INVALID_PARAM", err.Code)
	}
	if err.Kind != "client" {
		t.Errorf("kind = %q, want client", err.Kind)
	}
	if err.ID != "3d81ceae-475f-4600-b2a8-2bc116157532" {
		t.Errorf("id = %q, want Misskey's INVALID_PARAM id", err.ID)
	}
}

func TestFederationModeMapping(t *testing.T) {
	cases := map[string]string{
		"public":    "all",
		"allowlist": "specified",
		"none":      "none",
	}
	for mode, want := range cases {
		cfg := &config.Config{Domain: "d.example", FederationMode: mode}
		srv := NewServer(cfg)
		if got := srv.federationMode(); got != want {
			t.Errorf("federation mode %q maps to %q, want %q", mode, got, want)
		}
	}
}
