package account

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProfileDetailsPatch carries the optional profile metadata fields. A nil
// pointer leaves the value unchanged; a pointer to an empty value clears it.
// They are stored inside `user_profiles.settings`.
type ProfileDetailsPatch struct {
	Location           *string
	Birthday           *string
	Lang               *string
	Fields             *[]ProfileField
	NoCrawle           *bool
	PreventAiLearning  *bool
	IsExplorable       *bool
	HideOnlineStatus   *bool
	PublicReactions    *bool
	AlwaysMarkNsfw     *bool
	AutoSensitive      *bool
	CarefulBot         *bool
	AutoAcceptFollowed *bool
	InjectFeaturedNote *bool
}

// UpdateProfileDetails merges the provided metadata keys into the profile's
// settings JSON.
func (s *Service) UpdateProfileDetails(ctx context.Context, userID string, patch ProfileDetailsPatch) error {
	settings := map[string]any{}
	if patch.Location != nil {
		settings["location"] = *patch.Location
	}
	if patch.Birthday != nil {
		settings["birthday"] = *patch.Birthday
	}
	if patch.Lang != nil {
		settings["lang"] = *patch.Lang
	}
	if patch.Fields != nil {
		settings["fields"] = *patch.Fields
	}
	for key, value := range map[string]*bool{
		"noCrawle":           patch.NoCrawle,
		"preventAiLearning":  patch.PreventAiLearning,
		"isExplorable":       patch.IsExplorable,
		"hideOnlineStatus":   patch.HideOnlineStatus,
		"publicReactions":    patch.PublicReactions,
		"alwaysMarkNsfw":     patch.AlwaysMarkNsfw,
		"autoSensitive":      patch.AutoSensitive,
		"carefulBot":         patch.CarefulBot,
		"autoAcceptFollowed": patch.AutoAcceptFollowed,
		"injectFeaturedNote": patch.InjectFeaturedNote,
	} {
		if value != nil {
			settings[key] = *value
		}
	}
	if len(settings) == 0 {
		return nil
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("account: marshal profile details: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO user_profiles (user_id, settings) VALUES ($1, $2::jsonb)
ON CONFLICT (user_id) DO UPDATE SET settings = user_profiles.settings || EXCLUDED.settings
`, userID, raw); err != nil {
		return fmt.Errorf("account: update profile details: %w", err)
	}
	return nil
}
