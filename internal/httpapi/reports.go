package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/report"
)

const maxReportComment = 2048

// handleReportAbuse is POST /api/users/report-abuse. The report targets an
// account; reporting your own account is rejected because it cannot mean
// anything and would only create noise in the admin queue.
func (s *Server) handleReportAbuse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID  string `json:"userId"`
		Comment string `json:"comment"`
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
	if s.reports == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	comment := strings.TrimSpace(body.Comment)
	if body.UserID == "" || comment == "" || utf8.RuneCountInString(comment) > maxReportComment {
		writeAPIError(w, invalidParamError("userId and a comment of at most 2048 characters are required"))
		return
	}
	if body.UserID == info.User.ID {
		writeAPIError(w, invalidParamError("you cannot report yourself"))
		return
	}
	if _, err := s.accounts.GetByID(r.Context(), body.UserID); err != nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if _, err := s.reports.Create(r.Context(), info.User.ID, body.UserID, comment); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminAbuseReports is POST /api/admin/abuse-user-reports.
func (s *Server) handleAdminAbuseReports(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.reports == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Limit            int     `json:"limit"`
		SinceID          string  `json:"sinceId"`
		UntilID          string  `json:"untilId"`
		State            *string `json:"state"`
		ReporterOrigin   string  `json:"reporterOrigin"`
		TargetUserOrigin string  `json:"targetUserOrigin"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	state := ""
	if body.State != nil {
		state = *body.State
	}
	if state != "" && state != "resolved" && state != "unresolved" {
		writeAPIError(w, invalidParamError("state must be resolved or unresolved"))
		return
	}

	rows, err := s.reports.List(r.Context(), report.ListParams{
		Limit:            body.Limit,
		SinceID:          body.SinceID,
		UntilID:          body.UntilID,
		State:            state,
		ReporterOrigin:   body.ReporterOrigin,
		TargetUserOrigin: body.TargetUserOrigin,
	})
	if err != nil {
		writeAPIError(w, invalidParamError(err.Error()))
		return
	}

	ids := make([]string, 0, len(rows)*2)
	for _, item := range rows {
		ids = append(ids, item.ReporterID, item.TargetUserID)
	}
	users, err := s.accounts.UsersByIDs(r.Context(), ids)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		packUser := func(id string) any {
			u, ok := users[id]
			if !ok {
				return nil
			}
			return packUserDetailed(u, u.CreatedAt, false, loadProfile(r.Context(), s, u.ID))
		}
		out = append(out, map[string]any{
			"id":         item.ID,
			"targetUser": packUser(item.TargetUserID),
			"reporter":   packUser(item.ReporterID),
			"comment":    item.Comment,
			"resolved":   item.Resolved,
			"createdAt":  item.CreatedAt.UTC().Format(time.RFC3339Nano),
			// Forwarding to a remote instance is not implemented; the field is
			// part of the client shape, so it stays false rather than absent.
			"forwarded": false,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminResolveAbuseReport is POST /api/admin/resolve-abuse-user-report.
// Resolving an unknown report is a no-op, matching Misskey's lack of an error
// for this endpoint.
func (s *Server) handleAdminResolveAbuseReport(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.reports == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		ReportID string `json:"reportId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ReportID == "" {
		writeAPIError(w, invalidParamError("reportId is required"))
		return
	}
	if _, err := s.reports.Resolve(r.Context(), body.ReportID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
