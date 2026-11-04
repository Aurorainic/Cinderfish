package account

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Follow is one row of a follower or following page: the follow relation
// and the user at its far end. User is the follower for Followers and
// the followee for Following.
type Follow struct {
	ID        string
	CreatedAt time.Time
	User      *User
}

// followVisibility reads the two privacy settings of a user's profile.
func (s *Service) FollowVisibility(ctx context.Context, userID string) (followers, following string, err error) {
	err = s.pool.QueryRow(ctx, `
SELECT followers_visibility, following_visibility FROM user_profiles WHERE user_id = $1
`, userID).Scan(&followers, &following)
	if errors.Is(err, pgx.ErrNoRows) {
		// The columns default to public; a user without a profile row is
		// as open as one that never changed the setting.
		return "public", "public", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("account: read follow visibility: %w", err)
	}
	return followers, following, nil
}

// IsFollowing reports whether followerID follows followeeID.
func (s *Service) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	var follows bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM followings WHERE follower_id = $1 AND followee_id = $2)
`, followerID, followeeID).Scan(&follows)
	if err != nil {
		return false, fmt.Errorf("account: check follow: %w", err)
	}
	return follows, nil
}

// Followers returns the users who follow userID, newest follow first
// unless a sinceId is given without untilId.
func (s *Service) Followers(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Follow, error) {
	return s.listFollows(ctx, "followee_id", "follower_id", userID, limit, sinceID, untilID)
}

// Following returns the users userID follows, newest follow first
// unless a sinceId is given without untilId.
func (s *Service) Following(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Follow, error) {
	return s.listFollows(ctx, "follower_id", "followee_id", userID, limit, sinceID, untilID)
}

// listFollows pages one side of the followings table. anchor is the
// column holding userID, other the column naming the user each page
// carries; both are fixed identifiers, never request input.
func (s *Service) listFollows(ctx context.Context, anchor, other, userID string, limit int, sinceID, untilID string) ([]*Follow, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var q strings.Builder
	q.WriteString(`
SELECT f.id, f.created_at,
       u.id, u.username, u.username_lower, u.host, COALESCE(u.password_hash, ''), COALESCE(u.token, ''),
       u.is_admin, u.is_suspended, u.created_at
FROM followings f
JOIN users u ON u.id = f.`)
	q.WriteString(other)
	q.WriteString(`
WHERE f.`)
	q.WriteString(anchor)
	q.WriteString(` = $1`)

	args := []any{userID}
	if sinceID != "" {
		args = append(args, sinceID)
		fmt.Fprintf(&q, " AND f.id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		fmt.Fprintf(&q, " AND f.id < $%d", len(args))
	}
	// The cursor is the follow row's id, as in Misskey's
	// makePaginationQuery (packages/backend/src/core/QueryService.ts
	// L54-62): a sinceId without untilId pages upward in ascending order,
	// everything else is newest first.
	order := "DESC"
	if sinceID != "" && untilID == "" {
		order = "ASC"
	}
	q.WriteString(" ORDER BY f.id " + order + " LIMIT " + strconv.Itoa(limit))

	rows, err := s.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: list follows: %w", err)
	}
	defer rows.Close()

	out := []*Follow{}
	for rows.Next() {
		f := Follow{User: &User{}}
		u := f.User
		if err := rows.Scan(&f.ID, &f.CreatedAt,
			&u.ID, &u.Username, &u.UsernameLower, &u.Host, &u.PasswordHash, &u.Token,
			&u.IsAdmin, &u.IsSuspended, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan follow: %w", err)
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}
