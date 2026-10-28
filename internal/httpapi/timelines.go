package httpapi

import (
	"net/http"

	"milnfish/internal/note"
)

// timelineRequest is the shared parameter set of the timeline endpoints.
type timelineRequest struct {
	Limit                 int    `json:"limit"`
	SinceID               string `json:"sinceId"`
	UntilID               string `json:"untilId"`
	SinceDate             *int64 `json:"sinceDate"`
	UntilDate             *int64 `json:"untilDate"`
	WithFiles             bool   `json:"withFiles"`
	WithRenotes           bool   `json:"withRenotes"`
	WithReplies           bool   `json:"withReplies"`
	AllowPartial          bool   `json:"allowPartial"`
	IncludeMyRenotes      *bool  `json:"includeMyRenotes"`
	IncludeRenotedMyNotes *bool  `json:"includeRenotedMyNotes"`
	IncludeLocalRenotes   *bool  `json:"includeLocalRenotes"`
}

// handleLocalTimeline implements POST /api/notes/local-timeline.
func (s *Server) handleLocalTimeline(w http.ResponseWriter, r *http.Request) {
	s.serveTimeline(w, r, note.TimelineLocal, false)
}

// handleHybridTimeline implements POST /api/notes/hybrid-timeline.
func (s *Server) handleHybridTimeline(w http.ResponseWriter, r *http.Request) {
	s.serveTimeline(w, r, note.TimelineSocial, true)
}

// handleGlobalTimeline implements POST /api/notes/global-timeline.
func (s *Server) handleGlobalTimeline(w http.ResponseWriter, r *http.Request) {
	s.serveTimeline(w, r, note.TimelineGlobal, false)
}

// handleHomeTimeline implements POST /api/notes/timeline, the home
// timeline of the authenticated user.
func (s *Server) handleHomeTimeline(w http.ResponseWriter, r *http.Request) {
	s.serveTimeline(w, r, note.TimelineHome, true)
}

// serveTimeline answers a timeline request.
//
// requireCredential marks the endpoints Misskey protects: the hybrid and
// home timelines need a signed in user.
func (s *Server) serveTimeline(w http.ResponseWriter, r *http.Request, kind note.TimelineKind, requireCredential bool) {
	if s.timeline == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body timelineRequest
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	if requireCredential && viewerID == "" {
		writeAuthenticationError(w)
		return
	}

	params := note.TimelineParams{
		Kind:        kind,
		ViewerID:    viewerID,
		Limit:       body.Limit,
		SinceID:     body.SinceID,
		UntilID:     body.UntilID,
		WithFiles:   body.WithFiles,
		WithRenotes: body.WithRenotes,
		WithReplies: body.WithReplies,
		// Misskey defaults these to true.
		IncludeMyRenotes:      true,
		IncludeRenotedMyNotes: true,
		IncludeLocalRenotes:   true,
	}
	if body.IncludeMyRenotes != nil {
		params.IncludeMyRenotes = *body.IncludeMyRenotes
	}
	if body.IncludeRenotedMyNotes != nil {
		params.IncludeRenotedMyNotes = *body.IncludeRenotedMyNotes
	}
	if body.IncludeLocalRenotes != nil {
		params.IncludeLocalRenotes = *body.IncludeLocalRenotes
	}

	// Date cursors are translated into id cursors.
	if params.SinceID == "" && body.SinceDate != nil {
		params.SinceID, _ = s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.SinceDate))
	}
	if params.UntilID == "" && body.UntilDate != nil {
		params.UntilID, _ = s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.UntilDate))
	}

	notes, err := s.timeline.Timeline(r.Context(), params)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not load the timeline.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNoteDetail(r.Context(), s, n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}
