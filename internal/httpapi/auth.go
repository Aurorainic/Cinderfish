package httpapi

import (
	"context"
	"net/http"
	"strings"

	"milnfish/internal/account"
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
