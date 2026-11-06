package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// importSummary reports what i/import restored. Misskey answers an empty body;
// this port returns the numbers so the import page can show a result instead of
// guessing.
type importSummary struct {
	Notes             int `json:"notes"`
	Followings        int `json:"followings"`
	Favorites         int `json:"favorites"`
	Skipped           int `json:"skipped"`
	SkippedFollowings int `json:"skippedFollowings"`
	SkippedFavorites  int `json:"skippedFavorites"`
}

// handleIImport implements POST /api/i/import.
//
// The uploaded file must be a `cinderfish-account-export@1` document produced
// by i/export. Notes keep their original ids when free; followings and
// favorites are restored only when the target user or note is already known.
// Import is additive: nothing is deleted, and conflicts are skipped.
func (s *Server) handleIImport(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil || s.drive == nil || s.pool == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		FileID string `json:"fileId"`
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

	rc, f, err := s.drive.Open(r.Context(), body.FileID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	defer rc.Close()
	if f.UserID != info.User.ID {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	raw, err := io.ReadAll(io.LimitReader(rc, 32<<20))
	if err != nil {
		writeAPIError(w, invalidParamError("the file could not be read"))
		return
	}
	var doc exportAccount
	if err := json.Unmarshal(raw, &doc); err != nil {
		writeAPIError(w, invalidParamError("the file is not a JSON export"))
		return
	}
	if doc.Format != "cinderfish-account-export@1" {
		writeAPIError(w, invalidParamError("unsupported export format"))
		return
	}

	summary, err := s.importAccount(r.Context(), info.User.ID, &doc)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) importAccount(ctx context.Context, userID string, doc *exportAccount) (*importSummary, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	summary := &importSummary{}

	// Notes first: replies and favorites refer to them.
	ids := make([]string, 0, len(doc.Notes))
	for _, n := range doc.Notes {
		ids = append(ids, n.ID)
	}
	existing := map[string]struct{}{}
	if len(ids) > 0 {
		rows, err := tx.Query(ctx, `SELECT id FROM notes WHERE id = ANY($1)`, ids)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			existing[id] = struct{}{}
		}
		rows.Close()
	}

	for _, n := range doc.Notes {
		if n.ID == "" || len(n.Text) > note.MaxTextLength || !validImportVisibility(n.Visibility) {
			summary.Skipped++
			continue
		}
		if _, taken := existing[n.ID]; taken {
			summary.Skipped++
			continue
		}
		var replyID, renoteID *string
		if n.ReplyID != nil {
			if _, ok := existing[*n.ReplyID]; ok {
				replyID = n.ReplyID
			}
		}
		if n.RenoteID != nil {
			if _, ok := existing[*n.RenoteID]; ok {
				renoteID = n.RenoteID
			}
		}
		createdAt := parseImportTime(n.CreatedAt)
		text := n.Text
		var textPtr *string
		if text != "" {
			textPtr = &text
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO notes (id, user_id, visibility, cw, text, file_ids, reply_id, renote_id,
                   tags, mentions, emojis, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'[]'::jsonb,'[]'::jsonb,$10)
ON CONFLICT (id) DO NOTHING
`, n.ID, userID, n.Visibility, n.CW, textPtr, n.FileIDs, replyID, renoteID, n.Tags, createdAt); err != nil {
			return nil, err
		}
		existing[n.ID] = struct{}{}
		summary.Notes++
	}

	for _, entry := range doc.Followings {
		username, _ := entry["username"].(string)
		host, _ := entry["host"].(string)
		if username == "" {
			summary.SkippedFollowings++
			continue
		}
		var followeeID string
		query := `SELECT id FROM users WHERE username_lower = lower($1) AND host IS NULL`
		args := []any{username}
		if host != "" {
			query = `SELECT id FROM users WHERE username_lower = lower($1) AND host = $2`
			args = append(args, host)
		}
		if err := tx.QueryRow(ctx, query, args...).Scan(&followeeID); err != nil {
			summary.SkippedFollowings++
			continue
		}
		createdAt := time.Now()
		if raw, ok := entry["createdAt"].(string); ok {
			createdAt = parseImportTime(raw)
		}
		tag, err := tx.Exec(ctx, `
INSERT INTO followings (id, follower_id, followee_id, created_at)
VALUES ($1,$2,$3,$4) ON CONFLICT (follower_id, followee_id) DO NOTHING
`, idgen.New(), userID, followeeID, createdAt)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			summary.SkippedFollowings++
		} else {
			summary.Followings++
		}
	}

	for _, entry := range doc.Favorites {
		noteID, _ := entry["noteId"].(string)
		if noteID == "" {
			summary.SkippedFavorites++
			continue
		}
		createdAt := time.Now()
		if raw, ok := entry["createdAt"].(string); ok {
			createdAt = parseImportTime(raw)
		}
		tag, err := tx.Exec(ctx, `
INSERT INTO note_favorites (id, user_id, note_id, created_at)
SELECT $1,$2,$3,$4 WHERE EXISTS (SELECT 1 FROM notes WHERE id = $3)
ON CONFLICT (user_id, note_id) DO NOTHING
`, idgen.New(), userID, noteID, createdAt)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			summary.SkippedFavorites++
		} else {
			summary.Favorites++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return summary, nil
}

func validImportVisibility(v string) bool {
	switch v {
	case note.VisibilityPublic, note.VisibilityHome, note.VisibilityFollowers, note.VisibilitySpecified:
		return true
	default:
		return false
	}
}

func parseImportTime(value string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}
	return time.Now()
}
