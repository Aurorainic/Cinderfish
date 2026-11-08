package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/idgen"
	"github.com/Aurorainic/cinderfish/internal/media"
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
	// MovedToURI is the AP id of the account this one migrated to, empty
	// when it has not moved.
	MovedToURI string
	// AlsoKnownAs lists the actor URIs this account declares as aliases of
	// itself. The Move protocol reads it on the target side.
	AlsoKnownAs []string
}

// userColumns is the projection scanUser expects. Queries that join the
// users table alias it as u, so they spell the same list with a prefix.
const userColumns = `id, username, username_lower, host, COALESCE(password_hash, ''),
       COALESCE(token, ''), is_admin, is_suspended, created_at,
       COALESCE(moved_to_uri, ''), COALESCE(also_known_as, '[]'::jsonb)`

// RegistrationError describes why a signup was rejected.
type RegistrationError struct {
	Code    string
	Message string
}

func (e *RegistrationError) Error() string { return e.Message }

// Service provides account operations.
type Service struct {
	pool *pgxpool.Pool

	// drive resolves avatar files for UpdateProfile; attached by the
	// server wiring through AttachDrive.
	drive *media.DriveService

	// reservedUsernames returns the instance's preserved list (lowercase
	// entries). The server wires it to the live instance settings. Nil
	// means no names are reserved.
	reservedUsernames func() []string
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// SetReservedUsernames attaches the runtime preserved-usernames provider.
// Misskey keeps the list in meta.preservedUsernames and refuses signup and
// username/available for entries (SignupService.ts L91-95).
func (s *Service) SetReservedUsernames(fn func() []string) {
	s.reservedUsernames = fn
}

// isReserved reports whether a local username is on the preserved list.
func (s *Service) isReserved(lower string) bool {
	if s.reservedUsernames == nil {
		return false
	}
	for _, name := range s.reservedUsernames() {
		if strings.EqualFold(name, lower) {
			return true
		}
	}
	return false
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
	// Misskey answers USED_USERNAME for preserved names on both signup paths
	// (SignupService.ts L91-95, SignupApiService.ts L186-190).
	if s.isReserved(auth.NormalizeUsername(username)) {
		return nil, "", &RegistrationError{Code: "USED_USERNAME", Message: "This username is reserved."}
	}

	claimed := false
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
		claimed = true
	}

	user, err := s.Create(ctx, username, password)
	if err != nil {
		// A failed signup must not burn the code: the user can fix the
		// username and retry with the same invitation, as Misskey's
		// SignupApiService does when it releases the ticket.
		if claimed {
			s.releaseInvite(ctx, invitationCode)
		}
		return nil, "", err
	}
	if claimed {
		// Misskey stamps the ticket with usedBy once the account exists
		// (SignupApiService); the admin console shows who redeemed a code.
		s.bindInvite(ctx, invitationCode, user.ID)
	}
	return user, user.Token, nil
}

// Create makes a local account regardless of the registration mode. It is
// what an administrator uses to add a user while self-service signup is
// closed, which is the "admin" registration mode.
//
// Preserved usernames are not checked here: the list exists so users cannot
// claim a name, and an administrator choosing one deliberately is the way to
// use it (Misskey's admin/accounts/create does the same).
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

// releaseInvite makes a claimed code usable again after a failed signup. The
// conditional UPDATE leaves a code that somehow got bound to a user alone.
func (s *Service) releaseInvite(ctx context.Context, code string) {
	if code == "" {
		return
	}
	_, _ = s.pool.Exec(ctx, `
UPDATE registration_tickets SET used_at = NULL WHERE code = $1 AND used_by_id IS NULL
`, code)
}

// bindInvite records which account redeemed a code after its signup
// succeeded. Best effort: the signup is already committed, and the only loss
// if this fails is an unattributed row in the admin console.
func (s *Service) bindInvite(ctx context.Context, code, userID string) {
	if code == "" || userID == "" {
		return
	}
	_, _ = s.pool.Exec(ctx, `
UPDATE registration_tickets SET used_by_id = $2 WHERE code = $1 AND used_by_id IS NULL
`, code, userID)
}

// Invite is one registration ticket as the admin console sees it.
// Misskey packs the same shape (Packed<'InviteCode'>).
type Invite struct {
	ID        string
	Code      string
	CreatedAt time.Time
	ExpiresAt *time.Time
	CreatedBy *User
	UsedBy    *User
	UsedAt    *time.Time
	Used      bool
}

// InviteFilter selects which tickets ListInvites returns. Type and sort
// follow Misskey's admin/invite/list parameter values.
type InviteFilter struct {
	Limit  int
	Offset int
	Type   string // all | unused | used | expired
	Sort   string // +createdAt | -createdAt | +usedAt | -usedAt
}

// ErrInviteNotFound is returned when a ticket id does not exist.
var ErrInviteNotFound = errors.New("account: invite not found")

// ListInvites returns registration tickets for the admin console.
func (s *Service) ListInvites(ctx context.Context, filter InviteFilter) ([]Invite, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	conditions := []string{}
	args := []any{}
	switch filter.Type {
	case "unused":
		conditions = append(conditions, `t.used_by_id IS NULL`)
	case "used":
		conditions = append(conditions, `t.used_by_id IS NOT NULL`)
	case "expired":
		args = append(args, time.Now())
		conditions = append(conditions, `t.expires_at < $1`)
	case "", "all":
	default:
		return nil, fmt.Errorf("account: invalid invite type %q", filter.Type)
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Misskey sorts on the ticket id, whose leading base36 timestamp makes
	// +createdAt / -createdAt id-order sorts; usedAt sorts keep NULLs last
	// for the descending branch, matching the upstream query builder.
	orderBy := "t.id DESC"
	switch filter.Sort {
	case "-createdAt":
		orderBy = "t.id ASC"
	case "+usedAt":
		orderBy = "t.used_at DESC NULLS LAST"
	case "-usedAt":
		orderBy = "t.used_at ASC NULLS FIRST"
	}

	args = append(args, limit, filter.Offset)
	// The limit/offset placeholders follow whatever the filter added.
	// Building the whole statement here keeps the positional list in one
	// place; PostgreSQL counts placeholders from $1 regardless of where the
	// LIMIT clause sits.
	sql := `
SELECT t.id, t.code, t.created_at, t.expires_at, t.used_at, t.used_by_id IS NOT NULL,
       cu.id, COALESCE(cu.username, ''), COALESCE(cu.username_lower, ''), cu.host,
       COALESCE(cu.is_admin, false), COALESCE(cu.is_suspended, false), COALESCE(cu.created_at, t.created_at),
       uu.id, COALESCE(uu.username, ''), COALESCE(uu.username_lower, ''), uu.host,
       COALESCE(uu.is_admin, false), COALESCE(uu.is_suspended, false), COALESCE(uu.created_at, t.created_at)
FROM registration_tickets t
LEFT JOIN users cu ON cu.id = t.created_by_id
LEFT JOIN users uu ON uu.id = t.used_by_id
` + where + `
ORDER BY ` + orderBy + `
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("account: list invites: %w", err)
	}
	defer rows.Close()

	out := make([]Invite, 0, limit)
	for rows.Next() {
		var inv Invite
		var createdBy, usedBy User
		var createdByID, usedByID *string
		if err := rows.Scan(
			&inv.ID, &inv.Code, &inv.CreatedAt, &inv.ExpiresAt, &inv.UsedAt, &inv.Used,
			&createdByID, &createdBy.Username, &createdBy.UsernameLower, &createdBy.Host,
			&createdBy.IsAdmin, &createdBy.IsSuspended, &createdBy.CreatedAt,
			&usedByID, &usedBy.Username, &usedBy.UsernameLower, &usedBy.Host,
			&usedBy.IsAdmin, &usedBy.IsSuspended, &usedBy.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("account: scan invite: %w", err)
		}
		if createdByID != nil {
			createdBy.ID = *createdByID
			inv.CreatedBy = &createdBy
		}
		if usedByID != nil {
			usedBy.ID = *usedByID
			inv.UsedBy = &usedBy
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account: list invites: %w", err)
	}
	return out, nil
}

// DeleteInvite removes one registration ticket.
func (s *Service) DeleteInvite(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM registration_tickets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("account: delete invite: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrInviteNotFound
	}
	return nil
}

// CreateInvite generates an invitation code and returns the stored ticket.
// createdByID records the administrator who issued it (Misskey's
// RegistrationTicket.createdById); an empty id stores NULL for callers that
// predate the column.
func (s *Service) CreateInvite(ctx context.Context, createdByID string, expiresAt *time.Time) (Invite, error) {
	code := auth.SecureRndstrLower(16)
	now := time.Now()
	inv := Invite{Code: code, CreatedAt: now, ExpiresAt: expiresAt}
	row := s.pool.QueryRow(ctx, `
INSERT INTO registration_tickets (id, code, created_by_id, expires_at, created_at)
VALUES ($1, $2, NULLIF($3, ''), $4, $5)
RETURNING id, created_at
`, idgen.New(), code, createdByID, expiresAt, now)
	if err := row.Scan(&inv.ID, &inv.CreatedAt); err != nil {
		return Invite{}, fmt.Errorf("account: create invite: %w", err)
	}
	return inv, nil
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
// is, whether it is well formed, not reserved and not already taken.
func (s *Service) UsernameAvailable(ctx context.Context, username string) (bool, error) {
	if err := auth.ValidateUsername(username); err != nil {
		return false, nil
	}
	if s.isReserved(auth.NormalizeUsername(username)) {
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
SELECT `+userColumns+`
FROM users WHERE username_lower = $1 AND host IS NULL
`, auth.NormalizeUsername(username))
	return scanUser(row)
}

// GetByID loads any user by id, local or remote. A remote actor has no
// password, so the hash is coalesced: the column is nullable and packing
// a remote user must not fail.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
SELECT `+userColumns+`
FROM users WHERE id = $1
`, id)
	return scanUser(row)
}

// GetByURI loads any user by its actor URI. The account migration flow
// stores URIs (not ids) in also_known_as and moved_to_uri, so packing those
// needs a lookup that accepts both a local and a remote actor.
func (s *Service) GetByURI(ctx context.Context, uri string) (*User, error) {
	if uri == "" {
		return nil, ErrUserNotFound
	}
	row := s.pool.QueryRow(ctx, `
SELECT `+userColumns+`
FROM users WHERE uri = $1
`, uri)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.UsernameLower, &u.Host, &u.PasswordHash,
		&u.Token, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.MovedToURI, &u.AlsoKnownAs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan user: %w", err)
	}
	return &u, nil
}

// dummyPasswordHash is the bcrypt hash the Authenticate path compares
// against when the username does not exist, so both failure paths cost one
// bcrypt verification and a timing probe cannot enumerate accounts. The
// response still distinguishes the two cases with Misskey's error ids
// (noSuchUser vs incorrectPassword, SigninApiService.ts); the ids are
// public knowledge and username availability is leaked by
// username/available anyway, so only the timing channel is closed here.
var dummyPasswordHash, dummyHashErr = bcrypt.GenerateFromPassword([]byte("cinderfish-dummy-password"), auth.BcryptCost)

// Authenticate verifies a username and password and returns the user. A
// missing username burns the same bcrypt work as a real check before the
// caller answers with Misskey's noSuchUser error.
func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := s.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			if dummyHashErr == nil {
				_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			}
			return nil, ErrUserNotFound
		}
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
SELECT `+userColumns+`
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
