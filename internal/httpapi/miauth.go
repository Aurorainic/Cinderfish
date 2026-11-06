package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleMiAuthCheck is POST /api/miauth/{session}/check, the endpoint a
// third-party application polls after opening the approval page.
//
// It needs no credential: the session string itself is the capability, and it
// only ever answers once. The first successful poll returns the token and the
// user; later polls answer `{ok:false}` so the token is not leaked twice.
func (s *Server) handleMiAuthCheck(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	session := r.PathValue("session")
	token, user, err := s.tokens.FindUnfetchedBySession(r.Context(), session)
	if err != nil {
		if errors.Is(err, account.ErrInvalidToken) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false})
			return
		}
		writeInternalServerError(w)
		return
	}
	if err := s.tokens.MarkFetched(r.Context(), token.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"token": token.Token,
		"user":  packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID)),
	})
}
