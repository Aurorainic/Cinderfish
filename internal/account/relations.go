package account

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Relation is one entry of a mute or block list: the edge row plus the user
// on the other side of it. Misskey packs these as `mutee` / `blockee`.
type Relation struct {
	ID        string
	User      *User
	CreatedAt time.Time
}

// Mute records that muterID no longer wants to see muteeID. Re-muting an
// already muted user is not an error at the storage layer; the handler
// rejects it with ALREADY_MUTING before calling in.
func (s *Service) Mute(ctx context.Context, muterID, muteeID string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO mutes (id, muter_id, mutee_id) VALUES ($1, $2, $3)
ON CONFLICT (muter_id, mutee_id) DO NOTHING
`, idgen.New(), muterID, muteeID)
	if err != nil {
		return fmt.Errorf("account: mute: %w", err)
	}
	return nil
}

// Unmute removes the mute edge. It is a no-op when no edge exists.
func (s *Service) Unmute(ctx context.Context, muterID, muteeID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM mutes WHERE muter_id = $1 AND mutee_id = $2`, muterID, muteeID)
	if err != nil {
		return fmt.Errorf("account: unmute: %w", err)
	}
	return nil
}

// IsMuted reports whether muterID has muted muteeID.
func (s *Service) IsMuted(ctx context.Context, muterID, muteeID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM mutes WHERE muter_id = $1 AND mutee_id = $2)`,
		muterID, muteeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("account: is muted: %w", err)
	}
	return exists, nil
}

// Block records a block and drops every follow edge and pending follow
// request between the two accounts, in both directions. Misskey's
// blocking/create does the same, so a blocked user cannot keep reading the
// blocker's followers-only notes through an existing follow.
func (s *Service) Block(ctx context.Context, blockerID, blockeeID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: block: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO blocks (id, blocker_id, blockee_id) VALUES ($1, $2, $3)
ON CONFLICT (blocker_id, blockee_id) DO NOTHING
`, idgen.New(), blockerID, blockeeID); err != nil {
		return fmt.Errorf("account: block insert: %w", err)
	}
	pair := `($1, $2)`
	if _, err := tx.Exec(ctx, `
DELETE FROM followings
WHERE (follower_id, followee_id) IN (`+pair+`, ($2, $1))
`, blockerID, blockeeID); err != nil {
		return fmt.Errorf("account: block unfollow: %w", err)
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM follow_requests
WHERE (follower_id, followee_id) IN (`+pair+`, ($2, $1))
`, blockerID, blockeeID); err != nil {
		return fmt.Errorf("account: block cancel requests: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("account: block commit: %w", err)
	}
	return nil
}

// Unblock removes the block edge. It does not restore follows.
func (s *Service) Unblock(ctx context.Context, blockerID, blockeeID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blockee_id = $2`, blockerID, blockeeID)
	if err != nil {
		return fmt.Errorf("account: unblock: %w", err)
	}
	return nil
}

// IsBlocked reports whether blockerID has blocked blockeeID.
func (s *Service) IsBlocked(ctx context.Context, blockerID, blockeeID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM blocks WHERE blocker_id = $1 AND blockee_id = $2)`,
		blockerID, blockeeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("account: is blocked: %w", err)
	}
	return exists, nil
}

// Mutes lists a user's muted accounts, newest first.
func (s *Service) Mutes(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Relation, error) {
	return s.relations(ctx, "mutes", "muter_id", "mutee_id", userID, limit, sinceID, untilID)
}

// Blocks lists a user's blocked accounts, newest first.
func (s *Service) Blocks(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Relation, error) {
	return s.relations(ctx, "blocks", "blocker_id", "blockee_id", userID, limit, sinceID, untilID)
}

// relations reads one relation table. The table and column names are fixed
// literals chosen by the exported callers, never user input.
func (s *Service) relations(ctx context.Context, table, ownerCol, targetCol, userID string, limit int, sinceID, untilID string) ([]*Relation, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	query := `
SELECT r.id, r.created_at,
       u.id, u.username, u.username_lower, u.host, COALESCE(u.password_hash, ''),
       COALESCE(u.token, ''), u.is_admin, u.is_suspended, u.created_at
FROM ` + table + ` r JOIN users u ON u.id = r.` + targetCol + `
WHERE r.` + ownerCol + ` = $1`
	args := []any{userID}
	if sinceID != "" {
		args = append(args, sinceID)
		query += fmt.Sprintf(" AND r.id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		query += fmt.Sprintf(" AND r.id < $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY r.id DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: list %s: %w", table, err)
	}
	defer rows.Close()

	var out []*Relation
	for rows.Next() {
		var rel Relation
		var u User
		if err := rows.Scan(&rel.ID, &rel.CreatedAt,
			&u.ID, &u.Username, &u.UsernameLower, &u.Host, &u.PasswordHash,
			&u.Token, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan %s: %w", table, err)
		}
		rel.User = &u
		out = append(out, &rel)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("account: list %s: %w", table, err)
	}
	return out, nil
}
