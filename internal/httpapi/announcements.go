package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/announcement"
)

const (
	maxAnnouncementTitle = 256
	maxAnnouncementText  = 10000
	maxAnnouncementImage = 500
)

// packAnnouncement renders the shape Firefish/Misskey clients read. isRead is
// always false: this port does not track per-user read state yet, and claiming
// "read" would hide the notice.
func packAnnouncement(a *announcement.Announcement) map[string]any {
	var imageURL any
	if a.ImageURL != nil {
		imageURL = *a.ImageURL
	}
	var expires any
	if a.ExpiresAt != nil {
		expires = a.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id":        a.ID,
		"title":     a.Title,
		"text":      a.Text,
		"imageUrl":  imageURL,
		"isActive":  a.IsActive,
		"createdAt": a.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": a.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"expiresAt": expires,
		"isRead":    false,
	}
}

// handleAnnouncements is POST /api/announcements: the public, active notices.
func (s *Server) handleAnnouncements(w http.ResponseWriter, r *http.Request) {
	if s.announcements == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Limit   int    `json:"limit"`
		UntilID string `json:"untilId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	rows, err := s.announcements.List(r.Context(), true, body.Limit, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, packAnnouncement(a))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminAnnouncementsCreate is POST /api/admin/announcements/create.
func (s *Server) handleAdminAnnouncementsCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.announcements == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Title     string  `json:"title"`
		Text      string  `json:"text"`
		ImageURL  *string `json:"imageUrl"`
		ExpiresAt *int64  `json:"expiresAt"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	title := strings.TrimSpace(body.Title)
	text := strings.TrimSpace(body.Text)
	if title == "" || utf8.RuneCountInString(title) > maxAnnouncementTitle {
		writeAPIError(w, invalidParamError("title is required and at most 256 characters"))
		return
	}
	if text == "" || utf8.RuneCountInString(text) > maxAnnouncementText {
		writeAPIError(w, invalidParamError("text is required and at most 10000 characters"))
		return
	}

	var imageURL *string
	if body.ImageURL != nil {
		value := strings.TrimSpace(*body.ImageURL)
		if value != "" {
			if utf8.RuneCountInString(value) > maxAnnouncementImage {
				writeAPIError(w, invalidParamError("imageUrl is too long"))
				return
			}
			if !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "/") {
				writeAPIError(w, invalidParamError("imageUrl must be http(s) or an absolute path"))
				return
			}
			imageURL = &value
		}
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		moment := time.UnixMilli(*body.ExpiresAt).UTC()
		if !moment.After(time.Now()) {
			writeAPIError(w, invalidParamError("expiresAt must be in the future"))
			return
		}
		expiresAt = &moment
	}

	created, err := s.announcements.Create(r.Context(), title, text, imageURL, expiresAt)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packAnnouncement(created))
}

// handleAdminAnnouncementsList is POST /api/admin/announcements/list.
func (s *Server) handleAdminAnnouncementsList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.announcements == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Limit   int    `json:"limit"`
		UntilID string `json:"untilId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	rows, err := s.announcements.List(r.Context(), false, body.Limit, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, packAnnouncement(a))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminAnnouncementsDelete is POST /api/admin/announcements/delete.
func (s *Server) handleAdminAnnouncementsDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.announcements == nil {
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
	if _, err := s.announcements.Delete(r.Context(), body.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
