package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// authContextKey is the context key under which the authenticated user
// and token are stored.
type authContextKey struct{}

// AuthInfo carries the credentials attached to a request.
type AuthInfo struct {
	User  *account.User
	Token *account.AccessToken
}

// authenticate resolves the credentials of a request.
//
// Misskey accepts the token in the `i` body parameter and in the
// Authorization header (Bearer). The body parameter is checked first.
func (s *Server) authenticate(ctx context.Context, r *http.Request, body map[string]any) (*AuthInfo, error) {
	token := ""
	if body != nil {
		if v, ok := body["i"].(string); ok {
			token = v
		}
	}
	if token == "" {
		if h := r.Header.Get("Authorization"); h != "" {
			if strings.HasPrefix(strings.ToLower(h), "bearer ") {
				token = strings.TrimSpace(h[7:])
			}
		}
	}
	if token == "" {
		return nil, nil
	}

	user, accessToken, err := s.tokens.Resolve(ctx, token)
	if err != nil {
		return nil, err
	}
	return &AuthInfo{User: user, Token: accessToken}, nil
}

// withAuth attaches the authenticated user to the request context.
func withAuth(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authContextKey{}, info)
}

func authFromContext(ctx context.Context) *AuthInfo {
	info, _ := ctx.Value(authContextKey{}).(*AuthInfo)
	return info
}

// requireAdmin authenticates the request and checks that the caller is
// an administrator, writing the appropriate error when not.
// errIDForbidden is Misskey's permission-denied error id.
const errIDForbidden = "f4f0d9f1-3a2e-4a8b-9c7d-2e1b0f6a8c53"

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	info, err := s.authenticate(r.Context(), r, map[string]any{})
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return false
	}
	if !info.User.IsAdmin {
		writeAPIError(w, &APIError{
			Message: "You are not an administrator.", Code: "FORBIDDEN",
			ID: errIDForbidden, Kind: "permission", Status: http.StatusForbidden,
		})
		return false
	}
	return true
}
