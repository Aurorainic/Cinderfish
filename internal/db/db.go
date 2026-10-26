// Package db provides the PostgreSQL connection pool and embedded
// schema migrations for MilnFish.
package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrations maps version number to SQL script. Scripts run in ascending
// order inside a single transaction each; applied versions are recorded in
// schema_migrations. Kept in code so a MilnFish deployment is one binary.
var migrations = map[int]string{
	1: `
CREATE TABLE instance_meta (
    key   TEXT PRIMARY KEY,
    value JSONB NOT NULL
);

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL,
    host          TEXT,
    password_hash TEXT,
    is_admin      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (username, host)
);
CREATE INDEX idx_users_host ON users (host);

CREATE TABLE user_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT,
    description TEXT,
    settings    JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE followings (
    id         TEXT PRIMARY KEY,
    follower_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followee_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (follower_id, followee_id)
);

CREATE TABLE follow_requests (
    id         TEXT PRIMARY KEY,
    follower_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    followee_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (follower_id, followee_id)
);

CREATE TABLE blocks (
    id        TEXT PRIMARY KEY,
    blocker_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    blockee_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_id, blockee_id)
);

CREATE TABLE mutes (
    id        TEXT PRIMARY KEY,
    muter_id  TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mutee_id  TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (muter_id, mutee_id)
);

CREATE TABLE notes (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    visibility     TEXT NOT NULL CHECK (visibility IN ('public','home','followers','specified')),
    local_only     BOOLEAN NOT NULL DEFAULT FALSE,
    cw             TEXT,
    text           TEXT,
    poll           JSONB,
    reply_id       TEXT REFERENCES notes (id) ON DELETE SET NULL,
    renote_id      TEXT REFERENCES notes (id) ON DELETE SET NULL,
    renote_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    uri            TEXT UNIQUE,
    tags           JSONB NOT NULL DEFAULT '[]'::jsonb,
    mentions       JSONB NOT NULL DEFAULT '[]'::jsonb,
    emojis         JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX idx_notes_user_created ON notes (user_id, created_at DESC);
CREATE INDEX idx_notes_created ON notes (created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE note_visibilities (
    note_id           TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    visible_user_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, visible_user_id)
);

CREATE TABLE note_reactions (
    id      TEXT PRIMARY KEY,
    note_id TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    emoji   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (note_id, user_id, emoji)
);

CREATE TABLE drive_files (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size        BIGINT NOT NULL,
    md5         TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('local','s3')),
    thumbnail_key TEXT,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id          TEXT PRIMARY KEY,
    recipient_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    notifier_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    note_id     TEXT REFERENCES notes (id) ON DELETE CASCADE,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notifications_recipient ON notifications (recipient_id, created_at DESC);

CREATE TABLE access_tokens (
    id          TEXT PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    scopes      JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ap_keys (
    user_id      TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    public_key_pem  TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ap_activities (
    id         TEXT PRIMARY KEY,
    direction  TEXT NOT NULL CHECK (direction IN ('inbound','outbound')),
    activity   JSONB NOT NULL,
    actor_uri  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE deliver_queue (
    id              BIGSERIAL PRIMARY KEY,
    payload         JSONB NOT NULL,
    inbox_url       TEXT NOT NULL,
    attempts        INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error      TEXT
);
CREATE INDEX idx_deliver_pending ON deliver_queue (next_attempt_at) WHERE attempts < 12;

CREATE TABLE events (
    id         BIGSERIAL PRIMARY KEY,
    channel    TEXT NOT NULL,
    user_id    TEXT,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_events_recent ON events (created_at DESC);
`,

	// v2: account and authentication support for M2.
	2: `
ALTER TABLE users
    ADD COLUMN username_lower TEXT NOT NULL DEFAULT '',
    ADD COLUMN token TEXT,
    ADD COLUMN is_suspended BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE users SET username_lower = lower(username) WHERE username_lower = '';
CREATE UNIQUE INDEX idx_users_username_lower ON users (username_lower);
CREATE UNIQUE INDEX idx_users_token ON users (token) WHERE token IS NOT NULL;

ALTER TABLE user_profiles
    ADD COLUMN email TEXT;

-- Registration tickets back the invite-only registration mode.
CREATE TABLE registration_tickets (
    id         TEXT PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    used_by_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    used_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE access_tokens
    ADD COLUMN token TEXT,
    ADD COLUMN session TEXT,
    ADD COLUMN name TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN icon_url TEXT,
    ADD COLUMN last_used_at TIMESTAMPTZ;
CREATE INDEX idx_access_tokens_user ON access_tokens (user_id);

-- Login history, used by the signin flow and the admin panel.
CREATE TABLE signins (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ip         TEXT,
    success    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_signins_user ON signins (user_id, created_at DESC);

-- OAuth 2.0 authorization codes. Codes are single use; the row is
-- deleted or marked used once redeemed.
CREATE TABLE oauth_grant_codes (
    code          TEXT PRIMARY KEY,
    client_id     TEXT NOT NULL,
    user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    redirect_uri  TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    scopes        JSONB NOT NULL DEFAULT '[]'::jsonb,
    used          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
`,
}

// Migrate applies pending migrations in order and returns the version
// reached (0 for a fresh database).
func Migrate(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	if _, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return 0, fmt.Errorf("db: create schema_migrations: %w", err)
	}

	var current int
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return 0, fmt.Errorf("db: read current version: %w", err)
	}

	versions := make([]int, 0, len(migrations))
	for v := range migrations {
		versions = append(versions, v)
	}
	sort.Ints(versions)

	for _, v := range versions {
		if v <= current {
			continue
		}
		if err := applyOne(ctx, pool, v, migrations[v]); err != nil {
			return current, fmt.Errorf("db: migration %d: %w", v, err)
		}
		current = v
	}
	return current, nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, version int, script string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, strings.TrimSpace(script)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
