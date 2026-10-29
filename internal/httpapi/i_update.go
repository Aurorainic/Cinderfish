package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// handleIUpdate implements POST /api/i/update: it writes the editable
// profile fields of the authenticated user.
//
// name and description are optional; when provided, an empty string
// clears the field. avatarId names a drive file: empty clears the
// avatar, a non-empty id must belong to the caller.
func (s *Server) handleIUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		AvatarID    *string `json:"avatarId"`
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

	if body.AvatarID != nil && *body.AvatarID != "" {
		if err := s.checkOwnAvatarFile(r, info.User.ID, *body.AvatarID); err != nil {
			writeAPIError(w, err)
			return
		}
	}

	if err := s.accounts.UpdateProfile(r.Context(), info.User.ID, body.Name, body.Description, body.AvatarID); err != nil {
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
