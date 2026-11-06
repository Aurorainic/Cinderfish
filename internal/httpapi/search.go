package httpapi

import (
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// viewerIDForSearch resolves the optional credential of a read endpoint. A
// missing or bad token means anonymous, never a 401: search works for guests,
// who simply see public and home notes only.
func (s *Server) viewerIDForSearch(r *http.Request) string {
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		return info.User.ID
	}
	return ""
}

// writeNoteList packs a note slice the same way the timelines do.
func (s *Server) writeNoteList(w http.ResponseWriter, r *http.Request, notes []*note.Note, viewerID string) {
	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNoteDetail(r.Context(), s, n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleNotesSearch is POST /api/notes/search.
//
// There is no full-text index, so the query runs as the same ILIKE fallback
// Misskey uses without a search engine. Visibility, mutes, blocks and muted
// instance hosts apply, which also means a muted author cannot be found
// through search.
func (s *Server) handleNotesSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query     string        `json:"query"`
		SinceID   string        `json:"sinceId"`
		UntilID   string        `json:"untilId"`
		Limit     int           `json:"limit"`
		Offset    int           `json:"offset"`
		Host      nullableField `json:"host"`
		UserID    string        `json:"userId"`
		WithFiles bool          `json:"withFiles"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Query == "" {
		writeAPIError(w, invalidParamError("query is required"))
		return
	}
	if s.timeline == nil {
		writeInternalServerError(w)
		return
	}

	viewerID := s.viewerIDForSearch(r)
	notes, err := s.timeline.Search(r.Context(), note.SearchParams{
		Query:     body.Query,
		ViewerID:  viewerID,
		Limit:     body.Limit,
		Offset:    body.Offset,
		SinceID:   body.SinceID,
		UntilID:   body.UntilID,
		Host:      body.Host.asUpdatePointer(),
		UserID:    body.UserID,
		WithFiles: body.WithFiles,
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writeNoteList(w, r, notes, viewerID)
}

// handleNotesSearchByTag is POST /api/notes/search-by-tag. Only the single
// `tag` form is implemented; the upstream `query` array-of-arrays form has no
// client in this port.
func (s *Server) handleNotesSearchByTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tag       string `json:"tag"`
		SinceID   string `json:"sinceId"`
		UntilID   string `json:"untilId"`
		Limit     int    `json:"limit"`
		WithFiles bool   `json:"withFiles"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Tag == "" {
		writeAPIError(w, invalidParamError("tag is required"))
		return
	}
	if s.timeline == nil {
		writeInternalServerError(w)
		return
	}

	viewerID := s.viewerIDForSearch(r)
	notes, err := s.timeline.SearchByTag(r.Context(), note.SearchByTagParams{
		Tag:       body.Tag,
		ViewerID:  viewerID,
		Limit:     body.Limit,
		SinceID:   body.SinceID,
		UntilID:   body.UntilID,
		WithFiles: body.WithFiles,
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writeNoteList(w, r, notes, viewerID)
}

// handleUsersSearch is POST /api/users/search.
func (s *Server) handleUsersSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query   string        `json:"query"`
		Origin  string        `json:"origin"`
		Host    nullableField `json:"host"`
		Limit   int           `json:"limit"`
		Offset  int           `json:"offset"`
		SinceID string        `json:"sinceId"`
		UntilID string        `json:"untilId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Query == "" {
		writeAPIError(w, invalidParamError("query is required"))
		return
	}
	switch body.Origin {
	case "", "combined", "local", "remote":
	default:
		writeAPIError(w, invalidParamError("origin must be local, remote or combined"))
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	users, err := s.accounts.SearchUsers(r.Context(), account.UserSearchParams{
		Query:    body.Query,
		Origin:   body.Origin,
		Host:     body.Host.asUpdatePointer(),
		Limit:    body.Limit,
		Offset:   body.Offset,
		SinceID:  body.SinceID,
		UntilID:  body.UntilID,
		ViewerID: s.viewerIDForSearch(r),
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		packed := packUserDetailed(u, u.CreatedAt, false, loadProfile(r.Context(), s, u.ID))
		// Result cards show note/follower counts, so they are filled the same
		// way users/show fills them. Search returns at most `limit` rows, which
		// keeps the extra queries bounded.
		s.applyUserCounts(r.Context(), u.ID, packed)
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}
