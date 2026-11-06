package account

import (
	"context"
	"fmt"
)

// UserStats is the shape of users/stats. The three note buckets are disjoint
// so a client can show them side by side without double counting.
type UserStats struct {
	NotesCount     int
	RepliesCount   int
	RenotesCount   int
	RepliedCount   int
	RenotedCount   int
	PollVotesCount int
	PollVotedCount int
}

// Stats aggregates one account's activity from the stored rows.
//
// Upstream keeps some of these counters precomputed; Cinderfish derives them,
// which keeps the numbers right after a delete without a second write path.
func (s *Service) Stats(ctx context.Context, userID string) (*UserStats, error) {
	var stats UserStats
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FILTER (WHERE reply_id IS NULL AND renote_id IS NULL),
       count(*) FILTER (WHERE reply_id IS NOT NULL),
       count(*) FILTER (WHERE renote_id IS NOT NULL)
FROM notes WHERE user_id = $1 AND deleted_at IS NULL
`, userID).Scan(&stats.NotesCount, &stats.RepliesCount, &stats.RenotesCount)
	if err != nil {
		return nil, fmt.Errorf("account: note stats: %w", err)
	}

	if err := s.pool.QueryRow(ctx, `
SELECT count(DISTINCT reply_id) FROM notes
WHERE reply_user_id = $1 AND reply_id IS NOT NULL AND deleted_at IS NULL
`, userID).Scan(&stats.RepliedCount); err != nil {
		return nil, fmt.Errorf("account: replied count: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
SELECT count(DISTINCT renote_id) FROM notes
WHERE renote_user_id = $1 AND renote_id IS NOT NULL AND deleted_at IS NULL
`, userID).Scan(&stats.RenotedCount); err != nil {
		return nil, fmt.Errorf("account: renoted count: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM poll_votes WHERE user_id = $1
`, userID).Scan(&stats.PollVotesCount); err != nil {
		return nil, fmt.Errorf("account: poll votes count: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM poll_votes v JOIN notes n ON n.id = v.note_id
WHERE n.user_id = $1 AND n.deleted_at IS NULL
`, userID).Scan(&stats.PollVotedCount); err != nil {
		return nil, fmt.Errorf("account: poll voted count: %w", err)
	}
	return &stats, nil
}
