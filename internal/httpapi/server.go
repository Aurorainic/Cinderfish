// Package httpapi assembles MilnFish's HTTP surface: the Misskey-style
// RPC endpoints under /api, ActivityPub endpoints, and static frontend
// serving. M1 covers /api/meta and /api/ping.
package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"milnfish/internal/config"
)

// Server holds dependencies shared by all handlers.
type Server struct {
	cfg      *config.Config
	startedAt time.Time
	mux      *http.ServeMux
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{cfg: cfg, startedAt: time.Now()}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("POST /api/meta", s.handleMeta)
	s.mux.HandleFunc("POST /api/ping", s.handlePing)
	s.mux.HandleFunc("GET /api/ping", s.handlePing)
	s.mux.HandleFunc("/", notFoundHandler)
	return s
}

// notFoundHandler answers everything that did not match a route with a
// Misskey-style JSON error instead of Go's default plain-text 404, so
// API clients always see the documented error envelope.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, http.StatusNotFound, "UNKNOWN_ENDPOINT",
		fmt.Sprintf("No such endpoint: %s %s", r.Method, r.URL.Path),
		"00000000-0000-0000-0000-000000000001")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// writeJSON emits a Misskey-shaped success response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// writeAPIError emits a Misskey-shaped error:
// {"error":{"code":"...","message":"...","id":"..."}}.
func writeAPIError(w http.ResponseWriter, status int, code, message, id string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"id":      id,
		},
	})
}

// handleMeta answers POST /api/meta. Unauthenticated; safe fields only.
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"maintainerName":        "",
		"maintainerEmail":       "",
		"name":                  "MilnFish",
		"version":               "0.1.0",
		"providesTarball":       false,
		"providesTranslationKeys": false,
		"repositoryUrl":         "",
		"feedbackUrl":           "",
		"impressumUrl":          "",
		"privacyPolicyUrl":      "",
		"disableRegistration":   s.cfg.RegistrationMode != "open",
		"emailRequiredForSignup": false,
		"enableHcaptcha":        false,
		"enableMcaptcha":        false,
		"enableRecaptcha":       false,
		"enableTurnstile":       false,
		"swPublickey":           "",
		"themeColor":            "#86b300",
		"mascotImageUrl":        "/assets/ai.png",
		"bannerUrl":             "",
		"infoImageUrl":          "",
		"serverErrorImageUrl":   "",
		"notFoundImageUrl":      "",
		"instanceUrl":           "https://" + s.cfg.Domain,
		"maxNoteTextLength":     3000,
		"ads":                   []any{},
		"notesPerOneAd":         0,
		"enableEmail":           false,
		"enableServiceWorker":   false,
		"translatorAvailable":   false,
		"serverRules":           []any{},
		"policies":              map[string]any{},
		"mediaProxy":            "",
		"features": map[string]bool{
			"registration": s.cfg.RegistrationMode == "open",
			"emailRequiredForSignup": false,
			"localTimeline":  true,
			"globalTimeline": s.cfg.FederationMode == "public",
		},
	})
}

// handlePing answers POST|GET /api/ping with a body-echo envelope,
// matching Misskey's ping endpoint shape.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pong any `json:"pong"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			writeAPIError(w, http.StatusBadRequest, "INVALID_JSON",
				"Request body is not valid JSON.", "a7df1a0d-1a86-4b8e-8c3e-000000000001")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pong": body.Pong, "uptimeMs": time.Since(s.startedAt).Milliseconds()})
}
