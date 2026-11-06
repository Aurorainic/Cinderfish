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
	// Notify is "normal" when every note of the followee should notify the
	// follower, empty otherwise. WithReplies is the "include replies" switch.
	Notify      string
	WithReplies bool
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

// FollowQuery pages one side of the followings table. SinceID/UntilID are
// follow-row cursors; SinceDate/UntilDate bound created_at.
type FollowQuery struct {
	Limit     int
	SinceID   string
	UntilID   string
	SinceDate *time.Time
	UntilDate *time.Time
}

// Followers returns the users who follow userID, newest follow first
// unless a sinceId is given without untilId.
func (s *Service) Followers(ctx context.Context, userID string, q FollowQuery) ([]*Follow, error) {
	return s.listFollows(ctx, "followee_id", "follower_id", userID, q)
}

// Following returns the users userID follows, newest follow first
// unless a sinceId is given without untilId.
func (s *Service) Following(ctx context.Context, userID string, q FollowQuery) ([]*Follow, error) {
	return s.listFollows(ctx, "follower_id", "followee_id", userID, q)
}

// listFollows pages one side of the followings table. anchor is the
// column holding userID, other the column naming the user each page
// carries; both are fixed identifiers, never request input.
func (s *Service) listFollows(ctx context.Context, anchor, other, userID string, q FollowQuery) ([]*Follow, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var sql strings.Builder
	sql.WriteString(`
SELECT f.id, f.created_at, COALESCE(f.notify, ''), f.with_replies,
       u.id, u.username, u.username_lower, u.host, COALESCE(u.password_hash, ''), COALESCE(u.token, ''),
       u.is_admin, u.is_suspended, u.created_at
FROM followings f
JOIN users u ON u.id = f.`)
	sql.WriteString(other)
	sql.WriteString(`
WHERE f.`)
	sql.WriteString(anchor)
	sql.WriteString(` = $1`)

	args := []any{userID}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND f.id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND f.id < $%d", len(args))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		fmt.Fprintf(&sql, " AND f.created_at >= $%d", len(args))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		fmt.Fprintf(&sql, " AND f.created_at <= $%d", len(args))
	}
	// The cursor is the follow row's id, as in Misskey's
	// makePaginationQuery (packages/backend/src/core/QueryService.ts
	// L54-62): a sinceId without untilId pages upward in ascending order,
	// everything else is newest first.
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	sql.WriteString(" ORDER BY f.id " + order + " LIMIT " + strconv.Itoa(limit))

	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: list follows: %w", err)
	}
	defer rows.Close()

	out := []*Follow{}
	for rows.Next() {
		f := Follow{User: &User{}}
		u := f.User
		if err := rows.Scan(&f.ID, &f.CreatedAt, &f.Notify, &f.WithReplies,
			&u.ID, &u.Username, &u.UsernameLower, &u.Host, &u.PasswordHash, &u.Token,
			&u.IsAdmin, &u.IsSuspended, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan follow: %w", err)
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}

// ErrNotFollowing is returned when a follow relationship does not exist.
var ErrNotFollowing = errors.New("account: not following")

// UpdateFollowing applies the per-follow settings. A nil pointer leaves that
// field unchanged, which is what following/update needs: the client sends only
// the switches it flipped.
func (s *Service) UpdateFollowing(ctx context.Context, followerID, followeeID string, notify *string, withReplies *bool) error {
	sets := make([]string, 0, 2)
	args := make([]any, 0, 4)
	args = append(args, followerID, followeeID)
	if notify != nil {
		value := any(nil)
		if *notify != "" && *notify != "none" {
			value = *notify
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("notify = $%d", len(args)))
	}
	if withReplies != nil {
		args = append(args, *withReplies)
		sets = append(sets, fmt.Sprintf("with_replies = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil
	}

	tag, err := s.pool.Exec(ctx, `
UPDATE followings SET `+strings.Join(sets, ", ")+`
WHERE follower_id = $1 AND followee_id = $2
`, args...)
	if err != nil {
		return fmt.Errorf("account: update following: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFollowing
	}
	return nil
}

// UpdateAllFollowing applies the settings to every account the user follows.
func (s *Service) UpdateAllFollowing(ctx context.Context, followerID string, notify *string, withReplies *bool) error {
	sets := make([]string, 0, 2)
	args := make([]any, 0, 3)
	args = append(args, followerID)
	if notify != nil {
		value := any(nil)
		if *notify != "" && *notify != "none" {
			value = *notify
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("notify = $%d", len(args)))
	}
	if withReplies != nil {
		args = append(args, *withReplies)
		sets = append(sets, fmt.Sprintf("with_replies = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
UPDATE followings SET `+strings.Join(sets, ", ")+` WHERE follower_id = $1
`, args...); err != nil {
		return fmt.Errorf("account: update all following: %w", err)
	}
	return nil
}
