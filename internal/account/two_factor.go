package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/auth"
)

// ErrTwoFactorNotStarted is returned when i/2fa/done is called without a
// pending secret, i.e. i/2fa/register was never run or its secret expired.
var ErrTwoFactorNotStarted = errors.New("account: two-factor registration was not started")

// CheckPassword reports whether password matches the user's stored hash.
func (s *Service) CheckPassword(ctx context.Context, userID, password string) (bool, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(password_hash, '') FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return false, fmt.Errorf("account: read password hash: %w", err)
	}
	return auth.CheckPassword(hash, password), nil
}

// TwoFactorState is the stored TOTP state for one account. Enabled dictates
// signin's second step; TempSecret is the pending registration secret.
type TwoFactorState struct {
	Enabled    bool
	Secret     string
	TempSecret string
}

// GetTwoFactorState loads the TOTP state, reading a missing profile row as
// "not enabled".
func (s *Service) GetTwoFactorState(ctx context.Context, userID string) (TwoFactorState, error) {
	var secret, temp *string
	var enabled bool
	err := s.pool.QueryRow(ctx, `
SELECT two_factor_secret, two_factor_temp_secret, two_factor_enabled
FROM user_profiles WHERE user_id = $1
`, userID).Scan(&secret, &temp, &enabled)
	if err != nil {
		// A user without a profile row simply has no second factor.
		if isNoProfile(err) {
			return TwoFactorState{}, nil
		}
		return TwoFactorState{}, fmt.Errorf("account: get two-factor state: %w", err)
	}
	return TwoFactorState{
		Enabled:    enabled,
		Secret:     derefString(secret),
		TempSecret: derefString(temp),
	}, nil
}

// BeginTwoFactor stores a pending secret for the account, replacing any
// earlier pending registration.
func (s *Service) BeginTwoFactor(ctx context.Context, userID, secret string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO user_profiles (user_id, two_factor_temp_secret) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET two_factor_temp_secret = EXCLUDED.two_factor_temp_secret
`, userID, secret)
	if err != nil {
		return fmt.Errorf("account: store pending two-factor secret: %w", err)
	}
	return nil
}

// EnableTwoFactor promotes the pending secret to the active one. It fails
// with ErrTwoFactorNotStarted when no registration is pending.
func (s *Service) EnableTwoFactor(ctx context.Context, userID string) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE user_profiles
SET two_factor_secret = two_factor_temp_secret,
    two_factor_temp_secret = NULL,
    two_factor_enabled = TRUE
WHERE user_id = $1 AND two_factor_temp_secret IS NOT NULL
`, userID)
	if err != nil {
		return fmt.Errorf("account: enable two-factor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTwoFactorNotStarted
	}
	return nil
}

// DisableTwoFactor clears the active and pending secrets.
func (s *Service) DisableTwoFactor(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE user_profiles
SET two_factor_secret = NULL,
    two_factor_temp_secret = NULL,
    two_factor_enabled = FALSE,
    use_password_less_login = FALSE
WHERE user_id = $1
`, userID)
	if err != nil {
		return fmt.Errorf("account: disable two-factor: %w", err)
	}
	return nil
}

func isNoProfile(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
