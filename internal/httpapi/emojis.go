package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// Misskey error ids used by the emoji endpoints.
const (
	errIDNoSuchFile  = "b6992544-63e7-67f0-fa7f-32444b1b5306"
	errIDEmojiExists = "fdb8db8e-3a2a-4f4f-8c2b-2f2b1e0c9d11"
	errIDNoSuchEmoji = "6a5fbb34-9d25-4b4f-8d9f-3c3a5b1c8f2e"
)

// emojiNamePattern is Misskey's admin/emoji/add name pattern.
var emojiNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// packEmojiSimple renders an emoji in Misskey's EmojiSimple shape, which is
// what the public emojis endpoint returns.
func packEmojiSimple(e *media.Emoji) map[string]any {
	return map[string]any{
		"name":        e.Name,
		"category":    e.Category,
		"aliases":     e.Aliases,
		"url":         e.URL,
		"localOnly":   e.LocalOnly,
		"isSensitive": e.IsSensitive,
	}
}

// packAdminEmoji renders an emoji in Misskey's Emoji shape for the admin
// console.
func packAdminEmoji(e *media.Emoji) map[string]any {
	return map[string]any{
		"id":          e.ID,
		"name":        e.Name,
		"host":        e.Host,
		"category":    e.Category,
		"aliases":     e.Aliases,
		"url":         e.URL,
		"isSensitive": e.IsSensitive,
		"localOnly":   e.LocalOnly,
		"createdAt":   e.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// handleEmojis implements POST /api/emojis: the local custom emoji registry,
// readable without a credential like upstream.
func (s *Server) handleEmojis(w http.ResponseWriter, r *http.Request) {
	if s.emojis == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	list, err := s.emojis.ListLocal(r.Context())
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(list))
	for _, e := range list {
		out = append(out, packEmojiSimple(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"emojis": out})
}

// handleAdminEmojiList implements POST /api/admin/emoji/list.
func (s *Server) handleAdminEmojiList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.emojis == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Query   *string `json:"query"`
		Limit   int     `json:"limit"`
		SinceID string  `json:"sinceId"`
		UntilID string  `json:"untilId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	q := media.EmojiQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.Query != nil {
		q.Query = *body.Query
	}

	list, err := s.emojis.ListAdmin(r.Context(), q)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(list))
	for _, e := range list {
		out = append(out, packAdminEmoji(e))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminEmojiAdd implements POST /api/admin/emoji/add: a drive file the
// admin owns becomes a custom emoji under the given name.
func (s *Server) handleAdminEmojiAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.emojis == nil || s.drive == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Name     string   `json:"name"`
		FileID   string   `json:"fileId"`
		Category *string  `json:"category"`
		Aliases  []string `json:"aliases"`
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
	if !emojiNamePattern.MatchString(body.Name) {
		writeAPIError(w, invalidParamError("name must match ^[a-zA-Z0-9_]+$"))
		return
	}
	for _, alias := range body.Aliases {
		if !emojiNamePattern.MatchString(alias) {
			writeAPIError(w, invalidParamError("aliases must match ^[a-zA-Z0-9_]+$"))
			return
		}
	}

	file, err := s.drive.Get(r.Context(), body.FileID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: errIDNoSuchFile, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	emoji, err := s.emojis.Add(r.Context(), body.Name, file.URL, body.Category, body.Aliases, false)
	if err != nil {
		if errors.Is(err, media.ErrEmojiExists) {
			writeAPIError(w, &APIError{
				Message: "That emoji already exists.", Code: "EMOJI_ALREADY_EXISTS",
				ID: errIDEmojiExists, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packAdminEmoji(emoji))
}

// handleAdminEmojiDelete implements POST /api/admin/emoji/delete.
func (s *Server) handleAdminEmojiDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.emojis == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ID == "" {
		writeAPIError(w, invalidParamError("id is required"))
		return
	}

	if err := s.emojis.Delete(r.Context(), body.ID); err != nil {
		if errors.Is(err, media.ErrEmojiNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such emoji.", Code: "NO_SUCH_EMOJI",
				ID: errIDNoSuchEmoji, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}
