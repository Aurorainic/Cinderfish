// Package db provides the PostgreSQL connection pool and embedded
// schema migrations for Cinderfish.
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
// schema_migrations. Kept in code so a Cinderfish deployment is one binary.
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

	// v3: notes, reactions and notifications.
	3: `
-- notes.user_host marks the origin host of the note's author; it is
-- NULL for local notes and drives the local timeline filter.
ALTER TABLE notes
    ADD COLUMN user_host TEXT,
    ADD COLUMN reply_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN renote_user_host TEXT,
    ADD COLUMN name TEXT,
    ADD COLUMN url TEXT,
    ADD COLUMN renote_count INT NOT NULL DEFAULT 0,
    ADD COLUMN replies_count INT NOT NULL DEFAULT 0,
    ADD COLUMN clipped_count INT NOT NULL DEFAULT 0,
    ADD COLUMN is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN reaction_acceptance TEXT,
    ADD COLUMN file_ids TEXT[] NOT NULL DEFAULT '{}';

-- The inline poll column is superseded by the polls table.
ALTER TABLE notes DROP COLUMN IF EXISTS poll;

CREATE INDEX idx_notes_user_host ON notes (user_host);
CREATE INDEX idx_notes_reply ON notes (reply_id);
CREATE INDEX idx_notes_renote ON notes (renote_id);

-- Polls live in their own table so votes can be counted efficiently.
CREATE TABLE polls (
    id         TEXT PRIMARY KEY REFERENCES notes (id) ON DELETE CASCADE,
    choices    JSONB NOT NULL,
    multiple   BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE poll_votes (
    note_id   TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    user_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    choice    INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (note_id, user_id, choice)
);

-- Reactions are stored per emoji so counts can be aggregated.
ALTER TABLE note_reactions
    ADD COLUMN host TEXT;
CREATE INDEX idx_note_reactions_note ON note_reactions (note_id);

-- Notifications carry the note and the actor that triggered them.
ALTER TABLE notifications
    ADD COLUMN reaction TEXT;
CREATE INDEX idx_notifications_type ON notifications (recipient_id, type, created_at DESC);
`,

	// v4: federation support for M5.
	4: `
-- Remote users are the same rows as local users with host set; they
-- carry no password. uri is the ActivityPub actor id and is the join key
-- for every inbound activity.
ALTER TABLE users
    ADD COLUMN uri TEXT,
    ADD COLUMN inbox TEXT,
    ADD COLUMN shared_inbox TEXT,
    ADD COLUMN followers_uri TEXT,
    ADD COLUMN featured_uri TEXT,
    ADD COLUMN key_id TEXT,
    ADD COLUMN public_key_pem TEXT,
    ADD COLUMN is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_bot BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN last_fetched_at TIMESTAMPTZ,
    ADD COLUMN followers_count INT NOT NULL DEFAULT 0,
    ADD COLUMN following_count INT NOT NULL DEFAULT 0,
    ADD COLUMN notes_count INT NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX idx_users_uri ON users (uri) WHERE uri IS NOT NULL;
CREATE INDEX idx_users_remote ON users (host) WHERE host IS NOT NULL;

-- A remote user's avatar and banner are fetched from their actor document
-- rather than served out of our drive, so they need their own columns.
ALTER TABLE user_profiles
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN banner_url TEXT,
    ADD COLUMN fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN followers_visibility TEXT NOT NULL DEFAULT 'public',
    ADD COLUMN following_visibility TEXT NOT NULL DEFAULT 'public';

-- Follow provenance carries the Follow activity id so a later Undo can be
-- matched and so the Follow/Accept exchange stays idempotent.
ALTER TABLE followings
    ADD COLUMN uri TEXT,
    ADD COLUMN created_via TEXT;
ALTER TABLE follow_requests
    ADD COLUMN uri TEXT;
CREATE UNIQUE INDEX idx_follow_requests_uri ON follow_requests (uri) WHERE uri IS NOT NULL;

-- Notes carry the activity id that created them (the Create id for local
-- notes, the Announce id for remote renotes) so Undo and Delete can be
-- matched, plus ap_id: the canonical ActivityPub object id.
ALTER TABLE notes
    ADD COLUMN activity_uri TEXT,
    ADD COLUMN ap_id TEXT;
CREATE UNIQUE INDEX idx_notes_ap_id ON notes (ap_id) WHERE ap_id IS NOT NULL;
CREATE INDEX idx_notes_activity_uri ON notes (activity_uri) WHERE activity_uri IS NOT NULL;

-- The allowlist backing federation_mode = 'allowlist'. Rows are hostnames
-- matched by suffix; see federation.Policy.
CREATE TABLE federation_allowlist (
    host       TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Deliveries are per inbox, not per activity: one note fans out to every
-- follower's inbox and each gets its own retry schedule.
ALTER TABLE deliver_queue
    ADD COLUMN activity_id TEXT,
    ADD COLUMN key_id TEXT,
    ADD COLUMN user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN delivered_at TIMESTAMPTZ;
CREATE INDEX idx_deliver_queue_inbox ON deliver_queue (inbox_url) WHERE delivered_at IS NULL;

-- The unique index from v2 made usernames globally unique, which is
-- wrong once remote users live in the same table: alice@a.example and
-- alice@b.example are different people. Uniqueness belongs on the
-- (username, host) pair, which the original table already enforced.
DROP INDEX IF EXISTS idx_users_username_lower;
CREATE UNIQUE INDEX idx_users_username_lower_local ON users (username_lower) WHERE host IS NULL;
CREATE UNIQUE INDEX idx_users_username_lower_remote ON users (username_lower, host) WHERE host IS NOT NULL;
`,

	// v5: pinned notes.
	5: `
ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS pinned_note_ids TEXT[] NOT NULL DEFAULT '{}';
`,

	// v6: note favorites (Misskey notes/favorites/* and i/favorites).
	6: `
CREATE TABLE IF NOT EXISTS note_favorites (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note_id    TEXT NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, note_id)
);
CREATE INDEX IF NOT EXISTS idx_note_favorites_user ON note_favorites (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_note_favorites_note ON note_favorites (note_id);
`,

	// v7: drive folders (Misskey drive/folders/*).
	7: `
CREATE TABLE IF NOT EXISTS drive_folders (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    parent_id  TEXT REFERENCES drive_folders (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_drive_folders_user ON drive_folders (user_id, parent_id, id DESC);

-- A file may live in a folder; deleting the folder returns its files to the
-- parent rather than destroying them, which the delete handler does
-- explicitly, so the FK only has to avoid dangling ids.
ALTER TABLE drive_files
    ADD COLUMN IF NOT EXISTS folder_id TEXT REFERENCES drive_folders (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_drive_files_folder ON drive_files (folder_id) WHERE folder_id IS NOT NULL;
`,

	// v8: custom emoji registry (Misskey emojis and admin/emoji/*).
	8: `
CREATE TABLE IF NOT EXISTS emojis (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    host         TEXT,
    category     TEXT,
    aliases      JSONB NOT NULL DEFAULT '[]'::jsonb,
    url          TEXT NOT NULL,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    local_only   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_emojis_name_local ON emojis (name) WHERE host IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_emojis_name_host ON emojis (name, host) WHERE host IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_emojis_category ON emojis (category, name);
`,

	// v9: per-user client registry (Misskey i/registry/*). The scope is a
	// normalized text[] so equality is stable in the primary key.
	9: `
CREATE TABLE IF NOT EXISTS user_registry (
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    domain     TEXT NOT NULL DEFAULT '',
    scope      TEXT[] NOT NULL DEFAULT '{}',
    key        TEXT NOT NULL,
    value      JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, domain, scope, key)
);
CREATE INDEX IF NOT EXISTS idx_user_registry_user ON user_registry (user_id, scope);
`,

	// v10: instance announcements. `expires_at` is nullable so most notices are
	// permanent; the public list filters on is_active and the expiry.
	10: `
CREATE TABLE IF NOT EXISTS announcements (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    text       TEXT NOT NULL,
    image_url  TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_announcements_created ON announcements (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_announcements_active ON announcements (is_active, expires_at);
`,

	// v11: abuse reports. A report targets a user (Misskey's contract); the
	// resolved flag is the administrator's handling state.
	11: `
CREATE TABLE IF NOT EXISTS abuse_reports (
    id             TEXT PRIMARY KEY,
    reporter_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    comment        TEXT NOT NULL,
    resolved       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_unresolved ON abuse_reports (resolved, created_at DESC);
`,

	// v12: per-user outbound webhooks. `on` holds Misskey's event names; only
	// note/reply/renote are produced by this port.
	12: `
CREATE TABLE IF NOT EXISTS webhooks (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    url            TEXT NOT NULL,
    secret         TEXT NOT NULL,
    events         TEXT[] NOT NULL DEFAULT '{}',
    active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    latest_sent_at TIMESTAMPTZ,
    latest_status  INTEGER
);
CREATE INDEX IF NOT EXISTS idx_webhooks_user ON webhooks (user_id, created_at DESC);
`,

	// v13: MiAuth hand-off. `fetched` marks the one-time collection of a token
	// by the requesting application through miauth/{session}/check.
	13: `
ALTER TABLE access_tokens ADD COLUMN IF NOT EXISTS fetched BOOLEAN NOT NULL DEFAULT FALSE;
`,

	// v14: TOTP two-factor auth. `two_factor_temp_secret` holds the pending
	// secret between i/2fa/register and i/2fa/done; `two_factor_enabled` gates
	// the second step of signin.
	14: `
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS two_factor_secret TEXT;
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS two_factor_temp_secret TEXT;
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS use_password_less_login BOOLEAN NOT NULL DEFAULT FALSE;
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
