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
// is looked up in access_tokens by the token's SHA-256 hash. Issued
// tokens are high-entropy random strings, so an unsalted fast hash is the
// right lookup key (the same choice upstream makes by indexing the token
// value directly); storing only the hash keeps a database leak from
// yielding usable credentials.
func (t *TokenService) Resolve(ctx context.Context, token string) (*User, *AccessToken, error) {
	if token == "" {
		return nil, nil, ErrInvalidToken
	}

	if auth.IsNativeUserToken(token) {
		row := t.pool.QueryRow(ctx, `
SELECT `+userColumns+`
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
FROM access_tokens WHERE token_hash = $1
`, auth.HashToken(token))
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

// App is one authorized application, as listed by `i/apps`.
type App struct {
	ID          string
	Name        string
	Description string
	IconURL     string
	Scopes      []string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// Apps lists the third-party tokens issued to a user. Native tokens and the
// user's own session are not access_tokens rows, so they never appear here
// and cannot be revoked through this endpoint.
func (t *TokenService) Apps(ctx context.Context, userID, sort string) ([]*App, error) {
	order := "t.id ASC"
	switch sort {
	case "+createdAt":
		order = "t.created_at DESC"
	case "-createdAt":
		order = "t.created_at ASC"
	case "+lastUsedAt":
		order = "t.last_used_at DESC NULLS LAST"
	case "-lastUsedAt":
		order = "t.last_used_at ASC NULLS LAST"
	}

	rows, err := t.pool.Query(ctx, `
SELECT t.id, COALESCE(t.name, a.name, ''), COALESCE(t.description, a.description, ''),
       COALESCE(t.icon_url, ''),
       CASE WHEN a.id IS NOT NULL THEN a.permission ELSE t.scopes END,
       t.created_at, t.last_used_at
FROM access_tokens t LEFT JOIN apps a ON a.id = t.app_id
WHERE t.user_id = $1 ORDER BY `+order, userID)
	if err != nil {
		return nil, fmt.Errorf("account: list apps: %w", err)
	}
	defer rows.Close()

	var out []*App
	for rows.Next() {
		var app App
		var scopesJSON []byte
		if err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.IconURL,
			&scopesJSON, &app.CreatedAt, &app.LastUsedAt); err != nil {
			return nil, fmt.Errorf("account: scan app: %w", err)
		}
		if len(scopesJSON) > 0 {
			if err := json.Unmarshal(scopesJSON, &app.Scopes); err != nil {
				return nil, fmt.Errorf("account: unmarshal app scopes: %w", err)
			}
		}
		out = append(out, &app)
	}
	return out, rows.Err()
}

// RevokeOwned deletes one of the user's own access tokens. It reports whether
// a row was removed; a token belonging to someone else is never touched.
func (t *TokenService) RevokeOwned(ctx context.Context, userID, id string) (bool, error) {
	tag, err := t.pool.Exec(ctx, `DELETE FROM access_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, fmt.Errorf("account: revoke own token: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// FindUnfetchedBySession returns the token issued for a MiAuth session that the
// requesting application has not collected yet, together with its owner. A
// session that was never approved (or was already collected) reports
// ErrInvalidToken.
func (t *TokenService) FindUnfetchedBySession(ctx context.Context, session string) (*AccessToken, *User, error) {
	if session == "" {
		return nil, nil, ErrInvalidToken
	}
	row := t.pool.QueryRow(ctx, `
SELECT id, token, user_id, session, name, scopes, created_at
FROM access_tokens WHERE session = $1 AND NOT fetched
ORDER BY created_at DESC LIMIT 1
`, session)
	at, err := scanAccessToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidToken
	}
	if err != nil {
		return nil, nil, err
	}
	user, err := t.users.GetByID(ctx, at.UserID)
	if err != nil {
		return nil, nil, err
	}
	return at, user, nil
}

// MarkFetched records that the application collected the token. The second
// poll of the same session must answer "not ready" rather than leak the token
// again.
func (t *TokenService) MarkFetched(ctx context.Context, id string) error {
	if _, err := t.pool.Exec(ctx, `UPDATE access_tokens SET fetched = TRUE WHERE id = $1`, id); err != nil {
		return fmt.Errorf("account: mark token fetched: %w", err)
	}
	return nil
}
