package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/announcement"
	"github.com/Aurorainic/cinderfish/internal/mail"
)

const (
	maxAnnouncementTitle = 256
	maxAnnouncementText  = 10000
	maxAnnouncementImage = 500
)

// packAnnouncement renders the shape Firefish/Misskey clients read. isRead is
// resolved per caller; anonymous requests always get false.
func packAnnouncement(a *announcement.Announcement, isRead bool) map[string]any {
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
		"isRead":    isRead,
	}
}

// applyUnreadAnnouncements fills hasUnreadAnnouncement from the read marks.
func (s *Server) applyUnreadAnnouncements(ctx context.Context, userID string, packed map[string]any) {
	if s.announcements == nil {
		return
	}
	count, err := s.announcements.UnreadCount(ctx, userID)
	if err != nil {
		return
	}
	packed["hasUnreadAnnouncement"] = count > 0
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

	// Read state is only meaningful for a signed-in caller; an expired or
	// invalid token falls back to the anonymous view instead of failing.
	read := map[string]bool{}
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		ids := make([]string, 0, len(rows))
		for _, a := range rows {
			ids = append(ids, a.ID)
		}
		if resolved, err := s.announcements.ReadIDs(r.Context(), info.User.ID, ids); err == nil {
			read = resolved
		}
	}

	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, packAnnouncement(a, read[a.ID]))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleIReadAnnouncement implements POST /api/i/read-announcement.
func (s *Server) handleIReadAnnouncement(w http.ResponseWriter, r *http.Request) {
	if s.announcements == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		AnnouncementID string `json:"announcementId"`
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
	if body.AnnouncementID == "" {
		writeAPIError(w, invalidParamError("announcementId is required"))
		return
	}
	if err := s.announcements.MarkRead(r.Context(), info.User.ID, body.AnnouncementID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	// Notify the subscribers in the background: the announcement is already
	// saved, and a slow SMTP server must not delay the admin's request.
	s.mailAnnouncement(title, text)

	writeJSON(w, http.StatusOK, packAnnouncement(created, false))
}

// mailAnnouncement sends one notice to every verified subscriber.
func (s *Server) mailAnnouncement(title, text string) {
	if s.mail == nil || s.accounts == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		recipients, err := s.accounts.AnnouncementMailRecipients(ctx)
		if err != nil {
			log.Printf("mail: announcement recipients: %v", err)
			return
		}
		body := title + "\n\n" + text + "\n\n你可以在账号设置里取消订阅实例公告邮件。"
		for _, to := range recipients {
			if err := s.mail.Send(ctx, mail.Message{To: to, Subject: title, Body: body}); err != nil {
				log.Printf("mail: announcement to %s: %v", to, err)
			}
		}
	}()
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
		out = append(out, packAnnouncement(a, false))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminAnnouncementsUpdate is POST /api/admin/announcements/update.
// Absent fields keep their current value; `imageUrl: null` and
// `expiresAt: null` clear them.
func (s *Server) handleAdminAnnouncementsUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.announcements == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		ID        string          `json:"id"`
		Title     *string         `json:"title"`
		Text      *string         `json:"text"`
		ImageURL  nullableField   `json:"imageUrl"`
		IsActive  *bool           `json:"isActive"`
		ExpiresAt json.RawMessage `json:"expiresAt"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ID == "" {
		writeAPIError(w, invalidParamError("id is required"))
		return
	}

	params := announcement.UpdateParams{IsActive: body.IsActive}
	if body.Title != nil {
		title := strings.TrimSpace(*body.Title)
		if title == "" || utf8.RuneCountInString(title) > maxAnnouncementTitle {
			writeAPIError(w, invalidParamError("title is required and at most 256 characters"))
			return
		}
		params.Title = &title
	}
	if body.Text != nil {
		text := strings.TrimSpace(*body.Text)
		if text == "" || utf8.RuneCountInString(text) > maxAnnouncementText {
			writeAPIError(w, invalidParamError("text is required and at most 10000 characters"))
			return
		}
		params.Text = &text
	}
	if body.ImageURL.provided() {
		value := ""
		if v := body.ImageURL.asUpdatePointer(); v != nil {
			value = strings.TrimSpace(*v)
		}
		if value != "" {
			if utf8.RuneCountInString(value) > maxAnnouncementImage {
				writeAPIError(w, invalidParamError("imageUrl is too long"))
				return
			}
			if !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "/") {
				writeAPIError(w, invalidParamError("imageUrl must be http(s) or an absolute path"))
				return
			}
		}
		params.ImageURL = &value
	}
	if len(body.ExpiresAt) > 0 {
		params.SetExpiry = true
		if string(body.ExpiresAt) != "null" {
			var millis int64
			if err := json.Unmarshal(body.ExpiresAt, &millis); err != nil {
				writeAPIError(w, invalidParamError("expiresAt must be a millisecond timestamp or null"))
				return
			}
			moment := time.UnixMilli(millis).UTC()
			params.ExpiresAt = &moment
		}
	}

	if _, err := s.announcements.Update(r.Context(), body.ID, params); err != nil {
		if errors.Is(err, announcement.ErrNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such announcement.", Code: "NO_SUCH_ANNOUNCEMENT",
				ID: errIDInternal, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
