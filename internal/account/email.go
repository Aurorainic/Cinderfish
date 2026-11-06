package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// ErrEmailTaken is returned when an address already backs another account.
var ErrEmailTaken = errors.New("account: email already in use")

// ErrNoEmail is returned when an operation needs a stored address and the
// account has none.
var ErrNoEmail = errors.New("account: no email address on file")

// ErrInvalidEmailChallenge covers an unknown, expired or already-used link.
var ErrInvalidEmailChallenge = errors.New("account: invalid or expired email link")

// Email purposes. Each issued link is only usable by the flow that minted it,
// so a reset link cannot be replayed as a verification link.
const (
	EmailPurposeVerify = "verify"
	EmailPurposeReset  = "reset"
)

// EmailState is the address on file and whether its owner proved control
// of it.
type EmailState struct {
	Email    string
	Verified bool
}

// ValidateEmail reports whether s looks like a deliverable address. It is
// deliberately stricter than net/mail: one @, a dotted domain, no spaces.
func ValidateEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 254 || strings.ContainsAny(s, " \t\r\n,;<>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") ||
		strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return false
	}
	return true
}

// EmailState reads the account's stored address.
func (s *Service) EmailState(ctx context.Context, userID string) (EmailState, error) {
	var email *string
	var verified bool
	err := s.pool.QueryRow(ctx,
		`SELECT email, email_verified FROM user_profiles WHERE user_id = $1`, userID).
		Scan(&email, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmailState{}, nil
	}
	if err != nil {
		return EmailState{}, fmt.Errorf("account: read email: %w", err)
	}
	return EmailState{Email: derefString(email), Verified: verified}, nil
}

// EmailAvailable reports whether an address can be attached to userID. The
// public availability endpoint passes an empty userID.
func (s *Service) EmailAvailable(ctx context.Context, addr, userID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM user_profiles
    WHERE lower(email) = lower($1) AND user_id <> $2
)
`, addr, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("account: email availability: %w", err)
	}
	return !exists, nil
}

// SetEmail stores a new address and clears the verified flag: the owner has
// to prove control of the new mailbox before it counts as verified.
func (s *Service) SetEmail(ctx context.Context, userID, addr string) error {
	available, err := s.EmailAvailable(ctx, addr, userID)
	if err != nil {
		return err
	}
	if !available {
		return ErrEmailTaken
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO user_profiles (user_id, email, email_verified) VALUES ($1, $2, FALSE)
ON CONFLICT (user_id) DO UPDATE SET email = EXCLUDED.email, email_verified = FALSE
`, userID, addr)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		return fmt.Errorf("account: set email: %w", err)
	}
	return nil
}

// MarkEmailVerified records that the account owner followed a verification
// link sent to the address currently on file.
func (s *Service) MarkEmailVerified(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE user_profiles SET email_verified = TRUE WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("account: mark email verified: %w", err)
	}
	return nil
}

// EmailChallenge is one issued verification or reset link.
type EmailChallenge struct {
	ID        string
	UserID    string
	Purpose   string
	Email     string
	Code      string
	ExpiresAt time.Time
}

// IssueEmailChallenge mints a single-use code bound to userID and addr,
// retiring any unused code of the same purpose first so only the newest
// link works.
func (s *Service) IssueEmailChallenge(ctx context.Context, userID, purpose, addr string, ttl time.Duration) (*EmailChallenge, error) {
	code, err := randomCode()
	if err != nil {
		return nil, err
	}
	challenge := &EmailChallenge{
		ID:        idgen.New(),
		UserID:    userID,
		Purpose:   purpose,
		Email:     addr,
		Code:      code,
		ExpiresAt: time.Now().Add(ttl).UTC(),
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM email_challenges WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL`,
		userID, purpose); err != nil {
		return nil, fmt.Errorf("account: retire email challenge: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO email_challenges (id, user_id, purpose, email, code, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, challenge.ID, challenge.UserID, challenge.Purpose, challenge.Email, challenge.Code, challenge.ExpiresAt); err != nil {
		return nil, fmt.Errorf("account: insert email challenge: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return challenge, nil
}

// ConsumeEmailChallenge marks a code used and returns the row it named.
// Expired, unknown and already-used codes all report ErrInvalidEmailChallenge
// so the caller cannot probe the table.
func (s *Service) ConsumeEmailChallenge(ctx context.Context, purpose, code string) (*EmailChallenge, error) {
	var c EmailChallenge
	err := s.pool.QueryRow(ctx, `
UPDATE email_challenges SET used_at = now()
WHERE code = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
RETURNING id, user_id, purpose, email, code, expires_at
`, code, purpose).Scan(&c.ID, &c.UserID, &c.Purpose, &c.Email, &c.Code, &c.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidEmailChallenge
	}
	if err != nil {
		return nil, fmt.Errorf("account: consume email challenge: %w", err)
	}
	return &c, nil
}

// LatestEmailChallengeTime reports when a purpose last issued a link, so
// handlers can throttle repeated sends without another table.
func (s *Service) LatestEmailChallengeTime(ctx context.Context, userID, purpose string) (time.Time, bool, error) {
	var at time.Time
	err := s.pool.QueryRow(ctx, `
SELECT created_at FROM email_challenges
WHERE user_id = $1 AND purpose = $2
ORDER BY created_at DESC LIMIT 1
`, userID, purpose).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("account: latest email challenge: %w", err)
	}
	return at, true, nil
}

// ResetPassword replaces the password of the account a reset link named.
// The caller has already proven control of the mailbox, so no current
// password is required.
func (s *Service) ResetPassword(ctx context.Context, userID, password string) error {
	if password == "" {
		return errors.New("account: empty password")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, hash)
	if err != nil {
		return fmt.Errorf("account: reset password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidEmailChallenge
	}
	return nil
}

// randomCode returns 32 random bytes as hex: long enough to be unguessable
// in a URL, short enough to paste.
func randomCode() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("account: random code: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
