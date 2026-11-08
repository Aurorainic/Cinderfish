package httpapi

import (
	"context"
	"errors"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/streaming"
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
	rows, err := s.pool.Query(ctx, "SELECT followee_id, with_replies FROM followings WHERE follower_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]streaming.Follow)
	for rows.Next() {
		var id string
		var withReplies bool
		if err := rows.Scan(&id, &withReplies); err != nil {
			return nil, err
		}
		out[id] = streaming.Follow{WithReplies: withReplies}
	}
	return out, rows.Err()
}

// LoadListMembers implements streaming.ListLoader for the Server. A list
// that does not exist or is not owned by userID reports ok = false,
// which rejects the channel subscription the way upstream's init throw
// does; an ownership mismatch reports not-found rather than forbidden so
// other people's private list ids are not enumerable.
func (s *Server) LoadListMembers(ctx context.Context, listID, userID string) (map[string]streaming.ListMembership, bool, error) {
	if s.accounts == nil {
		return nil, false, nil
	}
	if _, err := s.accounts.OwnList(ctx, listID, userID); err != nil {
		return nil, false, nil
	}
	memberships, err := s.accounts.ListMemberships(ctx, listID, account.UserListMembershipQuery{Limit: 100})
	if err != nil {
		return nil, false, err
	}
	out := make(map[string]streaming.ListMembership, len(memberships))
	for _, m := range memberships {
		out[m.UserID] = streaming.ListMembership{WithReplies: m.WithReplies}
	}
	return out, true, nil
}

// AntennaOwned implements streaming.AntennaLoader for the Server.
func (s *Server) AntennaOwned(ctx context.Context, antennaID, viewerID string) (bool, error) {
	if s.accounts == nil {
		return false, nil
	}
	_, err := s.accounts.GetAntenna(ctx, antennaID, viewerID)
	if errors.Is(err, account.ErrAntennaNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ClipVisible implements streaming.ClipLoader for the Server: the owner
// and readers of a public clip may watch its stream.
func (s *Server) ClipVisible(ctx context.Context, clipID, viewerID string) (bool, error) {
	if s.accounts == nil {
		return false, nil
	}
	_, err := s.accounts.GetClip(ctx, clipID, viewerID)
	if errors.Is(err, account.ErrClipNotFound) {
		return false, nil
	}
	return err == nil, err
}
