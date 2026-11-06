package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SetKnownAs replaces an account's alias list. The URIs are stored in the
// order given; duplicates are the caller's problem, matching Misskey's
// storing of the resolved set.
func (s *Service) SetKnownAs(ctx context.Context, userID string, uris []string) error {
	if uris == nil {
		uris = []string{}
	}
	enc, err := json.Marshal(uris)
	if err != nil {
		return fmt.Errorf("account: encode known-as: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
UPDATE users SET also_known_as = $2::jsonb WHERE id = $1
`, userID, enc); err != nil {
		return fmt.Errorf("account: set known-as: %w", err)
	}
	return nil
}

// MarkMoved records that an account migrated to the actor at targetURI and
// adds that actor to the alias list, so the target's own Move validation
// finds this account's URI in it — the same bookkeeping Firefish performs.
func (s *Service) MarkMoved(ctx context.Context, userID, targetURI string) error {
	if targetURI == "" {
		return fmt.Errorf("account: empty move target")
	}

	var known json.RawMessage
	err := s.pool.QueryRow(ctx, `SELECT also_known_as FROM users WHERE id = $1`, userID).Scan(&known)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("account: load known-as: %w", err)
	}
	var aliases []string
	if len(known) > 0 {
		if err := json.Unmarshal(known, &aliases); err != nil {
			return fmt.Errorf("account: decode known-as: %w", err)
		}
	}
	found := false
	for _, a := range aliases {
		if a == targetURI {
			found = true
			break
		}
	}
	if !found {
		aliases = append(aliases, targetURI)
	}
	enc, err := json.Marshal(aliases)
	if err != nil {
		return fmt.Errorf("account: encode known-as: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
UPDATE users SET moved_to_uri = $2, also_known_as = $3::jsonb WHERE id = $1
`, userID, targetURI, enc); err != nil {
		return fmt.Errorf("account: mark moved: %w", err)
	}
	return nil
}
