package account

import (
	"context"
	"fmt"
	"time"

	"github.com/Aurorainic/cinderfish/internal/auth"
)

// SigninRecord is one row of the login history returned by i/signin-history.
type SigninRecord struct {
	ID        string
	CreatedAt time.Time
	IP        string
	Success   bool
}

// ChangePassword verifies the current password and replaces the stored hash.
func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	ok, err := s.CheckPassword(ctx, userID, current)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash); err != nil {
		return fmt.Errorf("account: change password: %w", err)
	}
	return nil
}

// RegenerateToken replaces the account's 16-character native token and
// returns the new one. The password confirms the caller knows it, since the
// old token is about to stop working.
func (s *Service) RegenerateToken(ctx context.Context, userID, password string) (string, error) {
	ok, err := s.CheckPassword(ctx, userID, password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidCredentials
	}
	token := auth.GenerateNativeToken()
	if _, err := s.pool.Exec(ctx, `UPDATE users SET token = $2 WHERE id = $1`, userID, token); err != nil {
		return "", fmt.Errorf("account: regenerate token: %w", err)
	}
	return token, nil
}

// SigninHistory returns the account's login attempts, newest first, using
// Misskey's id cursor pagination.
func (s *Service) SigninHistory(ctx context.Context, userID string, limit int, untilID, sinceID string) ([]SigninRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	query := `SELECT id, created_at, COALESCE(ip, ''), success FROM signins WHERE user_id = $1`
	args := []any{userID}
	if untilID != "" {
		args = append(args, untilID)
		query += fmt.Sprintf(" AND id < $%d", len(args))
	}
	if sinceID != "" {
		args = append(args, sinceID)
		query += fmt.Sprintf(" AND id > $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: signin history: %w", err)
	}
	defer rows.Close()

	out := make([]SigninRecord, 0, limit)
	for rows.Next() {
		var r SigninRecord
		if err := rows.Scan(&r.ID, &r.CreatedAt, &r.IP, &r.Success); err != nil {
			return nil, fmt.Errorf("account: scan signin: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account: signin history: %w", err)
	}
	return out, nil
}
