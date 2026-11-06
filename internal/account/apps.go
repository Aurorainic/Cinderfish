package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/auth"
	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// AppRecord is a registered third-party application, as created by app/create.
// The secret is what the client uses to poll an auth session; it is never
// shown to anyone but the app's owner.
type AppRecord struct {
	ID          string
	UserID      *string
	Name        string
	Description string
	Permission  []string
	CallbackURL *string
	Secret      string
	CreatedAt   time.Time
}

// AuthSession is one pending approval: a user opens /auth/<token>, and the
// client polls auth/session/userkey with the app secret.
type AuthSession struct {
	ID     string
	AppID  string
	Token  string
	UserID *string
}

// ErrAppNotFound is returned when an application secret or id is unknown.
var ErrAppNotFound = errors.New("account: app not found")

// ErrSessionNotFound is returned when an auth session token is unknown.
var ErrSessionNotFound = errors.New("account: auth session not found")

// ErrSessionPending is returned when a session has not been approved yet.
var ErrSessionPending = errors.New("account: auth session pending")

// CreateApp registers an application and returns it with its secret.
func (t *TokenService) CreateApp(ctx context.Context, userID, name, description string, permission []string, callbackURL string) (*AppRecord, error) {
	if name == "" {
		return nil, fmt.Errorf("account: app name is required")
	}
	if permission == nil {
		permission = []string{}
	}
	perm, err := json.Marshal(permission)
	if err != nil {
		return nil, fmt.Errorf("account: encode app permission: %w", err)
	}

	app := &AppRecord{
		ID:          idgen.New(),
		Name:        name,
		Description: description,
		Permission:  permission,
		Secret:      auth.GenerateMiAuthToken(),
		CreatedAt:   time.Now(),
	}
	if userID != "" {
		app.UserID = &userID
	}
	if callbackURL != "" {
		app.CallbackURL = &callbackURL
	}

	if _, err := t.pool.Exec(ctx, `
INSERT INTO apps (id, user_id, name, description, permission, callback_url, secret, created_at)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)
`, app.ID, app.UserID, app.Name, app.Description, perm, app.CallbackURL, app.Secret, app.CreatedAt); err != nil {
		return nil, fmt.Errorf("account: insert app: %w", err)
	}
	return app, nil
}

// AppByID loads an application. The caller decides whether the secret may be
// included in the response.
func (t *TokenService) AppByID(ctx context.Context, id string) (*AppRecord, error) {
	return t.scanApp(t.pool.QueryRow(ctx, `
SELECT id, user_id, name, description, permission, callback_url, secret, created_at
FROM apps WHERE id = $1
`, id))
}

// AppBySecret loads an application by the secret its client was issued.
func (t *TokenService) AppBySecret(ctx context.Context, secret string) (*AppRecord, error) {
	if secret == "" {
		return nil, ErrAppNotFound
	}
	return t.scanApp(t.pool.QueryRow(ctx, `
SELECT id, user_id, name, description, permission, callback_url, secret, created_at
FROM apps WHERE secret = $1
`, secret))
}

func (t *TokenService) scanApp(row pgx.Row) (*AppRecord, error) {
	var app AppRecord
	var perm []byte
	err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.Description, &perm,
		&app.CallbackURL, &app.Secret, &app.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan app: %w", err)
	}
	if len(perm) > 0 {
		if err := json.Unmarshal(perm, &app.Permission); err != nil {
			return nil, fmt.Errorf("account: decode app permission: %w", err)
		}
	}
	return &app, nil
}

// AppAuthorized reports whether the application already holds a token for the
// user. app/show and the approval page use it to show "already authorized".
func (t *TokenService) AppAuthorized(ctx context.Context, appID, userID string) (bool, error) {
	var n int
	if err := t.pool.QueryRow(ctx,
		`SELECT count(*) FROM access_tokens WHERE app_id = $1 AND user_id = $2`,
		appID, userID).Scan(&n); err != nil {
		return false, fmt.Errorf("account: check app authorization: %w", err)
	}
	return n > 0, nil
}

// CreateAuthSession opens a pending approval for an app.
func (t *TokenService) CreateAuthSession(ctx context.Context, appID string) (*AuthSession, error) {
	s := &AuthSession{ID: idgen.New(), AppID: appID, Token: auth.GenerateMiAuthToken()}
	if _, err := t.pool.Exec(ctx, `
INSERT INTO auth_sessions (id, app_id, token) VALUES ($1, $2, $3)
`, s.ID, s.AppID, s.Token); err != nil {
		return nil, fmt.Errorf("account: insert auth session: %w", err)
	}
	return s, nil
}

// SessionByToken loads a pending session by its public token.
func (t *TokenService) SessionByToken(ctx context.Context, token string) (*AuthSession, error) {
	s := &AuthSession{}
	err := t.pool.QueryRow(ctx, `
SELECT id, app_id, token, user_id FROM auth_sessions WHERE token = $1
`, token).Scan(&s.ID, &s.AppID, &s.Token, &s.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: load auth session: %w", err)
	}
	return s, nil
}

// AcceptSession records the approving user and issues the app's access token
// when the pair does not have one yet. The response carries the app so the
// caller can describe what was authorized.
func (t *TokenService) AcceptSession(ctx context.Context, token, userID string) (*AppRecord, error) {
	session, err := t.SessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	app, err := t.AppByID(ctx, session.AppID)
	if err != nil {
		return nil, err
	}
	authorized, err := t.AppAuthorized(ctx, app.ID, userID)
	if err != nil {
		return nil, err
	}
	if !authorized {
		accessToken := auth.GenerateMiAuthToken()
		if _, err := t.IssueForApp(ctx, userID, accessToken, app.ID, app.Name, app.Permission); err != nil {
			return nil, err
		}
	}
	if _, err := t.pool.Exec(ctx, `UPDATE auth_sessions SET user_id = $2 WHERE id = $1`, session.ID, userID); err != nil {
		return nil, fmt.Errorf("account: approve auth session: %w", err)
	}
	return app, nil
}

// SessionUserKey returns the access token issued for an approved session and
// the user who approved it, deleting the session so it cannot be collected
// twice. An unapproved session reports ErrSessionPending.
func (t *TokenService) SessionUserKey(ctx context.Context, appID, token string) (*AccessToken, string, error) {
	session, err := t.SessionByToken(ctx, token)
	if err != nil {
		return nil, "", err
	}
	if session.AppID != appID {
		return nil, "", ErrSessionNotFound
	}
	if session.UserID == nil {
		return nil, "", ErrSessionPending
	}

	at := &AccessToken{}
	err = t.pool.QueryRow(ctx, `
SELECT id, COALESCE(token, ''), user_id, session, name, scopes, created_at
FROM access_tokens WHERE app_id = $1 AND user_id = $2
`, appID, *session.UserID).Scan(&at.ID, &at.Token, &at.UserID, &at.Session, &at.Name, &at.Scopes, &at.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrInvalidToken
	}
	if err != nil {
		return nil, "", fmt.Errorf("account: load app token: %w", err)
	}

	if _, err := t.pool.Exec(ctx, `DELETE FROM auth_sessions WHERE id = $1`, session.ID); err != nil {
		return nil, "", fmt.Errorf("account: consume auth session: %w", err)
	}
	return at, *session.UserID, nil
}

// IssueForApp is Issue for a token bound to an application. The app's
// permissions are the token's scopes.
func (t *TokenService) IssueForApp(ctx context.Context, userID, token, appID, name string, scopes []string) (*AccessToken, error) {
	if scopes == nil {
		scopes = []string{}
	}
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, fmt.Errorf("account: marshal scopes: %w", err)
	}
	at := &AccessToken{ID: idgen.New(), Token: token, UserID: userID, CreatedAt: time.Now()}
	if name != "" {
		at.Name = &name
	}
	if _, err := t.pool.Exec(ctx, `
INSERT INTO access_tokens (id, token, token_hash, user_id, name, scopes, app_id)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
`, at.ID, at.Token, auth.HashToken(at.Token), at.UserID, at.Name, scopesJSON, appID); err != nil {
		return nil, fmt.Errorf("account: insert app token: %w", err)
	}
	return at, nil
}
