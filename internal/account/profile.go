package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// Profile is the editable part of a user's profile. Empty strings mean
// the column is NULL.
type Profile struct {
	Name        string
	Description string
	AvatarURL   string
	BannerURL   string
	// FollowersVisibility / FollowingVisibility are Misskey's
	// "public" | "followers" | "private" switches. An empty value reads as
	// "public", which is the column default.
	FollowersVisibility string
	FollowingVisibility string
	// MutedInstances lists instance hosts whose notes this user never
	// wants to see. It lives in `user_profiles.settings` so it does not
	// need a migration of its own.
	MutedInstances []string
	// TwoFactorEnabled reports whether TOTP is active for this account.
	// UsePasswordLessLogin is stored for schema parity but no security key
	// can be registered yet, so it always reads false.
	TwoFactorEnabled     bool
	UsePasswordLessLogin bool
	// Profile metadata. These live in `user_profiles.settings` as JSON so the
	// table does not grow a column per Misskey field.
	Location string
	Birthday string
	Lang     string
	Fields   []ProfileField
	// Boolean profile switches, also stored in `settings`.
	NoCrawle           bool
	PreventAiLearning  bool
	IsExplorable       bool
	HideOnlineStatus   bool
	PublicReactions    bool
	AlwaysMarkNsfw     bool
	AutoSensitive      bool
	CarefulBot         bool
	AutoAcceptFollowed bool
	InjectFeaturedNote bool
	// MutedNotificationTypes hides those notification kinds from the list
	// and the unread badge.
	MutedNotificationTypes []string
	// Email is the address on file; empty when none is set. Verified is
	// only meaningful when Email is non-empty.
	Email         string
	EmailVerified bool
	// EmailNotificationTypes lists the notification reasons that are also
	// mailed; ReceiveAnnouncementEmail subscribes to instance notices.
	EmailNotificationTypes   []string
	ReceiveAnnouncementEmail bool
	// IsLocked means new followers need manual approval. It lives in
	// users.is_locked (the same column a remote actor's locked flag uses), so
	// the federation code and the profile agree.
	IsLocked bool
}

// ProfileField is one row of Misskey's "fields" (profile metadata) list.
type ProfileField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ErrNoSuchAvatarFile is returned when an avatar file id does not name
// an existing drive file.
var ErrNoSuchAvatarFile = errors.New("account: no such avatar file")

// ErrNoSuchBannerFile is the banner counterpart.
var ErrNoSuchBannerFile = errors.New("account: no such banner file")

// resolveImageURL turns an empty id into NULL (clearing the column) and a file
// id into the drive URL. label only shapes the error message.
func (s *Service) resolveImageURL(ctx context.Context, fileID, label string) (any, error) {
	if fileID == "" {
		return nil, nil
	}
	if s.drive == nil {
		return nil, fmt.Errorf("account: no drive service attached to resolve %s", label)
	}
	f, err := s.drive.Get(ctx, fileID)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			if label == "banner" {
				return nil, ErrNoSuchBannerFile
			}
			return nil, ErrNoSuchAvatarFile
		}
		return nil, fmt.Errorf("account: resolve %s: %w", label, err)
	}
	return f.URL, nil
}

// AttachDrive wires the drive service used to resolve avatar files.
// UpdateProfile reports an error when an avatar is set without one.
func (s *Service) AttachDrive(drive *media.DriveService) {
	s.drive = drive
}

// GetProfile loads a user's profile. A user without a profile row reads
// as the zero value instead of failing.
func (s *Service) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	var name, description, avatarURL, bannerURL, followersVis, followingVis, email *string
	var settings []byte
	var twoFactorEnabled, usePasswordLessLogin, emailVerified, isLocked bool
	err := s.pool.QueryRow(ctx, `
SELECT name, description, avatar_url, banner_url, followers_visibility, following_visibility, settings,
       two_factor_enabled, use_password_less_login, email, email_verified,
       COALESCE((SELECT is_locked FROM users WHERE id = $1), FALSE)
FROM user_profiles WHERE user_id = $1
`, userID).Scan(&name, &description, &avatarURL, &bannerURL, &followersVis, &followingVis, &settings,
		&twoFactorEnabled, &usePasswordLessLogin, &email, &emailVerified, &isLocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return &Profile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("account: get profile: %w", err)
	}
	var stored struct {
		MutedInstances           []string       `json:"mutedInstances"`
		Location                 string         `json:"location"`
		Birthday                 string         `json:"birthday"`
		Lang                     string         `json:"lang"`
		Fields                   []ProfileField `json:"fields"`
		NoCrawle                 bool           `json:"noCrawle"`
		PreventAiLearning        bool           `json:"preventAiLearning"`
		IsExplorable             *bool          `json:"isExplorable"`
		HideOnlineStatus         bool           `json:"hideOnlineStatus"`
		PublicReactions          *bool          `json:"publicReactions"`
		AlwaysMarkNsfw           bool           `json:"alwaysMarkNsfw"`
		AutoSensitive            bool           `json:"autoSensitive"`
		CarefulBot               bool           `json:"carefulBot"`
		AutoAcceptFollowed       bool           `json:"autoAcceptFollowed"`
		InjectFeaturedNote       bool           `json:"injectFeaturedNote"`
		MutingNotificationTypes  []string       `json:"mutingNotificationTypes"`
		EmailNotificationTypes   []string       `json:"emailNotificationTypes"`
		ReceiveAnnouncementEmail bool           `json:"receiveAnnouncementEmail"`
	}
	if len(settings) > 0 {
		_ = json.Unmarshal(settings, &stored)
	}
	// Misskey's default is explorable; only an explicit false turns it off.
	isExplorable := true
	if stored.IsExplorable != nil {
		isExplorable = *stored.IsExplorable
	}
	return &Profile{
		Name:                     derefString(name),
		Description:              derefString(description),
		AvatarURL:                derefString(avatarURL),
		BannerURL:                derefString(bannerURL),
		FollowersVisibility:      derefString(followersVis),
		FollowingVisibility:      derefString(followingVis),
		MutedInstances:           stored.MutedInstances,
		TwoFactorEnabled:         twoFactorEnabled,
		UsePasswordLessLogin:     usePasswordLessLogin,
		Location:                 stored.Location,
		Birthday:                 stored.Birthday,
		Lang:                     stored.Lang,
		Fields:                   stored.Fields,
		NoCrawle:                 stored.NoCrawle,
		PreventAiLearning:        stored.PreventAiLearning,
		IsExplorable:             isExplorable,
		HideOnlineStatus:         stored.HideOnlineStatus,
		PublicReactions:          publicReactions(stored.PublicReactions),
		AlwaysMarkNsfw:           stored.AlwaysMarkNsfw,
		AutoSensitive:            stored.AutoSensitive,
		CarefulBot:               stored.CarefulBot,
		AutoAcceptFollowed:       stored.AutoAcceptFollowed,
		InjectFeaturedNote:       stored.InjectFeaturedNote,
		MutedNotificationTypes:   stored.MutingNotificationTypes,
		Email:                    derefString(email),
		EmailVerified:            emailVerified,
		IsLocked:                 isLocked,
		EmailNotificationTypes:   stored.EmailNotificationTypes,
		ReceiveAnnouncementEmail: stored.ReceiveAnnouncementEmail,
	}, nil
}

// UpdateProfile writes the editable profile fields. A nil pointer leaves
// the field unchanged; a non-nil empty string clears it. avatarFileID is
// resolved through the attached drive service: an empty string clears
// avatar_url, a non-empty id must name an existing drive file whose URL
// becomes avatar_url.
func (s *Service) UpdateProfile(ctx context.Context, userID string, name, description, avatarFileID, bannerFileID *string) error {
	var sets []string
	var args []any

	if name != nil {
		args = append(args, nullIfEmpty(*name))
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if description != nil {
		args = append(args, nullIfEmpty(*description))
		sets = append(sets, fmt.Sprintf("description = $%d", len(args)))
	}
	if avatarFileID != nil {
		url, err := s.resolveImageURL(ctx, *avatarFileID, "avatar")
		if err != nil {
			return err
		}
		args = append(args, url)
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", len(args)))
	}
	if bannerFileID != nil {
		url, err := s.resolveImageURL(ctx, *bannerFileID, "banner")
		if err != nil {
			return err
		}
		args = append(args, url)
		sets = append(sets, fmt.Sprintf("banner_url = $%d", len(args)))
	}

	if len(sets) == 0 {
		return nil
	}
	args = append(args, userID)
	query := "UPDATE user_profiles SET " + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE user_id = $%d", len(args))
	if _, err := s.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("account: update profile: %w", err)
	}
	return nil
}

// UpdateVisibility writes the two follow-list visibility switches. A nil
// pointer leaves that switch unchanged; the caller validates the values
// against Misskey's "public" | "followers" | "private" set.
func (s *Service) UpdateVisibility(ctx context.Context, userID string, followers, following *string) error {
	var sets []string
	var args []any
	if followers != nil {
		args = append(args, *followers)
		sets = append(sets, fmt.Sprintf("followers_visibility = $%d", len(args)))
	}
	if following != nil {
		args = append(args, *following)
		sets = append(sets, fmt.Sprintf("following_visibility = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, userID)
	query := "UPDATE user_profiles SET " + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE user_id = $%d", len(args))
	if _, err := s.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("account: update visibility: %w", err)
	}
	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// UpdateMutedInstances replaces the muted-instance list. The hosts are
// expected to be already normalised by the caller (lowercase, no scheme).
func (s *Service) UpdateMutedInstances(ctx context.Context, userID string, hosts []string) error {
	if hosts == nil {
		hosts = []string{}
	}
	payload, err := json.Marshal(map[string]any{"mutedInstances": hosts})
	if err != nil {
		return fmt.Errorf("account: encode muted instances: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO user_profiles (user_id, settings) VALUES ($1, $2::jsonb)
ON CONFLICT (user_id) DO UPDATE SET settings = user_profiles.settings || EXCLUDED.settings
`, userID, payload)
	if err != nil {
		return fmt.Errorf("account: update muted instances: %w", err)
	}
	return nil
}

// SetLocked switches manual follower approval on or off. The flag lives in
// users.is_locked so the ActivityPub inbox and the API agree on it.
func (s *Service) SetLocked(ctx context.Context, userID string, locked bool) error {
	if _, err := s.pool.Exec(ctx, `UPDATE users SET is_locked = $2 WHERE id = $1`, userID, locked); err != nil {
		return fmt.Errorf("account: set locked: %w", err)
	}
	return nil
}

// AnnouncementMailRecipients lists the verified addresses that asked to receive
// instance announcements. The subscription lives in user_profiles.settings, so
// the query filters there rather than in a column of its own.
func (s *Service) AnnouncementMailRecipients(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT email FROM user_profiles
WHERE email IS NOT NULL AND email <> '' AND email_verified
  AND COALESCE((settings->>'receiveAnnouncementEmail')::boolean, FALSE)
ORDER BY user_id
`)
	if err != nil {
		return nil, fmt.Errorf("account: announcement recipients: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("account: scan announcement recipient: %w", err)
		}
		out = append(out, email)
	}
	return out, rows.Err()
}

// publicReactions defaults to true, matching Misskey's profile settings. An
// absent key means the account never changed the switch, not that it opted out.
func publicReactions(stored *bool) bool {
	if stored == nil {
		return true
	}
	return *stored
}
