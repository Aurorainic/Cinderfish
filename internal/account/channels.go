package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Channel is one local-only themed board any local user may post to.
// Mirrors Misskey's Channel schema (autogen/types.ts L5017); PinnedNoteIDs
// is stored as jsonb, matching the v17-era shapes elsewhere.
type Channel struct {
	ID                    string
	UserID                *string
	Name                  string
	Description           *string
	BannerID              *string
	PinnedNoteIDs         []string
	Color                 string
	IsArchived            bool
	NotesCount            int
	UsersCount            int
	IsSensitive           bool
	AllowRenoteToExternal bool
	LastNotedAt           *time.Time
	CreatedAt             time.Time
}

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrChannelNotFound            = errors.New("account: channel not found")
	ErrChannelNameInvalid         = errors.New("account: channel name invalid")
	ErrChannelDescInvalid         = errors.New("account: channel description invalid")
	ErrAlreadyFollowingChannel    = errors.New("account: already following channel")
	ErrChannelAlreadyFavorited    = errors.New("account: channel already favorited")
	ErrAlreadyMutingChannel       = errors.New("account: already muting channel")
	ErrNotMutingChannel           = errors.New("account: not muting channel")
	ErrMuteExpiresInPast          = errors.New("account: mute expiry in the past")
	ErrCannotRenoteOutsideChannel = errors.New("account: cannot renote outside the channel")
)

// channelColorMax is the paramDef bound; Misskey's color is 1..16 chars.
const channelColorMax = 16

// ChannelInput carries channels/create's write path.
type ChannelInput struct {
	Name                  string
	Description           *string
	BannerID              *string
	Color                 string
	IsSensitive           bool
	AllowRenoteToExternal bool
}

// const channelColumns = ... (see below)

const channelColumns = `id, user_id, name, description, banner_id, pinned_note_ids,
       color, is_archived, notes_count, users_count, is_sensitive,
       allow_renote_to_external, last_noted_at, created_at`

// channelColumnsC is channelColumns qualified for queries that join the
// channels table under the alias c.
const channelColumnsC = `c.id, c.user_id, c.name, c.description, c.banner_id, c.pinned_note_ids,
       c.color, c.is_archived, c.notes_count, c.users_count, c.is_sensitive,
       c.allow_renote_to_external, c.last_noted_at, c.created_at`

func scanChannel(row pgx.Row) (*Channel, error) {
	var c Channel
	var userID, bannerID *string
	var description *string
	var pinned []byte
	err := row.Scan(&c.ID, &userID, &c.Name, &description, &bannerID, &pinned,
		&c.Color, &c.IsArchived, &c.NotesCount, &c.UsersCount, &c.IsSensitive,
		&c.AllowRenoteToExternal, &c.LastNotedAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChannelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan channel: %w", err)
	}
	c.UserID = userID
	c.BannerID = bannerID
	c.Description = description
	c.PinnedNoteIDs = decodeTextArray(pinned)
	return &c, nil
}

// CreateChannel makes a channel owned by userID. The banner must be one
// of the caller's drive files, as upstream looks the file up by
// (id, userId) and answers NO_SUCH_FILE otherwise.
func (s *Service) CreateChannel(ctx context.Context, userID string, in ChannelInput) (*Channel, error) {
	if in.Name == "" || len([]rune(in.Name)) > 128 {
		return nil, ErrChannelNameInvalid
	}
	if in.Description != nil && len([]rune(*in.Description)) > 2048 {
		return nil, ErrChannelDescInvalid
	}
	color := in.Color
	if color == "" {
		color = "#86b300"
	}
	if len([]rune(color)) > channelColorMax {
		return nil, ErrChannelNameInvalid
	}
	var bannerID *string
	if in.BannerID != nil && *in.BannerID != "" {
		var owner string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id FROM drive_files WHERE id = $1`, *in.BannerID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != userID) {
			return nil, ErrNoSuchBannerFile
		}
		if err != nil {
			return nil, fmt.Errorf("account: banner lookup: %w", err)
		}
		bannerID = in.BannerID
	}

	id := idgen.New()
	_, err := s.pool.Exec(ctx, `
INSERT INTO channels (id, user_id, name, description, banner_id, color,
                      is_sensitive, allow_renote_to_external)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, id, userID, in.Name, in.Description, bannerID, color,
		in.IsSensitive, in.AllowRenoteToExternal)
	if err != nil {
		return nil, fmt.Errorf("account: create channel: %w", err)
	}
	return s.GetChannel(ctx, id)
}

// GetChannel loads any channel — channels/show is public and archived
// channels stay readable, as upstream answers them too.
func (s *Service) GetChannel(ctx context.Context, channelID string) (*Channel, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+channelColumns+` FROM channels WHERE id = $1`, channelID)
	return scanChannel(row)
}

// GetActiveChannel loads a channel that accepts new notes: the lookup
// notes/create performs before attaching a note.
func (s *Service) GetActiveChannel(ctx context.Context, channelID string) (*Channel, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+channelColumns+` FROM channels WHERE id = $1 AND is_archived = false`, channelID)
	c, err := scanChannel(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChannelNotFound
	}
	return c, err
}

// OwnChannel loads a channel the user may edit — the owner. Cinderfish
// models no separate moderator flag beyond is_admin, which the handler
// checks for the upstream moderator path.
func (s *Service) OwnChannel(ctx context.Context, channelID, ownerID string) (*Channel, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+channelColumns+` FROM channels WHERE id = $1`, channelID)
	c, err := scanChannel(row)
	if err != nil {
		return nil, err
	}
	if c.UserID == nil || *c.UserID != ownerID {
		return nil, ErrChannelNotFound
	}
	return c, nil
}

// scanChannels drains a channel query.
func scanChannels(rows pgx.Rows) ([]*Channel, error) {
	defer rows.Close()
	out := []*Channel{}
	for rows.Next() {
		var c Channel
		var userID, bannerID *string
		var description *string
		var pinned []byte
		if err := rows.Scan(&c.ID, &userID, &c.Name, &description, &bannerID, &pinned,
			&c.Color, &c.IsArchived, &c.NotesCount, &c.UsersCount, &c.IsSensitive,
			&c.AllowRenoteToExternal, &c.LastNotedAt, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan channel: %w", err)
		}
		c.UserID = userID
		c.BannerID = bannerID
		c.Description = description
		c.PinnedNoteIDs = decodeTextArray(pinned)
		out = append(out, &c)
	}
	return out, rows.Err()
}

// UpdateChannel applies channels/update's partial patch. Cinderfish fixes
// the upstream bug that a null bannerId cannot clear the banner: an
// explicit null clears it, an omitted bannerId leaves it.
func (s *Service) UpdateChannel(ctx context.Context, channelID, editorID string, isAdmin bool, patch *ChannelPatch) (*Channel, error) {
	var c *Channel
	var err error
	if isAdmin {
		c, err = s.GetChannel(ctx, channelID)
	} else {
		c, err = s.OwnChannel(ctx, channelID, editorID)
	}
	if errors.Is(err, ErrChannelNotFound) && !isAdmin {
		// A non-owner receives ACCESS_DENIED only when the channel exists;
		// a missing one answers NO_SUCH_CHANNEL, as upstream checks first.
		var probe error
		_, probe = s.GetChannel(ctx, channelID)
		if probe == nil {
			return nil, ErrChannelAccessDenied()
		}
		return nil, ErrChannelNotFound
	}
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return c, nil
	}

	if patch.Name != nil {
		if *patch.Name == "" || len([]rune(*patch.Name)) > 128 {
			return nil, ErrChannelNameInvalid
		}
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET name = $1 WHERE id = $2`, *patch.Name, c.ID); err != nil {
			return nil, fmt.Errorf("account: update channel name: %w", err)
		}
		c.Name = *patch.Name
	}
	if patch.Description != nil && patch.DescriptionPresent {
		trimmed := *patch.Description
		if len([]rune(trimmed)) > 2048 {
			return nil, ErrChannelDescInvalid
		}
		var desc *string
		if trimmed != "" {
			desc = &trimmed
		}
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET description = $1 WHERE id = $2`, desc, c.ID); err != nil {
			return nil, fmt.Errorf("account: update channel description: %w", err)
		}
		c.Description = desc
	}
	if patch.BannerID != nil && patch.BannerIDPresent {
		if *patch.BannerID == "" || *patch.BannerID == "-" {
			// Cinderfish fixes upstream's silent no-clear: null drops the banner.
			if _, err := s.pool.Exec(ctx, `UPDATE channels SET banner_id = NULL WHERE id = $1`, c.ID); err != nil {
				return nil, fmt.Errorf("account: clear channel banner: %w", err)
			}
			c.BannerID = nil
		} else {
			var owner string
			err := s.pool.QueryRow(ctx,
				`SELECT user_id FROM drive_files WHERE id = $1`, *patch.BannerID).Scan(&owner)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != editorID) {
				return nil, ErrNoSuchBannerFile
			}
			if err != nil {
				return nil, fmt.Errorf("account: banner lookup: %w", err)
			}
			if _, err := s.pool.Exec(ctx, `UPDATE channels SET banner_id = $1 WHERE id = $2`, *patch.BannerID, c.ID); err != nil {
				return nil, fmt.Errorf("account: set channel banner: %w", err)
			}
			c.BannerID = patch.BannerID
		}
	}
	if patch.IsArchived != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET is_archived = $1 WHERE id = $2`, *patch.IsArchived, c.ID); err != nil {
			return nil, fmt.Errorf("account: archive channel: %w", err)
		}
		c.IsArchived = *patch.IsArchived
	}
	if patch.PinnedNoteIDs != nil {
		pinned, err := json.Marshal(patch.PinnedNoteIDs)
		if err != nil {
			return nil, fmt.Errorf("account: pin notes: %w", err)
		}
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET pinned_note_ids = $1 WHERE id = $2`, pinned, c.ID); err != nil {
			return nil, fmt.Errorf("account: pin notes: %w", err)
		}
		c.PinnedNoteIDs = patch.PinnedNoteIDs
	}
	if patch.Color != nil {
		trimmed := strings.TrimSpace(*patch.Color)
		if trimmed == "" || len([]rune(trimmed)) > channelColorMax {
			return nil, ErrChannelNameInvalid
		}
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET color = $1 WHERE id = $2`, trimmed, c.ID); err != nil {
			return nil, fmt.Errorf("account: update channel color: %w", err)
		}
		c.Color = trimmed
	}
	if patch.IsSensitive != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET is_sensitive = $1 WHERE id = $2`, *patch.IsSensitive, c.ID); err != nil {
			return nil, fmt.Errorf("account: update channel sensitivity: %w", err)
		}
		c.IsSensitive = *patch.IsSensitive
	}
	if patch.AllowRenoteToExternal != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE channels SET allow_renote_to_external = $1 WHERE id = $2`, *patch.AllowRenoteToExternal, c.ID); err != nil {
			return nil, fmt.Errorf("account: update channel renote policy: %w", err)
		}
		c.AllowRenoteToExternal = *patch.AllowRenoteToExternal
	}
	return c, nil
}

// ChannelPatch is the partial merge shape of channels/update.
type ChannelPatch struct {
	Name                  *string
	Description           *string
	DescriptionPresent    bool
	BannerID              *string
	BannerIDPresent       bool
	IsArchived            *bool
	PinnedNoteIDs         []string
	Color                 *string
	IsSensitive           *bool
	AllowRenoteToExternal *bool
}

// ErrChannelAccessDenied marks an edit of someone else's channel.
func ErrChannelAccessDenied() error {
	return errChannelAccessDenied
}

var errChannelAccessDenied = errors.New("account: channel access denied")

// FollowChannel, UnfollowChannel and the two favourites follow the
// fanout semantics upstream gives them: they affect the home timeline and
// the packed isFollowing/isFavorited flags only.
func (s *Service) FollowChannel(ctx context.Context, userID, channelID string) error {
	if _, err := s.GetChannel(ctx, channelID); err != nil {
		return err
	}
	exists, err := s.IsFollowingChannel(ctx, userID, channelID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyFollowingChannel
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO channel_followings (follower_id, followee_id) VALUES ($1, $2)
`, userID, channelID)
	if err != nil {
		return fmt.Errorf("account: follow channel: %w", err)
	}
	return nil
}

func (s *Service) UnfollowChannel(ctx context.Context, userID, channelID string) error {
	// Idempotent, as upstream's delete() with no error is.
	_, err := s.pool.Exec(ctx, `
DELETE FROM channel_followings WHERE follower_id = $1 AND followee_id = $2
`, userID, channelID)
	if err != nil {
		return fmt.Errorf("account: unfollow channel: %w", err)
	}
	return nil
}

// IsFollowingChannel reports whether userID follows channelID.
func (s *Service) IsFollowingChannel(ctx context.Context, userID, channelID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM channel_followings WHERE follower_id = $1 AND followee_id = $2)
`, userID, channelID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check channel follow: %w", err)
	}
	return f, nil
}

// FollowedChannelIDsOf returns every channel id the user follows.
func (s *Service) FollowedChannelIDsOf(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT followee_id FROM channel_followings WHERE follower_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: followed channels: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) FavoriteChannel(ctx context.Context, userID, channelID string) error {
	if _, err := s.GetChannel(ctx, channelID); err != nil {
		return err
	}
	// Upstream inserts without an existence check on the favourite row, so
	// a duplicate surfaces as a 500 from the unique index; Cinderfish
	// answers ALREADY_FAVORITED-style client error instead, recorded in
	// docs/HANDOVER.md.
	_, err := s.pool.Exec(ctx, `
INSERT INTO channel_favorites (id, user_id, channel_id) VALUES ($1, $2, $3)
`, idgen.New(), userID, channelID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrChannelAlreadyFavorited
		}
		return fmt.Errorf("account: favorite channel: %w", err)
	}
	return nil
}

func (s *Service) UnfavoriteChannel(ctx context.Context, userID, channelID string) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM channel_favorites WHERE user_id = $1 AND channel_id = $2
`, userID, channelID)
	if err != nil {
		return fmt.Errorf("account: unfavorite channel: %w", err)
	}
	return nil
}

// MuteChannel records a channel mute, optionally expiring. Duplicate
// muting is ALREADY_MUTING_CHANNEL; a past expiry is rejected, as upstream
// validates before the insert.
func (s *Service) MuteChannel(ctx context.Context, userID, channelID string, expiresAt *time.Time) error {
	if _, err := s.GetChannel(ctx, channelID); err != nil {
		return err
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return ErrMuteExpiresInPast
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM channel_mutings WHERE user_id = $1 AND channel_id = $2
      AND (expires_at IS NULL OR expires_at > now())
)
`, userID, channelID).Scan(&exists); err != nil {
		return fmt.Errorf("account: check channel mute: %w", err)
	}
	if exists {
		return ErrAlreadyMutingChannel
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO channel_mutings (id, user_id, channel_id, expires_at)
VALUES ($1, $2, $3, $4)
`, idgen.New(), userID, channelID, expiresAt); err != nil {
		return fmt.Errorf("account: mute channel: %w", err)
	}
	return nil
}

// UnmuteChannel removes a mute; muting nothing is NOT_MUTING_CHANNEL.
func (s *Service) UnmuteChannel(ctx context.Context, userID, channelID string) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM channel_mutings WHERE user_id = $1 AND channel_id = $2
`, userID, channelID)
	if err != nil {
		return fmt.Errorf("account: unmute channel: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotMutingChannel
	}
	return nil
}

// MutedChannelIDsOf returns the channel ids the user currently mutes.
func (s *Service) MutedChannelIDsOf(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT channel_id FROM channel_mutings
WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > now())
`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: muted channels: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SearchChannels answers channels/search. An empty query lists every
// non-archived channel, as upstream skips the ILIKE filter then.
func (s *Service) SearchChannels(ctx context.Context, query, typ string, limit int, sinceID, untilID string) ([]*Channel, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	conds := []string{"is_archived = false"}
	args := []any{}
	if query != "" {
		pattern := "%" + escapeLikeString(query) + "%"
		args = append(args, pattern)
		if typ == "nameOnly" {
			conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
		} else {
			conds = append(conds, fmt.Sprintf("(name ILIKE $%d OR COALESCE(description, '') ILIKE $%d)", len(args), len(args)))
		}
	}
	if sinceID != "" {
		args = append(args, sinceID)
		conds = append(conds, fmt.Sprintf("id > $%d", len(args)))
	}
	if untilID != "" {
		args = append(args, untilID)
		conds = append(conds, fmt.Sprintf("id < $%d", len(args)))
	}
	order := "DESC"
	if sinceID != "" && untilID == "" {
		order = "ASC"
	}
	args = append(args, limit)
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumns+` FROM channels
WHERE `+strings.Join(conds, " AND ")+`
ORDER BY id `+order+` LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("account: search channels: %w", err)
	}
	return scanChannels(rows)
}

// escapeLikeString escapes the LIKE wildcards of a channel search.
func escapeLikeString(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

// FeaturedChannels answers channels/featured: the ten most recently
// posted-to, non-archived channels.
func (s *Service) FeaturedChannels(ctx context.Context) ([]*Channel, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumns+` FROM channels
WHERE last_noted_at IS NOT NULL AND is_archived = false
ORDER BY last_noted_at DESC LIMIT 10
`)
	if err != nil {
		return nil, fmt.Errorf("account: featured channels: %w", err)
	}
	return scanChannels(rows)
}

// FollowedChannels pages the channels one user follows, by channel id.
func (s *Service) FollowedChannels(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Channel, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumnsC+` FROM channels c
JOIN channel_followings f ON f.followee_id = c.id
WHERE f.follower_id = $1
ORDER BY c.id DESC
LIMIT $2
`, userID, limit)
	_ = sinceID
	_ = untilID
	if err != nil {
		return nil, fmt.Errorf("account: followed channels: %w", err)
	}
	return scanChannels(rows)
}

// OwnedChannels pages the channels one user owns, non-archived only, as
// upstream's owned filters them.
func (s *Service) OwnedChannels(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Channel, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumns+` FROM channels
WHERE user_id = $1 AND is_archived = false
ORDER BY id DESC
LIMIT $2
`, userID, limit)
	_ = sinceID
	_ = untilID
	if err != nil {
		return nil, fmt.Errorf("account: owned channels: %w", err)
	}
	return scanChannels(rows)
}

// FavoriteChannels returns the channels the user favourited, unpaginated
// like upstream's my-favorites.
func (s *Service) FavoriteChannels(ctx context.Context, userID string) ([]*Channel, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumnsC+` FROM channels c
JOIN channel_favorites f ON f.channel_id = c.id
WHERE f.user_id = $1
ORDER BY f.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: favorite channels: %w", err)
	}
	return scanChannels(rows)
}

// MutedChannels returns the channels the user currently mutes.
func (s *Service) MutedChannels(ctx context.Context, userID string) ([]*Channel, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+channelColumnsC+` FROM channels c
JOIN channel_mutings m ON m.channel_id = c.id
WHERE m.user_id = $1 AND (m.expires_at IS NULL OR m.expires_at > now())
`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: muted channels: %w", err)
	}
	return scanChannels(rows)
}

// HasChannelFavorite reports whether userID favourited channelID, for the
// packed isFavorited flag.
func (s *Service) HasChannelFavorite(ctx context.Context, userID, channelID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM channel_favorites WHERE user_id = $1 AND channel_id = $2)
`, userID, channelID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check channel favorite: %w", err)
	}
	return f, nil
}
