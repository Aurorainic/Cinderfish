package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"milnfish/internal/account"
	"milnfish/internal/auth"
)

// OAuth error ids, mirroring Misskey's OAuth2 error responses.
const (
	errIDUnknownOAuthEndpoint = "aa49e620-26cb-4e28-aad6-8cbcb58db147"
	errIDOAuthInvalidRequest  = "b3c2e1d0-4f5a-4b6c-8d9e-0f1a2b3c4d5e"
	errIDOAuthInvalidGrant    = "c4d3e2f1-5a6b-4c7d-9e0f-1a2b3c4d5e6f"
)

// oauthGrant is an issued authorization code awaiting redemption.
type oauthGrant struct {
	Code          string
	ClientID      string
	UserID        string
	RedirectURI   string
	CodeChallenge string
	Scopes        []string
	Used          bool
}

// oauthMetadata is the RFC 8414 document served at
// /.well-known/oauth-authorization-server.
func (s *Server) oauthMetadata() map[string]any {
	base := "https://" + s.cfg.Domain
	return map[string]any{
		"issuer":                                         base,
		"authorization_endpoint":                         base + "/oauth/authorize",
		"token_endpoint":                                 base + "/oauth/token",
		"scopes_supported":                               []string{"read", "write", "follow", "upload"},
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code"},
		"service_documentation":                          "https://misskey-hub.net",
		"code_challenge_methods_supported":               []string{"S256"},
		"authorization_response_iss_parameter_supported": true,
	}
}

// handleOAuthMetadata serves GET /.well-known/oauth-authorization-server.
func (s *Server) handleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.oauthMetadata())
}

// handleOAuthAuthorize serves GET /oauth/authorize.
//
// The full flow needs an interactive consent page; without one the
// endpoint reports that a login is required, which is what Misskey's
// decision step would eventually confirm.
func (s *Server) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	if clientID == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "client_id must be provided", "")
		return
	}
	if redirectURI == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "redirect_uri must be provided", "")
		return
	}
	if responseType != "code" {
		s.writeOAuthError(w, http.StatusBadRequest, "unsupported_response_type", "Only the code response type is supported", "")
		return
	}
	if codeChallenge == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code_challenge parameter is required", "")
		return
	}
	if codeChallengeMethod != "S256" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "code_challenge_method parameter must be set as S256", "")
		return
	}

	// The request is well formed. A logged in session is needed to
	// complete the consent step; without one the flow cannot continue.
	info, err := s.authenticate(r.Context(), r, nil)
	if err != nil || info == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>MilnFish</title>` +
			`<p>Sign in to authorize this application.</p>`))
		return
	}

	// Issue the authorization code directly. The interactive consent page
	// is part of the frontend work (M6).
	code := auth.GenerateGrantCode()
	scopes := q["scope"]
	if len(scopes) == 0 {
		scopes = []string{"read"}
	}
	if _, err := s.pool.Exec(r.Context(), `
INSERT INTO oauth_grant_codes (code, client_id, user_id, redirect_uri, code_challenge, scopes)
VALUES ($1, $2, $3, $4, $5, $6)
`, code, clientID, info.User.ID, redirectURI, codeChallenge, mustJSON(scopes)); err != nil {
		s.writeOAuthError(w, http.StatusInternalServerError, "server_error", "Could not issue the authorization code", "")
		return
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri", "")
		return
	}
	values := target.Query()
	values.Set("code", code)
	if state := q.Get("state"); state != "" {
		values.Set("state", state)
	}
	values.Set("iss", "https://"+s.cfg.Domain)
	target.RawQuery = values.Encode()

	http.Redirect(w, r, target.String(), http.StatusFound)
}

// handleOAuthToken implements POST /oauth/token.
//
// The token endpoint uses form encoding, so the body is parsed with
// ParseForm rather than the JSON helper used by the RPC endpoints.
func (s *Server) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	// ParseForm only reads the body for form content types, and consumes
	// it otherwise. Decide on the content type before touching the body.
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body map[string]any
		if err := decodeOptionalJSON(r, &body); err != nil {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Could not parse the request body", "")
			return
		}
		form := make(url.Values)
		for _, key := range []string{"grant_type", "code", "client_id", "redirect_uri", "code_verifier"} {
			if v, ok := body[key].(string); ok {
				form.Set(key, v)
			}
		}
		r.PostForm = form
	} else {
		if err := r.ParseForm(); err != nil {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Could not parse the request body", "")
			return
		}
	}

	grantType := r.PostForm.Get("grant_type")
	if grantType == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "grant_type is required", "")
		return
	}
	if grantType != "authorization_code" {
		s.writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "Only the authorization_code grant is supported", "")
		return
	}

	code := r.PostForm.Get("code")
	clientID := r.PostForm.Get("client_id")
	redirectURI := r.PostForm.Get("redirect_uri")
	codeVerifier := r.PostForm.Get("code_verifier")

	if code == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "grant request is invalid", "")
		return
	}

	grant, err := s.redeemGrant(r.Context(), code)
	if err != nil {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "grant request is invalid", "")
		return
	}

	// RFC 6749 4.1.3: the client and redirect URI must match the request.
	if clientID != grant.ClientID || redirectURI != grant.RedirectURI {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "grant request is invalid", "")
		return
	}
	if codeVerifier == "" {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "grant request is invalid", "")
		return
	}
	if !auth.VerifyCodeChallenge(codeVerifier, grant.CodeChallenge) {
		s.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "grant request is invalid", "")
		return
	}

	accessToken := auth.GenerateOAuthToken()
	tokens := account.NewTokenService(s.pool)
	if _, err := tokens.Issue(r.Context(), grant.UserID, accessToken, "", clientID, "OAuth 2.0", "", grant.Scopes); err != nil {
		s.writeOAuthError(w, http.StatusInternalServerError, "server_error", "Could not issue the access token", "")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"scope":        strings.Join(grant.Scopes, " "),
	})
}

// redeemGrant atomically consumes an authorization code.
func (s *Server) redeemGrant(ctx context.Context, code string) (*oauthGrant, error) {
	row := s.pool.QueryRow(ctx, `
SELECT code, client_id, user_id, redirect_uri, code_challenge, scopes, used
FROM oauth_grant_codes WHERE code = $1
`, code)

	var g oauthGrant
	var scopesJSON []byte
	if err := row.Scan(&g.Code, &g.ClientID, &g.UserID, &g.RedirectURI, &g.CodeChallenge, &scopesJSON, &g.Used); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("oauth: scan grant: %w", err)
	}
	if g.Used {
		// A code must not be usable twice.
		return nil, errors.New("oauth: grant code already used")
	}
	if err := json.Unmarshal(scopesJSON, &g.Scopes); err != nil {
		return nil, fmt.Errorf("oauth: unmarshal scopes: %w", err)
	}

	ct, err := s.pool.Exec(ctx, `UPDATE oauth_grant_codes SET used = TRUE WHERE code = $1 AND used = FALSE`, code)
	if err != nil {
		return nil, fmt.Errorf("oauth: mark grant used: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, errors.New("oauth: grant code already used")
	}
	return &g, nil
}

// writeOAuthError emits an OAuth 2.0 style error response.
func (s *Server) writeOAuthError(w http.ResponseWriter, status int, code, description, state string) {
	writeJSON(w, status, map[string]any{
		"error":             code,
		"error_description": description,
		"state":             state,
	})
}

// mustJSON marshals a value, returning nil on failure.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}
