package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// AccessToken is an issued API token.
type AccessToken struct {
	ID        string
	Token     string
	UserID    string
	Session   *string
	Name      *string
	Scopes    []string
	CreatedAt time.Time
}

// ErrInvalidToken is returned when a token cannot be resolved.
var ErrInvalidToken = errors.New("account: invalid token")

// TokenService issues and resolves API tokens.
type TokenService struct {
	pool  *pgxpool.Pool
	users *Service
}

// NewTokenService returns a TokenService backed by the given pool.
func NewTokenService(pool *pgxpool.Pool) *TokenService {
	return &TokenService{pool: pool, users: NewService(pool)}
}

// Issue creates an access token for a user.
func (t *TokenService) Issue(ctx context.Context, userID, token, session, name, description, iconURL string, scopes []string) (*AccessToken, error) {
	if scopes == nil {
		scopes = []string{}
	}
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, fmt.Errorf("account: marshal scopes: %w", err)
	}

	at := &AccessToken{
		ID:        idgen.New(),
		Token:     token,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	if session != "" {
		at.Session = &session
	}
	if name != "" {
		at.Name = &name
	}

	if _, err := t.pool.Exec(ctx, `
INSERT INTO access_tokens (id, token, token_hash, user_id, session, name, description, icon_url, scopes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, at.ID, at.Token, auth.HashToken(at.Token), at.UserID, at.Session, at.Name,
		nullable(description), nullable(iconURL), scopesJSON); err != nil {
		return nil, fmt.Errorf("account: insert access token: %w", err)
	}
	return at, nil
}

// nullable maps an empty string to a SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Resolve finds the access token and its owner.
//
// It mirrors Misskey's AuthenticateService: a 16-character token is
// treated as a native user token looked up in users.token, anything else
// is looked up in access_tokens by exact token value.
func (t *TokenService) Resolve(ctx context.Context, token string) (*User, *AccessToken, error) {
	if token == "" {
		return nil, nil, ErrInvalidToken
	}

	if auth.IsNativeUserToken(token) {
		row := t.pool.QueryRow(ctx, `
SELECT id, username, username_lower, host, password_hash, COALESCE(token, ''),
       is_admin, is_suspended, created_at
FROM users WHERE token = $1
`, token)
		user, err := scanUser(row)
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidToken
		}
		if err != nil {
			return nil, nil, err
		}
		return user, nil, nil
	}

	row := t.pool.QueryRow(ctx, `
SELECT id, token, user_id, session, name, scopes, created_at
FROM access_tokens WHERE token = $1
`, token)
	at, err := scanAccessToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidToken
	}
	if err != nil {
		return nil, nil, err
	}

	_, _ = t.pool.Exec(ctx, `UPDATE access_tokens SET last_used_at = now() WHERE id = $1`, at.ID)

	user, err := t.users.GetByID(ctx, at.UserID)
	if err != nil {
		return nil, nil, err
	}
	return user, at, nil
}

func scanAccessToken(row pgx.Row) (*AccessToken, error) {
	var at AccessToken
	var scopesJSON []byte
	err := row.Scan(&at.ID, &at.Token, &at.UserID, &at.Session, &at.Name, &scopesJSON, &at.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("account: scan access token: %w", err)
	}
	if len(scopesJSON) > 0 {
		if err := json.Unmarshal(scopesJSON, &at.Scopes); err != nil {
			return nil, fmt.Errorf("account: unmarshal scopes: %w", err)
		}
	}
	return &at, nil
}

// Revoke deletes an access token.
func (t *TokenService) Revoke(ctx context.Context, id string) error {
	_, err := t.pool.Exec(ctx, `DELETE FROM access_tokens WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("account: revoke access token: %w", err)
	}
	return nil
}
