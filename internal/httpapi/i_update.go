package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// Profile field length caps, matching Misskey's user profile limits.
const (
	maxProfileNameLength        = 50
	maxProfileDescriptionLength = 500
)

// nullableField carries an optional request field: an absent key leaves
// the stored value unchanged, while an explicit JSON null and an empty
// string both clear it — Misskey's own client sends null to clear a
// field.
type nullableField struct {
	present bool
	value   *string
}

func (f *nullableField) UnmarshalJSON(data []byte) error {
	f.present = true
	if string(data) == "null" {
		f.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	f.value = &s
	return nil
}

// provided reports whether the request carried the field at all.
func (f *nullableField) provided() bool { return f.present }

// asUpdatePointer maps the field onto UpdateProfile's *string
// convention: nil is "not provided", a pointer to "" clears, a pointer
// to the value sets.
func (f *nullableField) asUpdatePointer() *string {
	if !f.present {
		return nil
	}
	if f.value == nil {
		s := ""
		return &s
	}
	return f.value
}

// handleIUpdate implements POST /api/i/update: it writes the editable
// profile fields of the authenticated user.
//
// name, description and avatarId are all optional. A provided null or
// empty string clears the field; avatarId otherwise names a drive file
// belonging to the caller. name is capped at 50 and description at 500
// characters.
func (s *Server) handleIUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        nullableField `json:"name"`
		Description nullableField `json:"description"`
		AvatarID    nullableField `json:"avatarId"`
		BannerID    nullableField `json:"bannerId"`
		// Misskey's users/update privacy switches.
		FollowersVisibility nullableField `json:"followersVisibility"`
		FollowingVisibility nullableField `json:"followingVisibility"`
		// mutedInstances takes a JSON array of hosts; null or [] clears it.
		// RawMessage keeps "absent" distinguishable from "cleared".
		MutedInstances json.RawMessage `json:"mutedInstances"`
		// Profile metadata, stored in user_profiles.settings.
		Location nullableField   `json:"location"`
		Birthday nullableField   `json:"birthday"`
		Lang     nullableField   `json:"lang"`
		Fields   json.RawMessage `json:"fields"`
		// Boolean profile switches. Pointers keep "absent" distinct from false.
		NoCrawle           *bool `json:"noCrawle"`
		PreventAiLearning  *bool `json:"preventAiLearning"`
		IsExplorable       *bool `json:"isExplorable"`
		HideOnlineStatus   *bool `json:"hideOnlineStatus"`
		PublicReactions    *bool `json:"publicReactions"`
		AlwaysMarkNsfw     *bool `json:"alwaysMarkNsfw"`
		AutoSensitive      *bool `json:"autoSensitive"`
		CarefulBot         *bool `json:"carefulBot"`
		AutoAcceptFollowed *bool `json:"autoAcceptFollowed"`
		InjectFeaturedNote *bool `json:"injectFeaturedNote"`
		// Manual follower approval. Stored in users.is_locked, the same column
		// the inbox reads for an inbound Follow.
		IsLocked *bool `json:"isLocked"`
		// Notification types the user does not want to see.
		MutingNotificationTypes json.RawMessage `json:"mutingNotificationTypes"`
		// Email notifications: which reasons are also mailed, and whether
		// instance announcements are mailed.
		EmailNotificationTypes   json.RawMessage `json:"emailNotificationTypes"`
		ReceiveAnnouncementEmail *bool           `json:"receiveAnnouncementEmail"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	if body.Name.provided() {
		if v := body.Name.asUpdatePointer(); v != nil && utf8.RuneCountInString(*v) > maxProfileNameLength {
			writeAPIError(w, invalidParamError("name is too long"))
			return
		}
	}
	if body.Description.provided() {
		if v := body.Description.asUpdatePointer(); v != nil && utf8.RuneCountInString(*v) > maxProfileDescriptionLength {
			writeAPIError(w, invalidParamError("description is too long"))
			return
		}
	}

	for _, field := range []struct {
		name  string
		value *string
	}{
		{"followersVisibility", body.FollowersVisibility.asUpdatePointer()},
		{"followingVisibility", body.FollowingVisibility.asUpdatePointer()},
	} {
		if field.value != nil && !validFollowVisibility(*field.value) {
			writeAPIError(w, invalidParamError(field.name+" must be public, followers or private"))
			return
		}
	}

	var mutedInstances []string
	if len(body.MutedInstances) > 0 {
		hosts, apiErr := parseMutedInstances(body.MutedInstances)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		mutedInstances = hosts
	}

	details, apiErr := parseProfileDetails(body.Location, body.Birthday, body.Lang, body.Fields)
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	details.NoCrawle = body.NoCrawle
	details.PreventAiLearning = body.PreventAiLearning
	details.IsExplorable = body.IsExplorable
	details.HideOnlineStatus = body.HideOnlineStatus
	details.PublicReactions = body.PublicReactions
	details.AlwaysMarkNsfw = body.AlwaysMarkNsfw
	details.AutoSensitive = body.AutoSensitive
	details.CarefulBot = body.CarefulBot
	details.AutoAcceptFollowed = body.AutoAcceptFollowed
	details.InjectFeaturedNote = body.InjectFeaturedNote
	if len(body.MutingNotificationTypes) > 0 {
		muting, apiErr := parseMutingNotificationTypes(body.MutingNotificationTypes)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		details.MutingNotificationTypes = &muting
	}
	if len(body.EmailNotificationTypes) > 0 {
		mailTypes, apiErr := parseEmailNotificationTypes(body.EmailNotificationTypes)
		if apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		details.EmailNotificationTypes = &mailTypes
	}
	details.ReceiveAnnouncementEmail = body.ReceiveAnnouncementEmail

	if err := s.accounts.UpdateProfileDetails(r.Context(), info.User.ID, details); err != nil {
		writeInternalServerError(w)
		return
	}

	if avatarID := body.AvatarID.asUpdatePointer(); avatarID != nil && *avatarID != "" {
		if err := s.checkOwnProfileFile(r, info.User.ID, *avatarID); err != nil {
			writeAPIError(w, err)
			return
		}
	}
	if bannerID := body.BannerID.asUpdatePointer(); bannerID != nil && *bannerID != "" {
		if err := s.checkOwnProfileFile(r, info.User.ID, *bannerID); err != nil {
			writeAPIError(w, err)
			return
		}
	}

	if len(body.MutedInstances) > 0 {
		if err := s.accounts.UpdateMutedInstances(r.Context(), info.User.ID, mutedInstances); err != nil {
			writeInternalServerError(w)
			return
		}
	}

	if err := s.accounts.UpdateVisibility(r.Context(), info.User.ID, body.FollowersVisibility.asUpdatePointer(), body.FollowingVisibility.asUpdatePointer()); err != nil {
		writeInternalServerError(w)
		return
	}

	if body.IsLocked != nil {
		if err := s.accounts.SetLocked(r.Context(), info.User.ID, *body.IsLocked); err != nil {
			writeInternalServerError(w)
			return
		}
	}

	if err := s.accounts.UpdateProfile(r.Context(), info.User.ID, body.Name.asUpdatePointer(), body.Description.asUpdatePointer(), body.AvatarID.asUpdatePointer(), body.BannerID.asUpdatePointer()); err != nil {
		if errors.Is(err, media.ErrNotFound) || errors.Is(err, account.ErrNoSuchAvatarFile) || errors.Is(err, account.ErrNoSuchBannerFile) {
			writeAPIError(w, &APIError{
				Message: "No such file.", Code: "NO_SUCH_FILE",
				ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeInternalServerError(w)
		return
	}

	packed := packMeDetailed(info.User, info.User.CreatedAt, info.Token == nil, loadProfile(r.Context(), s, info.User.ID))
	s.applyUserCounts(r.Context(), info.User.ID, packed)
	s.applyUnreadNotifications(r.Context(), info.User.ID, packed)
	s.applyUnreadAnnouncements(r.Context(), info.User.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// maxMutedInstances caps the list so a crafted request cannot grow the
// profile row without bound.
const maxMutedInstances = 200

// Profile metadata limits, following Misskey's user schema.
const (
	maxProfileLocationLength = 50
	maxProfileLangLength     = 16
	maxProfileFields         = 16
	maxProfileFieldName      = 255
	maxProfileFieldValue     = 1024
)

// parseProfileDetails validates the metadata fields of i/update. Empty
// strings clear a value; `fields` accepts an array or null.
func parseProfileDetails(location, birthday, lang nullableField, rawFields json.RawMessage) (account.ProfileDetailsPatch, *APIError) {
	var out account.ProfileDetailsPatch
	if location.provided() {
		value := ""
		if v := location.asUpdatePointer(); v != nil {
			value = strings.TrimSpace(*v)
		}
		if utf8.RuneCountInString(value) > maxProfileLocationLength {
			return out, invalidParamError("location is too long")
		}
		out.Location = &value
	}
	if birthday.provided() {
		value := ""
		if v := birthday.asUpdatePointer(); v != nil {
			value = strings.TrimSpace(*v)
		}
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return out, invalidParamError("birthday must be YYYY-MM-DD")
			}
		}
		out.Birthday = &value
	}
	if lang.provided() {
		value := ""
		if v := lang.asUpdatePointer(); v != nil {
			value = strings.TrimSpace(*v)
		}
		if value != "" {
			if len(value) > maxProfileLangLength || !validLangTag(value) {
				return out, invalidParamError("lang is not a valid language tag")
			}
		}
		out.Lang = &value
	}
	if len(rawFields) > 0 {
		fields := []account.ProfileField{}
		if string(rawFields) != "null" {
			if err := json.Unmarshal(rawFields, &fields); err != nil {
				return out, invalidParamError("fields must be an array of {name, value}")
			}
		}
		if len(fields) > maxProfileFields {
			return out, invalidParamError("fields has too many entries")
		}
		clean := make([]account.ProfileField, 0, len(fields))
		for _, f := range fields {
			name := strings.TrimSpace(f.Name)
			if name == "" {
				continue
			}
			value := strings.TrimSpace(f.Value)
			if utf8.RuneCountInString(name) > maxProfileFieldName || utf8.RuneCountInString(value) > maxProfileFieldValue {
				return out, invalidParamError("a profile field is too long")
			}
			clean = append(clean, account.ProfileField{Name: name, Value: value})
		}
		out.Fields = &clean
	}
	return out, nil
}

// validLangTag accepts BCP-47-ish tags such as "ja", "zh-CN" or "en-US".
func validLangTag(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
		default:
			return false
		}
	}
	return true
}

// knownNotificationTypes lists the kinds this backend produces. Misskey has
// more (follow, pollEnded, …), but accepting a type that can never arrive
// would make the setting look functional when it is not.
var knownNotificationTypes = map[string]struct{}{
	"mention":  {},
	"reply":    {},
	"renote":   {},
	"quote":    {},
	"reaction": {},
}

// parseMutingNotificationTypes validates i/update's mutingNotificationTypes.
func parseMutingNotificationTypes(raw json.RawMessage) ([]string, *APIError) {
	var types []string
	if string(raw) != "null" {
		if err := json.Unmarshal(raw, &types); err != nil {
			return nil, invalidParamError("mutingNotificationTypes must be an array of notification types")
		}
	}
	if len(types) > len(knownNotificationTypes) {
		return nil, invalidParamError("mutingNotificationTypes has too many entries")
	}
	out := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if _, ok := knownNotificationTypes[t]; !ok {
			return nil, invalidParamError("unknown notification type: " + t)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out, nil
}

// emailNotificationTypes are the reasons Firefish mails about. The list is
// smaller than the in-app one: reactions, renotes and quotes are not mailed,
// matching upstream's own email notification types.
var emailNotificationTypes = map[string]struct{}{
	"mention":              {},
	"reply":                {},
	"follow":               {},
	"receiveFollowRequest": {},
}

// parseEmailNotificationTypes validates i/update's emailNotificationTypes.
func parseEmailNotificationTypes(raw json.RawMessage) ([]string, *APIError) {
	var types []string
	if string(raw) != "null" {
		if err := json.Unmarshal(raw, &types); err != nil {
			return nil, invalidParamError("emailNotificationTypes must be an array of notification types")
		}
	}
	out := make([]string, 0, len(types))
	seen := make(map[string]struct{}, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if _, ok := emailNotificationTypes[t]; !ok {
			return nil, invalidParamError("unknown email notification type: " + t)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out, nil
}

// parseMutedInstances turns the `mutedInstances` request value into a clean
// host list. Hosts are lowercased and trimmed; a scheme, a path or a space is
// rejected rather than silently mangled.
func parseMutedInstances(raw json.RawMessage) ([]string, *APIError) {
	var hosts []string
	if string(raw) == "null" {
		return []string{}, nil
	}
	if err := json.Unmarshal(raw, &hosts); err != nil {
		return nil, invalidParamError("mutedInstances must be an array of hosts")
	}
	if len(hosts) > maxMutedInstances {
		return nil, invalidParamError("mutedInstances has too many entries")
	}
	out := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		h := strings.ToLower(strings.TrimSpace(host))
		if h == "" {
			continue
		}
		if len(h) > 253 || strings.ContainsAny(h, "/ \\@") {
			return nil, invalidParamError("mutedInstances entry is not a host: " + host)
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out, nil
}

// validFollowVisibility reports whether v is one of Misskey's three
// follow-list visibility values.
func validFollowVisibility(v string) bool {
	switch v {
	case "public", "followers", "private":
		return true
	}
	return false
}

// checkOwnProfileFile verifies that the given drive file exists and belongs to
// the caller before it can become their avatar or banner.
func (s *Server) checkOwnProfileFile(r *http.Request, userID, fileID string) *APIError {
	if s.drive == nil {
		return &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		}
	}
	f, err := s.drive.Get(r.Context(), fileID)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return &APIError{
				Message: "No such file.", Code: "NO_SUCH_FILE",
				ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
			}
		}
		return &APIError{
			Message: "Could not load the file.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		}
	}
	if f.UserID != userID {
		return &APIError{
			Message: "The file is not yours.", Code: "ACCESS_DENIED",
			ID: errIDForbidden, Kind: "permission", Status: http.StatusForbidden,
		}
	}
	return nil
}
