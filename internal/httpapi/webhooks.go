package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/webhook"
)

const (
	maxWebhookName   = 100
	maxWebhookURL    = 1024
	maxWebhookSecret = 1024
)

// packWebhook renders the shape the Firefish settings pages read. The secret
// is included because the owner edits it; it never leaves the owner's requests.
func packWebhook(w *webhook.Webhook) map[string]any {
	var sentAt any
	if w.LatestSentAt != nil {
		sentAt = w.LatestSentAt.UTC().Format(time.RFC3339Nano)
	}
	var status any
	if w.LatestStatus != nil {
		status = *w.LatestStatus
	}
	return map[string]any{
		"id":           w.ID,
		"userId":       w.UserID,
		"name":         w.Name,
		"url":          w.URL,
		"secret":       w.Secret,
		"on":           w.On,
		"active":       w.Active,
		"createdAt":    w.CreatedAt.UTC().Format(time.RFC3339Nano),
		"latestSentAt": sentAt,
		"latestStatus": status,
	}
}

// normalizeWebhookFields validates the shared create/update body. It returns
// an APIError for the client or the cleaned values.
func normalizeWebhookFields(name, url, secret string, on []string) (string, string, string, []string, *APIError) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	secret = strings.TrimSpace(secret)
	if name == "" || utf8.RuneCountInString(name) > maxWebhookName {
		return "", "", "", nil, invalidParamError("name is required and at most 100 characters")
	}
	if len(url) > maxWebhookURL || (!strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://")) {
		return "", "", "", nil, invalidParamError("url must be an http(s) URL of at most 1024 characters")
	}
	if secret == "" || len(secret) > maxWebhookSecret {
		return "", "", "", nil, invalidParamError("secret is required and at most 1024 characters")
	}
	if len(on) == 0 {
		return "", "", "", nil, invalidParamError("on must list at least one event")
	}
	valid := make(map[string]bool, len(webhook.EventTypes))
	for _, t := range webhook.EventTypes {
		valid[t] = true
	}
	seen := make(map[string]bool, len(on))
	out := make([]string, 0, len(on))
	for _, t := range on {
		if !valid[t] {
			return "", "", "", nil, invalidParamError("unknown webhook event: " + t)
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return name, url, secret, out, nil
}

// handleWebhookCreate is POST /api/i/webhooks/create.
func (s *Server) handleWebhookCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Secret string   `json:"secret"`
		On     []string `json:"on"`
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
	if s.webhooks == nil {
		writeInternalServerError(w)
		return
	}
	name, url, secret, on, apiErr := normalizeWebhookFields(body.Name, body.URL, body.Secret, body.On)
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	created, err := s.webhooks.Create(r.Context(), info.User.ID, name, url, secret, on)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packWebhook(created))
}

// handleWebhookList is POST /api/i/webhooks/list.
func (s *Server) handleWebhookList(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if s.webhooks == nil {
		writeInternalServerError(w)
		return
	}
	hooks, err := s.webhooks.List(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]map[string]any, 0, len(hooks))
	for _, hook := range hooks {
		out = append(out, packWebhook(hook))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleWebhookShow is POST /api/i/webhooks/show.
func (s *Server) handleWebhookShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebhookID string `json:"webhookId"`
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
	if s.webhooks == nil {
		writeInternalServerError(w)
		return
	}
	hook, err := s.webhooks.Show(r.Context(), info.User.ID, body.WebhookID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such webhook.", Code: "NO_SUCH_WEBHOOK",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	writeJSON(w, http.StatusOK, packWebhook(hook))
}

// handleWebhookUpdate is POST /api/i/webhooks/update.
func (s *Server) handleWebhookUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebhookID string   `json:"webhookId"`
		Name      string   `json:"name"`
		URL       string   `json:"url"`
		Secret    string   `json:"secret"`
		On        []string `json:"on"`
		Active    *bool    `json:"active"`
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
	if s.webhooks == nil || body.WebhookID == "" {
		writeAPIError(w, invalidParamError("webhookId is required"))
		return
	}
	name, url, secret, on, apiErr := normalizeWebhookFields(body.Name, body.URL, body.Secret, body.On)
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}
	active := true
	if body.Active != nil {
		active = *body.Active
	}
	updated, err := s.webhooks.Update(r.Context(), info.User.ID, body.WebhookID, name, url, secret, on, active)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such webhook.", Code: "NO_SUCH_WEBHOOK",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	writeJSON(w, http.StatusOK, packWebhook(updated))
}

// handleWebhookDelete is POST /api/i/webhooks/delete.
func (s *Server) handleWebhookDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebhookID string `json:"webhookId"`
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
	if s.webhooks == nil {
		writeInternalServerError(w)
		return
	}
	if _, err := s.webhooks.Delete(r.Context(), info.User.ID, body.WebhookID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
