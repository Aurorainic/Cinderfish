package httpapi

import "net/http"

// handleStats is POST /api/stats: instance-wide counters. Misskey exposes this
// for the about page and for charts that are not implemented here; the fields
// below are the subset this instance can answer cheaply.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}
	var (
		users, notes, originalNotes, reactions, following int64
		instances, driveFiles, driveUsage                 int64
	)
	err := s.pool.QueryRow(r.Context(), `
SELECT
  (SELECT count(*) FROM users WHERE host IS NULL AND NOT is_suspended),
  (SELECT count(*) FROM notes WHERE deleted_at IS NULL),
  (SELECT count(*) FROM notes WHERE deleted_at IS NULL AND renote_id IS NULL),
  (SELECT count(*) FROM note_reactions),
  (SELECT count(*) FROM followings),
  (SELECT count(DISTINCT host) FROM users WHERE host IS NOT NULL),
  (SELECT count(*) FROM drive_files),
  (SELECT COALESCE(sum(size), 0) FROM drive_files)
`).Scan(&users, &notes, &originalNotes, &reactions, &following, &instances, &driveFiles, &driveUsage)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"usersCount":         users,
		"notesCount":         notes,
		"originalNotesCount": originalNotes,
		"originalUsersCount": users,
		"reactionsCount":     reactions,
		"followingCount":     following,
		"instancesCount":     instances,
		"driveFiles":         driveFiles,
		"driveUsage":         driveUsage,
	})
}
