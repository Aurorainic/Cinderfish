package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
		// Misskey's users/update privacy switches.
		FollowersVisibility nullableField `json:"followersVisibility"`
		FollowingVisibility nullableField `json:"followingVisibility"`
		// mutedInstances takes a JSON array of hosts; null or [] clears it.
		// RawMessage keeps "absent" distinguishable from "cleared".
		MutedInstances json.RawMessage `json:"mutedInstances"`
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

	if avatarID := body.AvatarID.asUpdatePointer(); avatarID != nil && *avatarID != "" {
		if err := s.checkOwnAvatarFile(r, info.User.ID, *avatarID); err != nil {
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

	if err := s.accounts.UpdateProfile(r.Context(), info.User.ID, body.Name.asUpdatePointer(), body.Description.asUpdatePointer(), body.AvatarID.asUpdatePointer()); err != nil {
		if errors.Is(err, media.ErrNotFound) || errors.Is(err, account.ErrNoSuchAvatarFile) {
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
	writeJSON(w, http.StatusOK, packed)
}

// maxMutedInstances caps the list so a crafted request cannot grow the
// profile row without bound.
const maxMutedInstances = 200

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

// checkOwnAvatarFile verifies that the given drive file exists and
// belongs to the caller before it can become their avatar.
func (s *Server) checkOwnAvatarFile(r *http.Request, userID, fileID string) *APIError {
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
