package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// RegistrationMode controls who may create an account.
type RegistrationMode string

const (
	// RegistrationOpen lets anyone sign up.
	RegistrationOpen RegistrationMode = "open"
	// RegistrationInvite requires a valid invitation code.
	RegistrationInvite RegistrationMode = "invite"
	// RegistrationAdmin disables self-service signup entirely.
	RegistrationAdmin RegistrationMode = "admin"
)

// User is a local account.
type User struct {
	ID            string
	Username      string
	UsernameLower string
	Host          *string
	PasswordHash  string
	Token         string
	IsAdmin       bool
	IsSuspended   bool
	CreatedAt     time.Time
}

// RegistrationError describes why a signup was rejected.
type RegistrationError struct {
	Code    string
	Message string
}

func (e *RegistrationError) Error() string { return e.Message }

// Service provides account operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ErrUserNotFound is returned when a user does not exist.
var ErrUserNotFound = errors.New("account: user not found")

// ErrUsernameTaken is returned when a username is already in use.
var ErrUsernameTaken = errors.New("account: username already taken")

// ErrInvalidInvite is returned when an invitation code is missing or
// already used.
var ErrInvalidInvite = errors.New("account: invalid invitation code")

// Register creates a local account.
//
// mode selects the registration policy:
//   - open: the account is created directly;
//   - invite: invitationCode must name a valid, unused ticket;
//   - admin: self-service signup is refused.
//
// The returned token is the user's native API token (16 chars).
func (s *Service) Register(ctx context.Context, mode RegistrationMode, username, password, invitationCode string) (*User, string, error) {
	if err := auth.ValidateUsername(username); err != nil {
		return nil, "", &RegistrationError{Code: "INVALID_USERNAME", Message: "Invalid username."}
	}
	if password == "" {
		return nil, "", &RegistrationError{Code: "INVALID_PASSWORD", Message: "Invalid password."}
	}

	switch mode {
	case RegistrationAdmin:
		return nil, "", &RegistrationError{
			Code:    "REGISTRATION_DISABLED",
			Message: "Registration is disabled on this instance.",
		}
	case RegistrationInvite:
		if err := s.claimInvite(ctx, invitationCode); err != nil {
			return nil, "", err
		}
	}

	user, err := s.Create(ctx, username, password)
	if err != nil {
		return nil, "", err
	}
	return user, user.Token, nil
}

// Create makes a local account regardless of the registration mode. It is
// what an administrator uses to add a user while self-service signup is
// closed, which is the "admin" registration mode.
func (s *Service) Create(ctx context.Context, username, password string) (*User, error) {
	if err := auth.ValidateUsername(username); err != nil {
		return nil, &RegistrationError{Code: "INVALID_USERNAME", Message: "Invalid username."}
	}
	if password == "" {
		return nil, &RegistrationError{Code: "INVALID_PASSWORD", Message: "Invalid password."}
	}

	lower := auth.NormalizeUsername(username)
	exists, err := s.usernameExists(ctx, lower)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	token := auth.GenerateNativeToken()

	user := &User{
		ID:            idgen.New(),
		Username:      username,
		UsernameLower: lower,
		PasswordHash:  hash,
		Token:         token,
		CreatedAt:     time.Now(),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
INSERT INTO users (id, username, username_lower, password_hash, token)
VALUES ($1, $2, $3, $4, $5)
`, user.ID, user.Username, user.UsernameLower, user.PasswordHash, user.Token); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("account: insert user: %w", err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO user_profiles (user_id) VALUES ($1)`, user.ID); err != nil {
		return nil, fmt.Errorf("account: insert profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

// claimInvite reserves an invitation ticket for use, marking it used.
func (s *Service) claimInvite(ctx context.Context, code string) error {
	if code == "" {
		return &RegistrationError{Code: "INVALID_INVITE", Message: "An invitation code is required."}
	}
	ct, err := s.pool.Exec(ctx, `
UPDATE registration_tickets
SET used_at = now()
WHERE code = $1 AND used_at IS NULL AND used_by_id IS NULL
  AND (expires_at IS NULL OR expires_at > now())
`, code)
	if err != nil {
		return fmt.Errorf("account: claim invite: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrInvalidInvite
	}
	return nil
}

// CreateInvite generates an invitation code.
func (s *Service) CreateInvite(ctx context.Context, expiresAt *time.Time) (string, error) {
	code := auth.SecureRndstrLower(16)
	if _, err := s.pool.Exec(ctx, `
INSERT INTO registration_tickets (id, code, expires_at) VALUES ($1, $2, $3)
`, idgen.New(), code, expiresAt); err != nil {
		return "", fmt.Errorf("account: create invite: %w", err)
	}
	return code, nil
}

func (s *Service) usernameExists(ctx context.Context, lower string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE username_lower = $1 AND host IS NULL`, lower).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("account: check username: %w", err)
	}
	return n > 0, nil
}

// UsernameAvailable reports whether a username can be registered, that
// is, whether it is well formed and not already taken.
func (s *Service) UsernameAvailable(ctx context.Context, username string) (bool, error) {
	if err := auth.ValidateUsername(username); err != nil {
		return false, nil
	}
	exists, err := s.usernameExists(ctx, auth.NormalizeUsername(username))
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// GetByUsername loads a local user by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, username, username_lower, host, password_hash, COALESCE(token, ''),
       is_admin, is_suspended, created_at
FROM users WHERE username_lower = $1 AND host IS NULL
`, auth.NormalizeUsername(username))
	return scanUser(row)
}

// GetByID loads a local user by id.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, username, username_lower, host, password_hash, COALESCE(token, ''),
       is_admin, is_suspended, created_at
FROM users WHERE id = $1
`, id)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.UsernameLower, &u.Host, &u.PasswordHash,
		&u.Token, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan user: %w", err)
	}
	return &u, nil
}

// Authenticate verifies a username and password and returns the user.
func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := s.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// ErrInvalidCredentials is returned when a password does not match.
var ErrInvalidCredentials = errors.New("account: invalid credentials")

// RecordSignin appends a login history entry.
func (s *Service) RecordSignin(ctx context.Context, userID, ip string, success bool) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO signins (id, user_id, ip, success) VALUES ($1, $2, $3, $4)
`, idgen.New(), userID, ip, success)
	if err != nil {
		return fmt.Errorf("account: record signin: %w", err)
	}
	return nil
}

// CountUsers returns the number of local accounts.
func (s *Service) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE host IS NULL`).Scan(&n); err != nil {
		return 0, fmt.Errorf("account: count users: %w", err)
	}
	return n, nil
}

// ListLocal returns local accounts for the administration page, oldest
// first, with administrators at the front.
func (s *Service) ListLocal(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, username, username_lower, host, password_hash, COALESCE(token, ''),
       is_admin, is_suspended, created_at
FROM users WHERE host IS NULL
ORDER BY is_admin DESC, created_at ASC
LIMIT $1 OFFSET $2
`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("account: list users: %w", err)
	}
	defer rows.Close()

	out := []*User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetAdmin grants or revokes administrator rights.
func (s *Service) SetAdmin(ctx context.Context, userID string, admin bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE users SET is_admin = $1 WHERE id = $2`, admin, userID)
	if err != nil {
		return fmt.Errorf("account: set admin: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetSuspended blocks or restores a local account. A suspended account
// cannot sign in or use its token.
func (s *Service) SetSuspended(ctx context.Context, userID string, suspended bool) error {
	ct, err := s.pool.Exec(ctx, `UPDATE users SET is_suspended = $1 WHERE id = $2`, suspended, userID)
	if err != nil {
		return fmt.Errorf("account: set suspended: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
