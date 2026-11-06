package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// exportNote is one note in the account export.
type exportNote struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	CW         *string  `json:"cw"`
	Visibility string   `json:"visibility"`
	ReplyID    *string  `json:"replyId"`
	RenoteID   *string  `json:"renoteId"`
	FileIDs    []string `json:"fileIds"`
	Tags       []string `json:"tags"`
	CreatedAt  string   `json:"createdAt"`
}

// exportAccount is the document `i/export` answers with. It is a Cinderfish
// backup, not Misskey's zip: the fields are the ones this instance stores, so
// the file can be re-imported by tooling written against this contract later.
type exportAccount struct {
	ExportedAt string           `json:"exportedAt"`
	Format     string           `json:"format"`
	User       map[string]any   `json:"user"`
	Notes      []exportNote     `json:"notes"`
	Followings []map[string]any `json:"followings"`
	Followers  []map[string]any `json:"followers"`
	Favorites  []map[string]any `json:"favorites"`
	DriveFiles []map[string]any `json:"driveFiles"`
}

// handleIExport is POST /api/i/export. It builds the whole backup in memory
// and streams it as an attachment. The dataset of a personal instance is small
// enough for this; a large instance would need a background job (which is what
// Firefish does).
func (s *Server) handleIExport(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}
	uid := info.User.ID

	doc := exportAccount{
		ExportedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Format:     "cinderfish-account-export@1",
		User:       packMeDetailed(info.User, info.User.CreatedAt, false, loadProfile(r.Context(), s, uid)),
		Notes:      []exportNote{},
		Followings: []map[string]any{},
		Followers:  []map[string]any{},
		Favorites:  []map[string]any{},
		DriveFiles: []map[string]any{},
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT id, COALESCE(text, ''), cw, visibility, reply_id, renote_id,
       COALESCE(file_ids, '{}'), COALESCE(tags, '[]'::jsonb), created_at
FROM notes WHERE user_id = $1 AND deleted_at IS NULL ORDER BY id
`, uid)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	for rows.Next() {
		var n exportNote
		var files []string
		var tags []byte
		var created time.Time
		if err := rows.Scan(&n.ID, &n.Text, &n.CW, &n.Visibility, &n.ReplyID, &n.RenoteID,
			&files, &tags, &created); err != nil {
			rows.Close()
			writeInternalServerError(w)
			return
		}
		n.FileIDs = files
		_ = json.Unmarshal(tags, &n.Tags)
		n.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		doc.Notes = append(doc.Notes, n)
	}
	rows.Close()

	relation := func(query, id string) []map[string]any {
		out := []map[string]any{}
		rs, err := s.pool.Query(r.Context(), query, id)
		if err != nil {
			return out
		}
		defer rs.Close()
		for rs.Next() {
			var username string
			var host *string
			var created time.Time
			if err := rs.Scan(&username, &host, &created); err != nil {
				continue
			}
			entry := map[string]any{
				"username":  username,
				"host":      nil,
				"createdAt": created.UTC().Format(time.RFC3339Nano),
			}
			if host != nil {
				entry["host"] = *host
			}
			out = append(out, entry)
		}
		return out
	}
	doc.Followings = relation(`
SELECT u.username, u.host, f.created_at FROM followings f
JOIN users u ON u.id = f.followee_id WHERE f.follower_id = $1 ORDER BY f.created_at
`, uid)
	doc.Followers = relation(`
SELECT u.username, u.host, f.created_at FROM followings f
JOIN users u ON u.id = f.follower_id WHERE f.followee_id = $1 ORDER BY f.created_at
`, uid)

	if rs, err := s.pool.Query(r.Context(), `
SELECT note_id, created_at FROM note_favorites WHERE user_id = $1 ORDER BY created_at
`, uid); err == nil {
		defer rs.Close()
		for rs.Next() {
			var noteID string
			var created time.Time
			if err := rs.Scan(&noteID, &created); err != nil {
				continue
			}
			doc.Favorites = append(doc.Favorites, map[string]any{
				"noteId": noteID, "createdAt": created.UTC().Format(time.RFC3339Nano),
			})
		}
	}

	if rs, err := s.pool.Query(r.Context(), `
SELECT name, content_type, size, created_at FROM drive_files
WHERE user_id = $1 ORDER BY created_at
`, uid); err == nil {
		defer rs.Close()
		for rs.Next() {
			var name, contentType string
			var size int64
			var created time.Time
			if err := rs.Scan(&name, &contentType, &size, &created); err != nil {
				continue
			}
			doc.DriveFiles = append(doc.DriveFiles, map[string]any{
				"name": name, "type": contentType, "size": size,
				"createdAt": created.UTC().Format(time.RFC3339Nano),
			})
		}
	}

	filename := fmt.Sprintf("cinderfish-export-%s-%s.json", info.User.Username,
		time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		// The headers are already out; the truncated body is all we can do.
		return
	}
}

// handleIDeleteAccount is POST /api/i/delete-account. The password is checked
// again because a stolen token alone must not be enough to destroy the account.
//
// Cinderfish deletes immediately and hard (Firefish marks the account deleted
// and a job sweeps it later). Drive objects are removed first; the user row
// cascades the rest. The last administrator cannot be deleted, or the instance
// would have no way back in.
func (s *Server) handleIDeleteAccount(w http.ResponseWriter, r *http.Request) {
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
	if body.Password == "" {
		writeAPIError(w, invalidParamError("password is required"))
		return
	}
	if _, err := s.accounts.Authenticate(r.Context(), info.User.Username, body.Password); err != nil {
		writeAPIError(w, &APIError{
			Message: "Incorrect password.", Code: "INVALID_PASSWORD",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusForbidden,
		})
		return
	}

	if info.User.IsAdmin {
		var admins int
		if err := s.pool.QueryRow(r.Context(),
			`SELECT count(*) FROM users WHERE is_admin AND NOT is_suspended`).Scan(&admins); err != nil {
			writeInternalServerError(w)
			return
		}
		if admins <= 1 {
			writeAPIError(w, &APIError{
				Message: "The last administrator cannot be deleted.", Code: "LAST_ADMIN",
				ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
	}

	// Remove stored bytes first; deleting the user row only removes metadata.
	if s.drive != nil {
		cursor := ""
		for {
			files, err := s.drive.ListAll(r.Context(), media.AdminDriveParams{
				UserID: &info.User.ID, Limit: 100, UntilID: cursor,
			})
			if err != nil || len(files) == 0 {
				break
			}
			for _, f := range files {
				_ = s.drive.Delete(r.Context(), f.ID, info.User.ID)
			}
			if len(files) < 100 {
				break
			}
			cursor = files[len(files)-1].ID
		}
	}

	if _, err := s.pool.Exec(r.Context(), `DELETE FROM users WHERE id = $1`, info.User.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
