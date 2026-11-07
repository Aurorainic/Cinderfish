package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/federation"
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
			"forwarded":  item.Forwarded,
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

// handleAdminForwardAbuseReport is POST /api/admin/forward-abuse-user-report.
//
// Misskey forwards the complaint to the reported account's instance as an
// ActivityPub Flag, signed by the instance actor. Cinderfish has no instance
// actor, so the acting administrator's own account authors and signs the Flag;
// the receiving side only needs a verifiable actor.
func (s *Server) handleAdminForwardAbuseReport(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if !info.User.IsAdmin {
		writeAPIError(w, &APIError{
			Message: "You are not an administrator.", Code: "FORBIDDEN",
			ID: errIDForbidden, Kind: "permission", Status: http.StatusForbidden,
		})
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
	row, err := s.reports.Get(r.Context(), body.ReportID)
	if errors.Is(err, report.ErrNotFound) {
		writeAPIError(w, &APIError{
			Message: "No such abuse report.", Code: "NO_SUCH_ABUSE_REPORT",
			ID: "8763e21b-d9bc-40be-acf6-54c1a6986493", Kind: "server", Status: http.StatusNotFound,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if s.federationStore == nil || s.fedRenderer == nil || s.federationDeliver == nil {
		writeInternalServerError(w)
		return
	}

	target, err := s.federationStore.UserByID(r.Context(), row.TargetUserID)
	if err != nil || target.Host == "" {
		// Misskey fails with a 500 here; a 400 says what is wrong instead.
		writeAPIError(w, &APIError{
			Message: "Only reports about remote accounts can be forwarded.",
			Code:    "CANNOT_FORWARD", ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	// Claim the forward first: a second request (or a double click) must not
	// send the same Flag again.
	claimed, err := s.reports.MarkForwarded(r.Context(), row.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !claimed {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.deliverReportFlag(r.Context(), info.User.ID, row, target); err != nil {
		log.Printf("httpapi: forward report %s: %v", row.ID, err)
		// Put the report back so the administrator can retry.
		if _, revertErr := s.reports.UnmarkForwarded(r.Context(), row.ID); revertErr != nil {
			log.Printf("httpapi: forward report %s: revert forwarded flag: %v", row.ID, revertErr)
		}
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deliverReportFlag renders and queues the Flag for a report about a remote
// account. The remote actor's shared inbox wins over its personal inbox, which
// matches how every other activity is addressed.
func (s *Server) deliverReportFlag(ctx context.Context, adminID string, row *report.Report, target *federation.RemoteUser) error {
	inbox := target.SharedInbox
	if inbox == "" {
		inbox = target.Inbox
	}
	if inbox == "" {
		return errors.New("the remote account has no inbox")
	}
	actor := s.fedRenderer.ActorRef(adminID)
	flag := s.fedRenderer.RenderFlag(actor, target.URI, row.Comment)
	return s.federationDeliver.Enqueue(context.WithoutCancel(ctx), inbox, "", s.fedRenderer.KeyID(adminID), adminID, flag)
}
