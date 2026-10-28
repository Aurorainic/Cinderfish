package httpapi

import (
	"context"

	"milnfish/internal/account"
	"milnfish/internal/streaming"
)

// streamingAuth adapts the token service to the streaming Authenticator.
type streamingAuth struct {
	tokens *account.TokenService
}

// AuthenticateStream resolves a streaming token.
//
// Misskey requires a read:account permission on app tokens used for
// streaming; native user tokens carry every permission implicitly.
func (a streamingAuth) AuthenticateStream(ctx context.Context, token string) (*streaming.StreamUser, error) {
	user, appToken, err := a.tokens.Resolve(ctx, token)
	if err != nil || user == nil {
		return nil, streaming.ErrInvalidToken
	}
	if appToken != nil && !hasReadAccount(appToken.Scopes) {
		// Misskey reports a missing permission as an authentication
		// failure on the streaming handshake.
		return nil, streaming.ErrInvalidToken
	}
	if user.IsSuspended {
		return nil, streaming.ErrSuspended
	}
	return &streaming.StreamUser{ID: user.ID}, nil
}

// hasReadAccount reports whether a scope list grants read:account.
func hasReadAccount(scopes []string) bool {
	for _, s := range scopes {
		if s == "read:account" {
			return true
		}
	}
	return false
}

// LoadFollows implements streaming.ViewerLoader for the Server.
func (s *Server) LoadFollows(ctx context.Context, userID string) (map[string]streaming.Follow, error) {
	rows, err := s.pool.Query(ctx, "SELECT followee_id FROM followings WHERE follower_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]streaming.Follow)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		// followings has no withReplies column yet; false matches
		// Misskey's default for a fresh follow.
		out[id] = streaming.Follow{WithReplies: false}
	}
	return out, rows.Err()
}
