package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
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

	if avatarID := body.AvatarID.asUpdatePointer(); avatarID != nil && *avatarID != "" {
		if err := s.checkOwnAvatarFile(r, info.User.ID, *avatarID); err != nil {
			writeAPIError(w, err)
			return
		}
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
