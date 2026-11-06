package federation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Store persists remote actors and notes. It is the only part of the
// federation package that touches the database directly.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a store over pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ErrRemoteNotFound is returned when a remote object cannot be resolved
// or stored.
var ErrRemoteNotFound = errors.New("federation: remote object not found")

// RemoteUser is a remote actor as stored in the users table.
type RemoteUser struct {
	ID           string
	Username     string
	Host         string
	URI          string
	Inbox        string
	SharedInbox  string
	FollowersURI string
	FeaturedURI  string
	KeyID        string
	PublicKeyPEM string
	Name         string
	Description  string
	AvatarURL    string
	BannerURL    string
	IsLocked     bool
	IsBot        bool
	IsDeleted    bool
	// MovedToURI is the actor this one migrated to, if any.
	MovedToURI string
	// AlsoKnownAs lists the actor URIs this actor declares as aliases of
	// itself; an account move is validated against it.
	AlsoKnownAs []string
	Fields      []ProfileField
}

// UpsertRemoteUser inserts or updates a remote actor keyed by its AP id.
// Re-fetching an actor refreshes the mutable fields and leaves follow
// relationships alone.
func (s *Store) UpsertRemoteUser(ctx context.Context, u RemoteUser) (*RemoteUser, error) {
	if u.URI == "" || u.Host == "" || u.Username == "" {
		return nil, fmt.Errorf("federation: remote actor is missing uri, host or username")
	}
	usernameLower := strings.ToLower(u.Username)

	var id string
	err := s.pool.QueryRow(ctx, "SELECT id FROM users WHERE uri = $1", u.URI).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		id = idgen.New()
		_, err = s.pool.Exec(ctx, `
INSERT INTO users (id, username, username_lower, host, uri, inbox, shared_inbox,
                   followers_uri, featured_uri, key_id, public_key_pem,
                   is_locked, is_bot, is_deleted, last_fetched_at, created_at,
                   moved_to_uri, also_known_as)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now(),now(),$15,$16::jsonb)
		`, id, u.Username, usernameLower, u.Host, u.URI, u.Inbox, u.SharedInbox,
			u.FollowersURI, u.FeaturedURI, u.KeyID, u.PublicKeyPEM,
			u.IsLocked, u.IsBot, u.IsDeleted,
			nullIfEmpty(u.MovedToURI), marshalStrings(u.AlsoKnownAs))
		if err != nil {
			return nil, fmt.Errorf("federation: insert remote user: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("federation: look up remote user: %w", err)
	default:
		_, err = s.pool.Exec(ctx, `
UPDATE users SET inbox = $2, shared_inbox = $3, followers_uri = $4, featured_uri = $5,
                 key_id = $6, public_key_pem = $7, is_locked = $8, is_bot = $9,
                 is_deleted = $10, last_fetched_at = now(),
                 moved_to_uri = COALESCE($11, moved_to_uri),
                 also_known_as = CASE WHEN $12::jsonb = '[]'::jsonb THEN also_known_as ELSE $12::jsonb END
WHERE id = $1
		`, id, u.Inbox, u.SharedInbox, u.FollowersURI, u.FeaturedURI,
			u.KeyID, u.PublicKeyPEM, u.IsLocked, u.IsBot, u.IsDeleted,
			nullIfEmpty(u.MovedToURI), marshalStrings(u.AlsoKnownAs))
		if err != nil {
			return nil, fmt.Errorf("federation: update remote user: %w", err)
		}
	}

	fields := marshalFields(u.Fields)
	if _, err := s.pool.Exec(ctx, `
INSERT INTO user_profiles (user_id, name, description, avatar_url, banner_url, fields)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (user_id) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description,
    avatar_url = EXCLUDED.avatar_url, banner_url = EXCLUDED.banner_url,
    fields = EXCLUDED.fields
	`, id, nullIfEmpty(u.Name), nullIfEmpty(u.Description), nullIfEmpty(u.AvatarURL), nullIfEmpty(u.BannerURL), fields); err != nil {
		return nil, fmt.Errorf("federation: upsert remote profile: %w", err)
	}

	u.ID = id
	return &u, nil
}

// UserByURI loads a remote user by its AP id.
func (s *Store) UserByURI(ctx context.Context, uri string) (*RemoteUser, error) {
	u := &RemoteUser{}
	var name, desc, avatar, banner *string
	err := s.pool.QueryRow(ctx, `
SELECT u.id, u.username, COALESCE(u.host, ''), u.uri, COALESCE(u.inbox, ''),
       COALESCE(u.shared_inbox, ''), COALESCE(u.followers_uri, ''),
       COALESCE(u.featured_uri, ''), COALESCE(u.key_id, ''),
       COALESCE(u.public_key_pem, ''), p.name, p.description, p.avatar_url, p.banner_url,
       u.is_locked, u.is_bot, u.is_deleted,
       COALESCE(u.moved_to_uri, ''), COALESCE(u.also_known_as, '[]'::jsonb)
FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.uri = $1
	`, uri).Scan(&u.ID, &u.Username, &u.Host, &u.URI, &u.Inbox, &u.SharedInbox,
		&u.FollowersURI, &u.FeaturedURI, &u.KeyID, &u.PublicKeyPEM,
		&name, &desc, &avatar, &banner, &u.IsLocked, &u.IsBot, &u.IsDeleted,
		&u.MovedToURI, &u.AlsoKnownAs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: load remote user: %w", err)
	}
	u.Name = deref(name)
	u.Description = deref(desc)
	u.AvatarURL = deref(avatar)
	u.BannerURL = deref(banner)
	return u, nil
}

// LocalUserByID resolves a local user id to the data the inbox needs.
func (s *Store) LocalUserByID(ctx context.Context, id string) (*localActor, error) {
	var locked bool
	err := s.pool.QueryRow(ctx, "SELECT is_locked FROM users WHERE id = $1 AND host IS NULL", id).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: load local user: %w", err)
	}
	return &localActor{id: id, locked: locked}, nil
}

// LocalUserLocked reports whether a local account requires manual approval of
// new followers. An unknown id reports false so a caller that just looked the
// user up does not have to distinguish the two failures.
func (s *Store) LocalUserLocked(ctx context.Context, id string) (bool, error) {
	var locked bool
	err := s.pool.QueryRow(ctx, "SELECT is_locked FROM users WHERE id = $1 AND host IS NULL", id).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("federation: local user lock state: %w", err)
	}
	return locked, nil
}

// LocalUserOfAPID maps a note id or actor id belonging to this instance
// to the local user that owns it.
func (s *Store) LocalUserOfAPID(ctx context.Context, apID, baseURL string) (string, error) {
	if rest, ok := strings.CutPrefix(apID, baseURL+"/notes/"); ok {
		if id := lastPathSegment(rest); id != "" {
			var userID string
			if err := s.pool.QueryRow(ctx, "SELECT user_id FROM notes WHERE id = $1 AND user_host IS NULL", id).Scan(&userID); err == nil {
				return userID, nil
			}
		}
	}
	if rest, ok := strings.CutPrefix(apID, baseURL+"/users/"); ok {
		if id := lastPathSegment(rest); id != "" {
			var n int
			if err := s.pool.QueryRow(ctx, "SELECT 1 FROM users WHERE id = $1 AND host IS NULL", id).Scan(&n); err == nil {
				return id, nil
			}
		}
	}
	return "", ErrRemoteNotFound
}

// NoteOwnerByAPID returns the local owner of a note addressed by its AP
// id, plus the note author's host (nil for a local note).
func (s *Store) NoteOwnerByAPID(ctx context.Context, apID string) (string, *string, error) {
	var userID string
	var host *string
	err := s.pool.QueryRow(ctx, "SELECT user_id, user_host FROM notes WHERE ap_id = $1 OR uri = $1", apID).Scan(&userID, &host)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrRemoteNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("federation: note owner: %w", err)
	}
	return userID, host, nil
}

// MarkUserDeleted flags a remote actor deleted after an inbound Delete.
func (s *Store) MarkUserDeleted(ctx context.Context, id string) error {
	if _, err := s.pool.Exec(ctx, "UPDATE users SET is_deleted = TRUE WHERE id = $1", id); err != nil {
		return fmt.Errorf("federation: mark user deleted: %w", err)
	}
	return nil
}

// RecordBlock stores an inbound Block and removes any follow between the
// two users in either direction.
func (s *Store) RecordBlock(ctx context.Context, blockerID, blockeeID string) error {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO blocks (id, blocker_id, blockee_id) VALUES ($1,$2,$3)
ON CONFLICT (blocker_id, blockee_id) DO NOTHING
	`, idgen.New(), blockerID, blockeeID); err != nil {
		return fmt.Errorf("federation: record block: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
DELETE FROM followings WHERE (follower_id = $1 AND followee_id = $2)
                          OR (follower_id = $2 AND followee_id = $1)
	`, blockerID, blockeeID); err != nil {
		return fmt.Errorf("federation: remove follows after block: %w", err)
	}
	return nil
}

// RecordReport stores an abuse report filed by a remote actor. The reporter
// is the remote user row, so the admin list and the resolve endpoint work the
// same as for a local report.
func (s *Store) RecordReport(ctx context.Context, reporterID, targetUserID, comment string) error {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO abuse_reports (id, reporter_id, target_user_id, comment)
VALUES ($1,$2,$3,$4)
	`, idgen.New(), reporterID, targetUserID, comment); err != nil {
		return fmt.Errorf("federation: record report: %w", err)
	}
	return nil
}

// RecordActivity appends an inbound or outbound activity to the audit
// table.
func (s *Store) RecordActivity(ctx context.Context, direction, actorURI string, activity Object) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("federation: encode activity: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO ap_activities (id, direction, activity, actor_uri) VALUES ($1,$2,$3,$4)
	`, idgen.New(), direction, body, nullIfEmpty(actorURI)); err != nil {
		return fmt.Errorf("federation: record activity: %w", err)
	}
	return nil
}

// LocalKey returns a local user's stored keypair.
func (s *Store) LocalKey(ctx context.Context, userID string) (publicPEM, privatePEM string, err error) {
	err = s.pool.QueryRow(ctx, `
SELECT public_key_pem, private_key_pem FROM ap_keys WHERE user_id = $1
	`, userID).Scan(&publicPEM, &privatePEM)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("federation: no key pair for user %s", userID)
	}
	if err != nil {
		return "", "", fmt.Errorf("federation: load key pair: %w", err)
	}
	return publicPEM, privatePEM, nil
}

// EnsureLocalKey creates and stores a keypair for a local user that has
// none and returns whatever ends up stored.
func (s *Store) EnsureLocalKey(ctx context.Context, userID string) (*Keypair, error) {
	if pub, priv, err := s.LocalKey(ctx, userID); err == nil {
		return &Keypair{PublicPEM: pub, PrivatePEM: priv}, nil
	}
	kp, err := GenerateKeypair()
	if err != nil {
		return nil, err
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO ap_keys (user_id, public_key_pem, private_key_pem) VALUES ($1,$2,$3)
ON CONFLICT (user_id) DO NOTHING
	`, userID, kp.PublicPEM, kp.PrivatePEM); err != nil {
		return nil, fmt.Errorf("federation: store key pair: %w", err)
	}
	if pub, priv, err := s.LocalKey(ctx, userID); err == nil {
		return &Keypair{PublicPEM: pub, PrivatePEM: priv}, nil
	}
	return kp, nil
}

// UserByHandle loads a remote user by username and host, the form a mention
// carries before it is resolved to an actor URI.
func (s *Store) UserByHandle(ctx context.Context, username, host string) (*RemoteUser, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
SELECT id FROM users WHERE username_lower = lower($1) AND host = $2
`, username, host).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: find user by handle: %w", err)
	}
	return s.UserByID(ctx, id)
}

// NoteExists reports whether a note with this AP id is already stored.
func (s *Store) NoteExists(ctx context.Context, apID string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, "SELECT 1 FROM notes WHERE ap_id = $1 OR uri = $1 LIMIT 1", apID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("federation: check note: %w", err)
	}
	return true, nil
}

// NoteIDByAPID returns the local id of a stored note addressed by its AP id.
func (s *Store) NoteIDByAPID(ctx context.Context, apID string) (string, error) {
	var id string
	if err := s.pool.QueryRow(ctx, `SELECT id FROM notes WHERE ap_id = $1 OR uri = $1 LIMIT 1`, apID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRemoteNotFound
		}
		return "", fmt.Errorf("federation: load note id: %w", err)
	}
	return id, nil
}

// RemoteNote is a note received over ActivityPub.
type RemoteNote struct {
	APID        string
	ActivityURI string
	UserID      string
	UserHost    string
	Visibility  string
	Text        string
	CW          string
	ReplyAPID   string
	QuoteAPID   string
	// ReplyLocalID/QuoteLocalID carry an already resolved local note id when
	// the target is a local note addressed by its ActivityPub URL; the
	// stored row only knows ap_id/uri for remote notes.
	ReplyLocalID string
	QuoteLocalID string
	Tags         []string
	Emojis       []EmojiRef
	Sensitive    bool
	CreatedAt    time.Time
	// Poll is set for an inbound Question: the choices, whether more than one
	// may be picked, and an optional closing time. Empty means a plain note.
	PollChoices   []string
	PollMultiple  bool
	PollExpiresAt *time.Time
}

// InsertRemoteNote stores a received note. Reply and renote targets are
// resolved to local note ids when they are already known.
func (s *Store) InsertRemoteNote(ctx context.Context, n RemoteNote) (string, error) {
	id := idgen.New()

	var replyID, renoteID, renoteUserID *string
	if n.ReplyLocalID != "" {
		replyID = &n.ReplyLocalID
	} else if n.ReplyAPID != "" {
		var rid string
		if err := s.pool.QueryRow(ctx, "SELECT id FROM notes WHERE ap_id = $1 OR uri = $1", n.ReplyAPID).Scan(&rid); err == nil {
			replyID = &rid
		}
	}
	if n.QuoteLocalID != "" {
		var ruid string
		if err := s.pool.QueryRow(ctx, "SELECT user_id FROM notes WHERE id = $1", n.QuoteLocalID).Scan(&ruid); err == nil {
			rid := n.QuoteLocalID
			renoteID, renoteUserID = &rid, &ruid
		}
	} else if n.QuoteAPID != "" {
		var rid, ruid string
		if err := s.pool.QueryRow(ctx, "SELECT id, user_id FROM notes WHERE ap_id = $1 OR uri = $1", n.QuoteAPID).Scan(&rid, &ruid); err == nil {
			renoteID, renoteUserID = &rid, &ruid
		}
	}

	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	emojis := n.Emojis
	if emojis == nil {
		emojis = []EmojiRef{}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("federation: begin remote note: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
INSERT INTO notes (id, user_id, user_host, visibility, local_only, cw, text,
                   reply_id, renote_id, renote_user_id, ap_id, activity_uri,
                   tags, mentions, emojis, is_sensitive, created_at)
VALUES ($1,$2,$3,$4,FALSE,$5,$6,$7,$8,$9,$10,$11,$12,'[]'::jsonb,$13,$14,$15)
	`, id, n.UserID, n.UserHost, n.Visibility, nullIfEmpty(n.CW), nullIfEmpty(n.Text),
		replyID, renoteID, renoteUserID, n.APID, n.ActivityURI,
		marshalStrings(tags), marshalAny(emojis), n.Sensitive, n.CreatedAt); err != nil {
		return "", fmt.Errorf("federation: insert remote note: %w", err)
	}

	// A Question stores its poll in the same table local polls use, so the
	// existing note packing and vote endpoints render and accept it without
	// a second code path.
	if len(n.PollChoices) >= 2 {
		if _, err := tx.Exec(ctx, `
INSERT INTO polls (id, choices, multiple, expires_at) VALUES ($1,$2,$3,$4)
`, id, marshalStrings(n.PollChoices), n.PollMultiple, n.PollExpiresAt); err != nil {
			return "", fmt.Errorf("federation: insert remote poll: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("federation: commit remote note: %w", err)
	}
	return id, nil
}

// UpdateRemoteNote applies an inbound Update to a stored remote note and
// reports its local id. Only content moves: visibility and audience stay as
// stored, because a peer need not resend them and guessing wrong would either
// leak a followers-only note or hide it from the people already addressed.
func (s *Store) UpdateRemoteNote(ctx context.Context, apID string, text, cw *string, sensitive bool, updatedAt time.Time) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
UPDATE notes SET text = $2, cw = $3, is_sensitive = $4, updated_at = $5
WHERE ap_id = $1 AND deleted_at IS NULL
RETURNING id
`, apID, text, cw, sensitive, updatedAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRemoteNotFound
	}
	if err != nil {
		return "", fmt.Errorf("federation: update remote note: %w", err)
	}
	return id, nil
}

// FollowRequest records an inbound Follow from a remote actor to a local
// user who manually approves followers.
func (s *Store) FollowRequest(ctx context.Context, uri, followerID, followeeID string) error {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO follow_requests (id, follower_id, followee_id, uri)
VALUES ($1,$2,$3,$4) ON CONFLICT (follower_id, followee_id) DO NOTHING
	`, idgen.New(), followerID, followeeID, nullIfEmpty(uri)); err != nil {
		return fmt.Errorf("federation: record follow request: %w", err)
	}
	return nil
}

// FollowRequest is one pending follow request row.
type FollowRequest struct {
	ID         string
	FollowerID string
	FolloweeID string
	// URI is the id of the Follow activity a remote actor sent. It is
	// empty for a request a local user (or a local row) carries.
	URI       string
	CreatedAt time.Time
}

// FollowRequestQuery pages one side of the follow_requests table.
// FollowerID lists the requests a user sent, FolloweeID the requests a
// user received; a caller sets exactly one of them.
type FollowRequestQuery struct {
	FollowerID string
	FolloweeID string
	Limit      int
	SinceID    string
	UntilID    string
	SinceDate  *time.Time
	UntilDate  *time.Time
}

// FollowRequests returns pending requests. The id cursors bound the row
// id and the date cursors bound created_at; ordering follows Misskey's
// makePaginationQuery (packages/backend/src/core/QueryService.ts L54-84),
// where a lower bound alone pages upward and everything else is newest
// first.
func (s *Store) FollowRequests(ctx context.Context, q FollowRequestQuery) ([]*FollowRequest, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var conds []string
	var args []any
	add := func(col, op string, arg any) {
		args = append(args, arg)
		conds = append(conds, fmt.Sprintf("%s %s $%d", col, op, len(args)))
	}
	if q.FollowerID != "" {
		add("follower_id", "=", q.FollowerID)
	}
	if q.FolloweeID != "" {
		add("followee_id", "=", q.FolloweeID)
	}
	if q.SinceID != "" {
		add("id", ">", q.SinceID)
	}
	if q.UntilID != "" {
		add("id", "<", q.UntilID)
	}
	if q.SinceDate != nil {
		add("created_at", ">", *q.SinceDate)
	}
	if q.UntilDate != nil {
		add("created_at", "<", *q.UntilDate)
	}

	query := "SELECT " + followRequestColumns + " FROM follow_requests"
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	order := "DESC"
	if (q.SinceID != "" && q.UntilID == "") ||
		(q.SinceID == "" && q.UntilID == "" && q.SinceDate != nil && q.UntilDate == nil) {
		order = "ASC"
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", order, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("federation: list follow requests: %w", err)
	}
	defer rows.Close()

	out := []*FollowRequest{}
	for rows.Next() {
		req, err := scanFollowRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("federation: scan follow request: %w", err)
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

// AcceptFollowRequest turns a pending request into a follow. The request
// row is deleted, the following is recorded with the request's activity
// id, and both users' counters are refreshed, all in one transaction.
//
// The returned request carries the uri the caller answers a remote
// follower with; ErrRemoteNotFound means there was no pending request.
func (s *Store) AcceptFollowRequest(ctx context.Context, followerID, followeeID string) (*FollowRequest, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("federation: begin accept: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	req, err := scanFollowRequest(tx.QueryRow(ctx, `
DELETE FROM follow_requests WHERE follower_id = $1 AND followee_id = $2
RETURNING `+followRequestColumns, followerID, followeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: accept follow request: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO followings (id, follower_id, followee_id, uri, created_via)
VALUES ($1,$2,$3,$4,'request') ON CONFLICT (follower_id, followee_id) DO NOTHING
	`, idgen.New(), followerID, followeeID, nullIfEmpty(req.URI)); err != nil {
		return nil, fmt.Errorf("federation: record accepted follow: %w", err)
	}
	if err := refreshFollowCounts(ctx, tx, followerID, followeeID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("federation: commit accept: %w", err)
	}
	return req, nil
}

// RemoveFollowRequest deletes one pending request and returns it, without
// recording a follow: following/requests/reject passes the follower the
// caller received the request from, following/requests/cancel passes the
// caller as the follower. The delete is a single statement, so it is
// atomic on its own and needs no explicit transaction.
//
// ErrRemoteNotFound means there was no such request.
func (s *Store) RemoveFollowRequest(ctx context.Context, followerID, followeeID string) (*FollowRequest, error) {
	req, err := scanFollowRequest(s.pool.QueryRow(ctx, `
DELETE FROM follow_requests WHERE follower_id = $1 AND followee_id = $2
RETURNING `+followRequestColumns, followerID, followeeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: remove follow request: %w", err)
	}
	return req, nil
}

// Follow records an accepted Follow.
func (s *Store) Follow(ctx context.Context, uri, followerID, followeeID string) error {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO followings (id, follower_id, followee_id, uri, created_via)
VALUES ($1,$2,$3,$4,'activitypub') ON CONFLICT (follower_id, followee_id) DO NOTHING
	`, idgen.New(), followerID, followeeID, nullIfEmpty(uri)); err != nil {
		return fmt.Errorf("federation: record follow: %w", err)
	}
	_ = refreshFollowCounts(ctx, s.pool, followerID, followeeID)
	return nil
}

// Unfollow removes a follow and any pending request.
func (s *Store) Unfollow(ctx context.Context, followerID, followeeID string) error {
	if _, err := s.pool.Exec(ctx, `
DELETE FROM followings WHERE follower_id = $1 AND followee_id = $2
	`, followerID, followeeID); err != nil {
		return fmt.Errorf("federation: unfollow: %w", err)
	}
	_, _ = s.pool.Exec(ctx, "DELETE FROM follow_requests WHERE follower_id = $1 AND followee_id = $2", followerID, followeeID)
	_ = refreshFollowCounts(ctx, s.pool, followerID, followeeID)
	return nil
}

// MarkMoved records that a remote actor migrated to targetURI, appending the
// target to its alias list. The move itself is validated by the caller; this
// only writes the state peers read back from the actor document.
func (s *Store) MarkMoved(ctx context.Context, userID, targetURI string) error {
	if targetURI == "" {
		return fmt.Errorf("federation: empty move target")
	}
	var known json.RawMessage
	if err := s.pool.QueryRow(ctx, `SELECT also_known_as FROM users WHERE id = $1`, userID).Scan(&known); err != nil {
		return fmt.Errorf("federation: load known-as: %w", err)
	}
	var aliases []string
	if len(known) > 0 {
		if err := json.Unmarshal(known, &aliases); err != nil {
			return fmt.Errorf("federation: decode known-as: %w", err)
		}
	}
	found := false
	for _, a := range aliases {
		if a == targetURI {
			found = true
			break
		}
	}
	if !found {
		aliases = append(aliases, targetURI)
	}
	if _, err := s.pool.Exec(ctx, `
UPDATE users SET moved_to_uri = $2, also_known_as = $3::jsonb WHERE id = $1
	`, userID, targetURI, marshalStrings(aliases)); err != nil {
		return fmt.Errorf("federation: mark moved: %w", err)
	}
	return nil
}

// LocalFollowerIDs returns the local accounts that follow userID. The
// account migration moves them to the target account, and a remote follower
// is the remote instance's business, so only local rows are returned.
func (s *Store) LocalFollowerIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT f.follower_id FROM followings f JOIN users u ON u.id = f.follower_id
WHERE f.followee_id = $1 AND u.host IS NULL
ORDER BY f.created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("federation: local followers: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// followRequestColumns is the projection the follow request helpers scan.
const followRequestColumns = `id, follower_id, followee_id, COALESCE(uri, ''), created_at`

// scanFollowRequest reads one follow_requests row.
func scanFollowRequest(row pgx.Row) (*FollowRequest, error) {
	req := &FollowRequest{}
	if err := row.Scan(&req.ID, &req.FollowerID, &req.FolloweeID, &req.URI, &req.CreatedAt); err != nil {
		return nil, err
	}
	return req, nil
}

// execer is the statement runner shared by pgxpool.Pool and pgx.Tx, so a
// helper can run both inside and outside a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// refreshFollowCounts recomputes both users' counters from the followings
// table. The counters are derived data: a caller outside a transaction
// may ignore the error, but a transactional caller returns it so the
// whole change rolls back with them.
func refreshFollowCounts(ctx context.Context, db execer, followerID, followeeID string) error {
	if _, err := db.Exec(ctx, `
UPDATE users SET followers_count = (SELECT count(*) FROM followings WHERE followee_id = $1) WHERE id = $1
	`, followeeID); err != nil {
		return fmt.Errorf("federation: refresh follower count: %w", err)
	}
	if _, err := db.Exec(ctx, `
UPDATE users SET following_count = (SELECT count(*) FROM followings WHERE follower_id = $1) WHERE id = $1
	`, followerID); err != nil {
		return fmt.Errorf("federation: refresh following count: %w", err)
	}
	return nil
}

// HasFollowing reports whether follower follows followee.
func (s *Store) HasFollowing(ctx context.Context, followerID, followeeID string) bool {
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM followings WHERE follower_id = $1 AND followee_id = $2
	`, followerID, followeeID).Scan(&n)
	return err == nil && n > 0
}

// HasPendingFollowRequest reports whether follower has a follow request
// awaiting approval by followee.
func (s *Store) HasPendingFollowRequest(ctx context.Context, followerID, followeeID string) bool {
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM follow_requests WHERE follower_id = $1 AND followee_id = $2
	`, followerID, followeeID).Scan(&n)
	return err == nil && n > 0
}

// DeleteNoteByAPID soft deletes a received note.
func (s *Store) DeleteNoteByAPID(ctx context.Context, apID, actorUserID string) error {
	ct, err := s.pool.Exec(ctx, `
UPDATE notes SET deleted_at = now() WHERE (ap_id = $1 OR uri = $1) AND user_id = $2 AND deleted_at IS NULL
	`, apID, actorUserID)
	if err != nil {
		return fmt.Errorf("federation: delete remote note: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrRemoteNotFound
	}
	return nil
}

// NoteIDByAPIDOrURL resolves a note's local id from an ActivityPub id. A local
// note is addressed as <baseURL>/notes/<id> and has no ap_id column value, so
// that form is resolved directly before falling back to the stored ids.
func (s *Store) NoteIDByAPIDOrURL(ctx context.Context, apID, baseURL string) (string, error) {
	if rest, ok := strings.CutPrefix(apID, baseURL+"/notes/"); ok {
		id := lastPathSegment(rest)
		if id != "" {
			var found string
			if err := s.pool.QueryRow(ctx, "SELECT id FROM notes WHERE id = $1", id).Scan(&found); err == nil {
				return found, nil
			}
		}
	}
	return s.NoteIDByAPID(ctx, apID)
}

// ReactNoteID stores a reaction addressed by local note id, which is the form
// both local users and inbound activities end up using.
func (s *Store) ReactNoteID(ctx context.Context, noteID, userID, reaction string) error {
	if _, err := s.pool.Exec(ctx, `
INSERT INTO note_reactions (id, note_id, user_id, emoji, host)
VALUES ($1,$2,$3,$4,(SELECT host FROM users WHERE id = $3))
ON CONFLICT (note_id, user_id, emoji) DO NOTHING
	`, idgen.New(), noteID, userID, reaction); err != nil {
		return fmt.Errorf("federation: store reaction: %w", err)
	}
	return nil
}

// UnreactNoteID removes a reaction addressed by local note id.
func (s *Store) UnreactNoteID(ctx context.Context, noteID, userID string) error {
	if _, err := s.pool.Exec(ctx, "DELETE FROM note_reactions WHERE user_id = $2 AND note_id = $1", noteID, userID); err != nil {
		return fmt.Errorf("federation: remove reaction: %w", err)
	}
	return nil
}

// Notification creates a notification row for a remote actor's action.
func (s *Store) Notification(ctx context.Context, recipientID, typ, notifierID, noteID string) (string, error) {
	id := idgen.New()
	_, err := s.pool.Exec(ctx, `
INSERT INTO notifications (id, recipient_id, type, notifier_id, note_id) VALUES ($1,$2,$3,$4,$5)
	`, id, recipientID, typ, notifierID, nullIfEmpty(noteID))
	if err != nil {
		return "", fmt.Errorf("federation: create notification: %w", err)
	}
	return id, nil
}

// NoteOwner returns the local user id that owns a note and its host.
func (s *Store) NoteOwner(ctx context.Context, noteID string) (userID string, userHost *string, err error) {
	err = s.pool.QueryRow(ctx, "SELECT user_id, user_host FROM notes WHERE id = $1", noteID).Scan(&userID, &userHost)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrRemoteNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("federation: note owner: %w", err)
	}
	return userID, userHost, nil
}

// logf is the package's diagnostic hook; tests may replace it.
var logf = log.Printf

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func marshalFields(f []ProfileField) []byte {
	if f == nil {
		f = []ProfileField{}
	}
	b, _ := json.Marshal(f)
	return b
}

// EmojiRef is one custom emoji referenced by a remote note.
type EmojiRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func marshalAny(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func marshalStrings(s []string) []byte {
	b, _ := json.Marshal(s)
	return b
}

// FollowCount returns how many followers or followees a user has.
func (s *Store) FollowCount(ctx context.Context, userID string, followers bool) (int, error) {
	var n int
	var err error
	if followers {
		err = s.pool.QueryRow(ctx, "SELECT count(*) FROM followings WHERE followee_id = $1", userID).Scan(&n)
	} else {
		err = s.pool.QueryRow(ctx, "SELECT count(*) FROM followings WHERE follower_id = $1", userID).Scan(&n)
	}
	if err != nil {
		return 0, fmt.Errorf("federation: count follows: %w", err)
	}
	return n, nil
}

// FollowRefs returns the actor URLs of a user's followers or followees,
// newest follow first, at most limit, skipping anything at or after
// cursor.
func (s *Store) FollowRefs(ctx context.Context, userID string, followers bool, limit int, cursor string) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	var rows pgx.Rows
	var err error
	if followers {
		rows, err = s.pool.Query(ctx, `
SELECT COALESCE(u.uri, ''), u.id, f.id FROM followings f JOIN users u ON u.id = f.follower_id
WHERE f.followee_id = $1 AND ($2 = '' OR f.id < $2)
ORDER BY f.id DESC LIMIT $3
		`, userID, cursor, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
SELECT COALESCE(u.uri, ''), u.id, f.id FROM followings f JOIN users u ON u.id = f.followee_id
WHERE f.follower_id = $1 AND ($2 = '' OR f.id < $2)
ORDER BY f.id DESC LIMIT $3
		`, userID, cursor, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("federation: list follows: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var uri, id, followID string
		if err := rows.Scan(&uri, &id, &followID); err != nil {
			return nil, err
		}
		if uri == "" {
			uri = "" + id
		}
		out = append(out, uri)
	}
	return out, rows.Err()
}

// UserByID loads any user row, local or remote, by id.
func (s *Store) UserByID(ctx context.Context, id string) (*RemoteUser, error) {
	u := &RemoteUser{}
	var host *string
	err := s.pool.QueryRow(ctx, `
SELECT id, username, host, COALESCE(uri, ''), COALESCE(inbox, ''),
       COALESCE(shared_inbox, ''), COALESCE(followers_uri, ''),
       COALESCE(featured_uri, ''), COALESCE(key_id, ''), COALESCE(public_key_pem, ''),
       is_locked, is_bot, is_deleted,
       COALESCE(moved_to_uri, ''), COALESCE(also_known_as, '[]'::jsonb)
FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &host, &u.URI, &u.Inbox, &u.SharedInbox,
		&u.FollowersURI, &u.FeaturedURI, &u.KeyID, &u.PublicKeyPEM,
		&u.IsLocked, &u.IsBot, &u.IsDeleted,
		&u.MovedToURI, &u.AlsoKnownAs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRemoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("federation: load user: %w", err)
	}
	if host != nil {
		u.Host = *host
	}
	return u, nil
}
