package httpapi

import (
	"net/http"
)

// handleAPShow implements POST /api/ap/show: resolve an arbitrary
// ActivityPub URL into a note or a user.
//
// It is what the remote-interaction page and "open this URL in my instance"
// flows call. The object is fetched (signed, policy-checked) and stored the
// same way inbound federation stores it, so the answer is a normal packed
// entity the client already knows how to render.
func (s *Server) handleAPShow(w http.ResponseWriter, r *http.Request) {
	if s.federationInbox == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		URI string `json:"uri"`
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
	if body.URI == "" {
		writeAPIError(w, invalidParamError("uri is required"))
		return
	}

	kind, id, err := s.federationInbox.ImportObject(r.Context(), body.URI)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such object.", Code: "NO_SUCH_OBJECT",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	switch kind {
	case "User":
		user, err := s.accounts.GetByID(r.Context(), id)
		if err != nil || user == nil {
			writeInternalServerError(w)
			return
		}
		packed := packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID))
		s.packMovedTo(r.Context(), user, packed)
		writeJSON(w, http.StatusOK, map[string]any{"type": "User", "object": packed})
	case "Note":
		if s.notes == nil {
			writeInternalServerError(w)
			return
		}
		n, err := s.notes.Get(r.Context(), id)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		pc := s.notePackContext(r.Context(), n, info.User.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"type":   "Note",
			"object": packNoteDetail(r.Context(), s, n, pc),
		})
	default:
		writeInternalServerError(w)
	}
}
