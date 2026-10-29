package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleUsersNotes implements POST /api/users/notes: the public and home
// notes of one local user, newest first. Followers-only notes are not
// included, matching the endpoint's anonymous scope.
func (s *Server) handleUsersNotes(w http.ResponseWriter, r *http.Request) {
	if s.timeline == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		UserID   string `json:"userId"`
		Username string `json:"username"`
		Limit    int    `json:"limit"`
		SinceID  string `json:"sinceId"`
		UntilID  string `json:"untilId"`
		WithFiles   *bool `json:"withFiles"`
		WithReplies *bool `json:"withReplies"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	var (
		user *account.User
		err  error
	)
	switch {
	case body.UserID != "":
		user, err = s.accounts.GetByID(r.Context(), body.UserID)
	case body.Username != "":
		user, err = s.accounts.GetByUsername(r.Context(), body.Username)
	default:
		writeAPIError(w, invalidParamError("userId or username is required"))
		return
	}
	if err != nil || user == nil {
		if err != nil && !errors.Is(err, account.ErrUserNotFound) {
			writeInternalServerError(w)
			return
		}
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	viewerID := ""
	if info, _ := s.authenticate(r.Context(), r, bodyAsMap(r)); info != nil {
		viewerID = info.User.ID
	}

	// Misskey's users/notes defaults: withFiles is false and withReplies
	// is true unless the request says otherwise.
	withFiles := body.WithFiles != nil && *body.WithFiles
	withReplies := body.WithReplies == nil || *body.WithReplies

	notes, err := s.timeline.UserNotes(r.Context(), user.ID, body.Limit, body.SinceID, body.UntilID, withFiles, withReplies)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNoteDetail(r.Context(), s, n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}

// applyPinnedNotes fills pinnedNoteIds and pinnedNotes with the user's
// pin list in stored order, at most five notes packed.
func (s *Server) applyPinnedNotes(ctx context.Context, userID string, packed map[string]any) {
	if s.accounts == nil || s.notes == nil {
		return
	}
	ids, err := s.accounts.PinnedNoteIDs(ctx, userID)
	if err != nil {
		return
	}
	packed["pinnedNoteIds"] = ids
	notes := make([]any, 0, len(ids))
	for i, id := range ids {
		if i >= 5 {
			break
		}
		n, err := s.notes.Get(ctx, id)
		if err != nil {
			continue
		}
		notes = append(notes, packNoteDetail(ctx, s, n, s.notePackContext(ctx, n, "")))
	}
	packed["pinnedNotes"] = notes
}

// applyUserCounts fills the profile counters that the UserDetailed schema
// always carries. The follows are counted live so they stay correct even
// for rows written before the counters existed.
func (s *Server) applyUserCounts(ctx context.Context, userID string, packed map[string]any) {
	if s.pool == nil {
		return
	}
	var followers, following int
	if err := s.pool.QueryRow(ctx,
		`SELECT followers_count, following_count FROM users WHERE id = $1`, userID).
		Scan(&followers, &following); err == nil {
		packed["followersCount"] = followers
		packed["followingCount"] = following
	}
	var notes int
	if err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM notes
WHERE user_id = $1 AND deleted_at IS NULL AND local_only = FALSE
  AND visibility IN ('public', 'home')`, userID).Scan(&notes); err == nil {
		packed["notesCount"] = notes
	}
}
