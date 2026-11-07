package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// errIDNoSuchFileAttachedNotes is Misskey's NO_SUCH_FILE id for
// drive/files/attached-notes.
const errIDNoSuchFileAttachedNotes = "c118ece3-2e4b-4296-99d1-51756e32d232"

// handleDriveAttachedNotes implements POST /api/drive/files/attached-notes:
// the notes that reference one of the caller's drive files.
func (s *Server) handleDriveAttachedNotes(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil || s.notes == nil || s.pool == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		FileID    string `json:"fileId"`
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
	if body.FileID == "" {
		writeAPIError(w, invalidParamError("fileId is required"))
		return
	}

	file, err := s.drive.Get(r.Context(), body.FileID)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			writeAttachedNotesNoSuchFile(w)
			return
		}
		writeInternalServerError(w)
		return
	}
	// Misskey answers NO_SUCH_FILE for someone else's file, so the endpoint
	// does not confirm that a private upload exists.
	if file.UserID != info.User.ID {
		writeAttachedNotesNoSuchFile(w)
		return
	}

	limit := body.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	args := []any{body.FileID}
	conds := []string{"$1 = ANY(file_ids)", "deleted_at IS NULL"}
	if body.SinceID != "" {
		args = append(args, body.SinceID)
		conds = append(conds, "id > $"+strconv.Itoa(len(args)))
	}
	if body.UntilID != "" {
		args = append(args, body.UntilID)
		conds = append(conds, "id < $"+strconv.Itoa(len(args)))
	}
	if body.SinceDate != nil {
		args = append(args, time.UnixMilli(*body.SinceDate))
		conds = append(conds, "created_at >= $"+strconv.Itoa(len(args)))
	}
	if body.UntilDate != nil {
		args = append(args, time.UnixMilli(*body.UntilDate))
		conds = append(conds, "created_at <= $"+strconv.Itoa(len(args)))
	}
	order := "id DESC"
	if body.SinceID != "" && body.UntilID == "" && body.UntilDate == nil {
		order = "id ASC"
	}
	args = append(args, limit)
	query := "SELECT id FROM notes WHERE "
	for i, cond := range conds {
		if i > 0 {
			query += " AND "
		}
		query += cond
	}
	query += " ORDER BY " + order + " LIMIT $" + strconv.Itoa(len(args))

	rows, err := s.pool.Query(r.Context(), query, args...)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			writeInternalServerError(w)
			return
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeInternalServerError(w)
		return
	}
	rows.Close()

	out := make([]any, 0, len(ids))
	for _, id := range ids {
		n, err := s.notes.Get(r.Context(), id)
		if err != nil || n == nil {
			continue
		}
		visible, err := s.notes.VisibleTo(r.Context(), n, info.User.ID)
		if err != nil || !visible {
			continue
		}
		out = append(out, packNote(n, s.notePackContext(r.Context(), n, info.User.ID)))
	}
	writeJSON(w, http.StatusOK, out)
}

func writeAttachedNotesNoSuchFile(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "No such file.", Code: "NO_SUCH_FILE",
		ID: errIDNoSuchFileAttachedNotes, Kind: "client", Status: http.StatusNotFound,
	})
}
