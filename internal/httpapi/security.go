package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleIChangePassword implements POST /api/i/change-password.
//
// Misskey's paramDef requires both passwords; the new one only has to be
// non-empty. A wrong current password answers the signin id-only envelope so
// clients that already branch on it keep working.
func (s *Server) handleIChangePassword(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
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
	if body.NewPassword == "" {
		writeAPIError(w, invalidParamError("newPassword is required"))
		return
	}

	err = s.accounts.ChangePassword(r.Context(), info.User.ID, body.CurrentPassword, body.NewPassword)
	if errors.Is(err, account.ErrInvalidCredentials) {
		writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
		return
	}
	if err != nil {
		writeAPIError(w, internalError("Could not change the password."))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleIRegenerateToken implements POST /api/i/regenerate-token.
//
// Misskey answers an empty body and pushes `myTokenRegenerated` over
// Streaming; Cinderfish returns the new token as well so the browser can
// replace the credential it has stored without waiting for a push.
func (s *Server) handleIRegenerateToken(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Password string `json:"password"`
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

	token, err := s.accounts.RegenerateToken(r.Context(), info.User.ID, body.Password)
	if errors.Is(err, account.ErrInvalidCredentials) {
		writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
		return
	}
	if err != nil {
		writeAPIError(w, internalError("Could not regenerate the token."))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}

// handleISigninHistory implements POST /api/i/signin-history.
func (s *Server) handleISigninHistory(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Limit   int    `json:"limit"`
		SinceID string `json:"sinceId"`
		UntilID string `json:"untilId"`
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

	records, err := s.accounts.SigninHistory(r.Context(), info.User.ID, body.Limit, body.UntilID, body.SinceID)
	if err != nil {
		writeAPIError(w, internalError("Could not read the signin history."))
		return
	}

	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, map[string]any{
			"id":        rec.ID,
			"createdAt": rec.CreatedAt.UTC().Format(time.RFC3339Nano),
			"ip":        rec.IP,
			"headers":   map[string]any{},
			"success":   rec.Success,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
