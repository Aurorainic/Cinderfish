// Package httpapi assembles MilnFish's HTTP surface: the Misskey-style
// RPC endpoints under /api, ActivityPub endpoints, and static frontend
// serving. M1 covers /api/meta and /api/ping.
package httpapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/account"
	"milnfish/internal/config"
)

// Server holds dependencies shared by all handlers.
type Server struct {
	cfg       *config.Config
	startedAt time.Time
	mux       *http.ServeMux

	pool     *pgxpool.Pool
	accounts *account.Service
	tokens   *account.TokenService
}

// NewServer returns a Server with no database backing. It serves the
// endpoints that do not need persistence.
func NewServer(cfg *config.Config) *Server {
	s := &Server{cfg: cfg, startedAt: time.Now()}
	s.mux = http.NewServeMux()
	s.registerRoutes()
	return s
}

// NewServerWithDB returns a Server wired to the given database pool.
func NewServerWithDB(cfg *config.Config, pool *pgxpool.Pool) *Server {
	s := NewServer(cfg)
	s.pool = pool
	s.accounts = account.NewService(pool)
	s.tokens = account.NewTokenService(pool)
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /api/meta", s.handleMeta)
	s.mux.HandleFunc("POST /api/ping", s.handlePing)
	s.mux.HandleFunc("GET /api/ping", s.handlePing)

	s.mux.HandleFunc("POST /api/signup", s.handleSignup)
	s.mux.HandleFunc("POST /api/signin", s.handleSignin)
	s.mux.HandleFunc("POST /api/username/available", s.handleUsernameAvailable)
	s.mux.HandleFunc("POST /api/i", s.handleI)
	s.mux.HandleFunc("POST /api/users/show", s.handleUsersShow)
	s.mux.HandleFunc("POST /api/miauth/gen-token", s.handleMiAuthGenToken)

	s.mux.HandleFunc("GET /.well-known/oauth-authorization-server", s.handleOAuthMetadata)
	s.mux.HandleFunc("GET /oauth/authorize", s.handleOAuthAuthorize)
	s.mux.HandleFunc("POST /oauth/token", s.handleOAuthToken)

	s.mux.HandleFunc("/", notFoundHandler)
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

// APIError mirrors Misskey's ApiError: the response body is
// {"error":{"message","code","id","kind"[, "info"]}} and kind is always
// present (client, server or permission).
type APIError struct {
	Message string
	Code    string
	ID      string
	Kind    string // client | server | permission
	Info    any
	HasInfo bool
	Status  int
}

func (e *APIError) Error() string { return e.Message }

// Well-known error ids, copied from Misskey so clients and operators can
// recognise them (packages/backend/src/server/api/error.ts).
const (
	errIDInvalidParam    = "3d81ceae-475f-4600-b2a8-2bc116157532"
	errIDAuthentication  = "b0a7f5f8-dc2f-4171-b91f-de88ad238e14"
	errIDInternal        = "5d37dbcb-891e-41ca-a3d6-e690c97775ac"
	errIDNoSuchEndpoint  = "fc9c4a0b-f3a1-4b0e-9a8c-6f7f3d0f6a12"
	errIDRegistrationOff = "4d371c9a-3a0f-4c8b-9b8e-2f9c0a1d3e45"
)

// writeAPIError emits the Misskey error envelope.
func writeAPIError(w http.ResponseWriter, e *APIError) {
	body := map[string]any{
		"message": e.Message,
		"code":    e.Code,
		"id":      e.ID,
		"kind":    e.Kind,
	}
	if e.HasInfo {
		body["info"] = e.Info
	}
	status := e.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Misskey"`)
	}
	writeJSON(w, status, map[string]any{"error": body})
}

// invalidParamError reports a request that failed parameter validation.
func invalidParamError(info any) *APIError {
	return &APIError{
		Message: "Invalid param.",
		Code:    "INVALID_PARAM",
		ID:      errIDInvalidParam,
		Kind:    "client",
		Info:    info,
		HasInfo: true,
		Status:  http.StatusBadRequest,
	}
}

// writeAuthenticationError emits Misskey's authentication failure
// response: HTTP 401 with a WWW-Authenticate header.
func writeAuthenticationError(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="Misskey", error="invalid_token", error_description="Authentication failed. Please ensure your token is correct."`)
	writeAPIError(w, &APIError{
		Message: "Authentication failed. Please ensure your token is correct.",
		Code:    "AUTHENTICATION_FAILED",
		ID:      errIDAuthentication,
		Kind:    "client",
		Status:  http.StatusUnauthorized,
	})
}

// clientIP returns the remote address of the request.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// notFoundHandler answers everything that did not match a route with a
// Misskey-style JSON error instead of Go's default plain-text 404, so
// API clients always see the documented error envelope.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, &APIError{
		Message: fmt.Sprintf("No such endpoint: %s %s", r.Method, r.URL.Path),
		Code:    "NO_SUCH_ENDPOINT",
		ID:      errIDNoSuchEndpoint,
		Kind:    "client",
		Status:  http.StatusNotFound,
	})
}
