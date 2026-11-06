package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/note"
)

// handleNotesFavoritesCreate implements POST /api/notes/favorites/create.
//
// Misskey caps this endpoint at 20 calls an hour and answers with the packed
// NoteFavorite; checkRateLimit enforces the same window.
func (s *Server) handleNotesFavoritesCreate(w http.ResponseWriter, r *http.Request) {
	if s.favorites == nil || s.notes == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
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

	if !s.checkRateLimit(w, info.User.ID, limitFavoritesCreate) {
		return
	}
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}

	fav, err := s.favorites.Add(r.Context(), info.User.ID, body.NoteID)
	if err != nil {
		if errors.Is(err, note.ErrNoSuchFavorite) {
			writeAPIError(w, &APIError{
				Message: "No such note.", Code: "NO_SUCH_NOTE",
				ID: errIDNoSuchNote, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}

	n, err := s.notes.Get(r.Context(), fav.NoteID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	pc := s.notePackContext(r.Context(), n, info.User.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        fav.ID,
		"createdAt": fav.CreatedAt.UTC().Format(time.RFC3339Nano),
		"noteId":    fav.NoteID,
		"note":      packNote(n, pc),
	})
}

// handleNotesFavoritesDelete implements POST /api/notes/favorites/delete.
// Deleting a favorite that is not there succeeds, as upstream does.
func (s *Server) handleNotesFavoritesDelete(w http.ResponseWriter, r *http.Request) {
	if s.favorites == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		NoteID string `json:"noteId"`
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
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}

	if err := s.favorites.Remove(r.Context(), info.User.ID, body.NoteID); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// handleIFavorites implements POST /api/i/favorites: the caller's favorites,
// newest first, each with its packed note.
func (s *Server) handleIFavorites(w http.ResponseWriter, r *http.Request) {
	if s.favorites == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Limit     int    `json:"limit"`
		SinceID   string `json:"sinceId"`
		UntilID   string `json:"untilId"`
		SinceDate *int64 `json:"sinceDate"`
		UntilDate *int64 `json:"untilDate"`
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

	q := note.FavoriteQuery{
		Limit:   body.Limit,
		SinceID: body.SinceID,
		UntilID: body.UntilID,
	}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		q.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		q.UntilDate = &t
	}

	list, err := s.favorites.List(r.Context(), info.User.ID, q)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(list))
	for _, fav := range list {
		pc := s.notePackContext(r.Context(), fav.Note, info.User.ID)
		out = append(out, map[string]any{
			"id":        fav.ID,
			"createdAt": fav.CreatedAt.UTC().Format(time.RFC3339Nano),
			"noteId":    fav.NoteID,
			"note":      packNote(fav.Note, pc),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
