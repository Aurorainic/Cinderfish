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

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/auth"
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

// oauthAuthorizeParams is a validated authorization request. Only the code
// flow with PKCE is accepted, which is what Firebase/Misskey clients use.
type oauthAuthorizeParams struct {
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	State         string
	Scopes        []string
}

// oauthRequestError is a validation failure that maps onto an OAuth 2.0
// error response.
type oauthRequestError struct {
	code        string
	description string
}

func (e *oauthRequestError) Error() string { return e.description }

// parseOAuthAuthorize validates the query parameters of an authorization
// request. Both the GET endpoint and the SPA's consent call use it, so the
// rules cannot drift between them.
func parseOAuthAuthorize(q url.Values) (*oauthAuthorizeParams, *oauthRequestError) {
	p := &oauthAuthorizeParams{
		ClientID:      q.Get("client_id"),
		RedirectURI:   q.Get("redirect_uri"),
		CodeChallenge: q.Get("code_challenge"),
		State:         q.Get("state"),
	}
	switch {
	case p.ClientID == "":
		return nil, &oauthRequestError{"invalid_request", "client_id must be provided"}
	case p.RedirectURI == "":
		return nil, &oauthRequestError{"invalid_request", "redirect_uri must be provided"}
	case q.Get("response_type") != "code":
		return nil, &oauthRequestError{"unsupported_response_type", "Only the code response type is supported"}
	case p.CodeChallenge == "":
		return nil, &oauthRequestError{"invalid_request", "code_challenge parameter is required"}
	case q.Get("code_challenge_method") != "S256":
		return nil, &oauthRequestError{"invalid_request", "code_challenge_method parameter must be set as S256"}
	}

	// scope may be repeated or space separated; both forms appear in the wild.
	for _, raw := range q["scope"] {
		for _, scope := range strings.Fields(raw) {
			p.Scopes = append(p.Scopes, scope)
		}
	}
	if len(p.Scopes) == 0 {
		p.Scopes = []string{"read"}
	}
	if err := validateRedirectURI(p.ClientID, p.RedirectURI); err != nil {
		return nil, err
	}
	return p, nil
}

// validateRedirectURI pins the redirect to the client's own origin, the
// rule Misskey's OAuth provider enforces by discovering the client's
// registered redirect URIs from client_id (OAuth2ProviderService.ts:
// `clientInfo.redirectUris.includes(redirectUriValue)`). Cinderfish has no
// client discovery, so an https client_id URL stands in for the
// registration: the redirect must land on the same host. Without this an
// attacker could stage a consent page with their own redirect_uri and
// collect the victim's authorization code.
func validateRedirectURI(clientID, redirectURI string) *oauthRequestError {
	client, err := url.Parse(clientID)
	if err != nil || client.Scheme != "https" && client.Scheme != "http" || client.Host == "" {
		return &oauthRequestError{"invalid_request", "client_id must be an https URL identifying the client"}
	}
	target, err := url.Parse(redirectURI)
	if err != nil || (target.Scheme != "https" && target.Scheme != "http") || target.Host == "" {
		return &oauthRequestError{"invalid_request", "redirect_uri must be an absolute http(s) URL"}
	}
	if !strings.EqualFold(target.Host, client.Host) {
		return &oauthRequestError{"invalid_request", "redirect_uri must be on the client_id host"}
	}
	if target.Fragment != "" {
		return &oauthRequestError{"invalid_request", "redirect_uri must not carry a fragment"}
	}
	return nil
}

// issueOAuthGrant records an authorization code for the user and returns it.
func (s *Server) issueOAuthGrant(ctx context.Context, userID string, p *oauthAuthorizeParams) (string, error) {
	code := auth.GenerateGrantCode()
	if _, err := s.pool.Exec(ctx, `
INSERT INTO oauth_grant_codes (code, client_id, user_id, redirect_uri, code_challenge, scopes)
VALUES ($1, $2, $3, $4, $5, $6)
`, code, p.ClientID, userID, p.RedirectURI, p.CodeChallenge, mustJSON(p.Scopes)); err != nil {
		return "", err
	}
	return code, nil
}

// oauthRedirectURL builds the client redirect carrying the code and state.
func (s *Server) oauthRedirectURL(p *oauthAuthorizeParams, code string) (string, error) {
	target, err := url.Parse(p.RedirectURI)
	if err != nil {
		return "", err
	}
	values := target.Query()
	values.Set("code", code)
	if p.State != "" {
		values.Set("state", p.State)
	}
	values.Set("iss", "https://"+s.cfg.Domain)
	target.RawQuery = values.Encode()
	return target.String(), nil
}

// handleOAuthAuthorize serves GET /oauth/authorize.
//
// Two callers share this path. A client that already has a credential (an
// Authorization header) gets the authorization code immediately, which keeps
// the previous server-to-server behaviour. A browser without one is served the
// SPA, which renders the consent page and then calls POST /api/oauth/authorize
// with the signed-in user's token.
func (s *Server) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	params, paramErr := parseOAuthAuthorize(r.URL.Query())
	if paramErr != nil {
		s.writeOAuthError(w, http.StatusBadRequest, paramErr.code, paramErr.description, "")
		return
	}

	info, err := s.authenticate(r.Context(), r, nil)
	if err == nil && info != nil {
		code, err := s.issueOAuthGrant(r.Context(), info.User.ID, params)
		if err != nil {
			s.writeOAuthError(w, http.StatusInternalServerError, "server_error", "Could not issue the authorization code", "")
			return
		}
		target, err := s.oauthRedirectURL(params, code)
		if err != nil {
			s.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid redirect_uri", "")
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	// No credential: this is a browser asking for the consent page.
	if s.web != nil {
		s.web.serve(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>CinderFish</title>` +
		`<p>Sign in to authorize this application.</p>`))
}

// handleOAuthAuthorizeAPI implements POST /api/oauth/authorize, the call the
// consent page makes once the user approves.
//
// The response is the client redirect URL rather than a 302, because the SPA
// has to leave the API call's JSON world and navigate the browser itself.
// This endpoint is Cinderfish-specific: Misskey's clients never call it.
func (s *Server) handleOAuthAuthorizeAPI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID            string          `json:"client_id"`
		RedirectURI         string          `json:"redirect_uri"`
		ResponseType        string          `json:"response_type"`
		CodeChallenge       string          `json:"code_challenge"`
		CodeChallengeMethod string          `json:"code_challenge_method"`
		State               string          `json:"state"`
		Scope               json.RawMessage `json:"scope"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	q := url.Values{}
	q.Set("client_id", body.ClientID)
	q.Set("redirect_uri", body.RedirectURI)
	q.Set("response_type", body.ResponseType)
	q.Set("code_challenge", body.CodeChallenge)
	q.Set("code_challenge_method", body.CodeChallengeMethod)
	q.Set("state", body.State)
	if len(body.Scope) > 0 {
		var list []string
		if err := json.Unmarshal(body.Scope, &list); err == nil {
			for _, scope := range list {
				q.Add("scope", scope)
			}
		} else {
			var single string
			if err := json.Unmarshal(body.Scope, &single); err == nil {
				q.Add("scope", single)
			}
		}
	}
	params, paramErr := parseOAuthAuthorize(q)
	if paramErr != nil {
		writeAPIError(w, &APIError{
			Message: paramErr.description, Code: "INVALID_PARAM",
			ID: errIDOAuthInvalidRequest, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	code, err := s.issueOAuthGrant(r.Context(), info.User.ID, params)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	target, err := s.oauthRedirectURL(params, code)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Invalid redirect_uri.", Code: "INVALID_PARAM",
			ID: errIDOAuthInvalidRequest, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"redirect": target})
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
