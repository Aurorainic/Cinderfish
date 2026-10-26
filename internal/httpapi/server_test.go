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
	}
	return NewServer(cfg)
}

func TestMeta(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/meta", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("meta response is not JSON: %v", err)
	}
	if m["name"] != "MilnFish" {
		t.Errorf("meta.name = %v", m["name"])
	}
	if m["instanceUrl"] != "https://fish.example" {
		t.Errorf("meta.instanceUrl = %v", m["instanceUrl"])
	}
	if m["disableRegistration"] != true {
		t.Errorf("invite mode must disable open registration, got %v", m["disableRegistration"])
	}
}

func TestPing(t *testing.T) {
	srv := testServer(t)

	body := bytes.NewBufferString(`{"pong": 42}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ping", body)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("ping response is not JSON: %v", err)
	}
	if p["pong"] != float64(42) {
		t.Errorf("ping.pong = %v, want 42", p["pong"])
	}
	if _, ok := p["uptimeMs"]; !ok {
		t.Error("ping response missing uptimeMs")
	}
}

func TestPingInvalidJSON(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ping", bytes.NewBufferString("{oops"))
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var e struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("error response is not JSON: %v", err)
	}
	if e.Error.Code != "INVALID_JSON" {
		t.Errorf("error.code = %q, want INVALID_JSON", e.Error.Code)
	}
}

func TestUnknownEndpointMisskeyError(t *testing.T) {
	srv := testServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nonexistent", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var e map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("404 response is not Misskey-style JSON error: %v", err)
	}
	if _, ok := e["error"]; !ok {
		t.Errorf("404 body missing error envelope: %v", e)
	}
}
