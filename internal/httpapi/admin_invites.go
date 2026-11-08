package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Invitation code administration. The contract is Misskey's
// admin/invite/create, admin/invite/list and admin/invite/delete
// (packages/backend/src/server/api/endpoints/{admin/,}invite/): the packed
// shape mirrors Packed<'InviteCode'>, so misskey-js clients can read it
// unchanged. Deleting is admin-only here: upstream also lets the ticket's
// creator delete their own code, but Cinderfish only issues codes from the
// admin console, where every issuer is a moderator.

// packInvite renders one ticket in the InviteCode shape.
func packInvite(inv account.Invite) map[string]any {
	out := map[string]any{
		"id":        inv.ID,
		"code":      inv.Code,
		"createdAt": inv.CreatedAt.UTC().Format(time.RFC3339Nano),
		"expiresAt": nil,
		"usedAt":    nil,
		"used":      inv.Used,
		"createdBy": nil,
		"usedBy":    nil,
	}
	if inv.ExpiresAt != nil {
		out["expiresAt"] = inv.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if inv.UsedAt != nil {
		out["usedAt"] = inv.UsedAt.UTC().Format(time.RFC3339Nano)
	}
	if inv.CreatedBy != nil {
		out["createdBy"] = packUserLite(inv.CreatedBy, "", nil)
	}
	if inv.UsedBy != nil {
		out["usedBy"] = packUserLite(inv.UsedBy, "", nil)
	}
	return out
}

// handleAdminInviteCreate is POST /api/admin/invite/create.
func (s *Server) handleAdminInviteCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Count     int     `json:"count"`
		ExpiresAt *string `json:"expiresAt"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Count <= 0 {
		body.Count = 1
	}
	if body.Count > 100 {
		writeAPIError(w, invalidParamError("count must be at most 100"))
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		moment, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			writeAPIError(w, &APIError{
				Message: "Invalid date-time format", Code: "INVALID_DATE_TIME",
				ID: "f1380b15-3760-4c6c-a1db-5c3aaf1cbd49", Kind: "client",
				Status: http.StatusBadRequest,
			})
			return
		}
		expiresAt = &moment
	}

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	issuerID := ""
	if info != nil {
		issuerID = info.User.ID
	}

	out := make([]map[string]any, 0, body.Count)
	for range body.Count {
		inv, err := s.accounts.CreateInvite(r.Context(), issuerID, expiresAt)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		// The INSERT returns the bare ticket; attach the issuer so the
		// response carries createdBy like the list endpoint does.
		if issuerID != "" {
			if issuer, err := s.accounts.GetByID(r.Context(), issuerID); err == nil {
				inv.CreatedBy = issuer
			}
		}
		out = append(out, packInvite(inv))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminInviteList is POST /api/admin/invite/list.
func (s *Server) handleAdminInviteList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
		Type   string `json:"type"`
		Sort   string `json:"sort"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	switch body.Type {
	case "", "all", "unused", "used", "expired":
	default:
		writeAPIError(w, invalidParamError("type must be all, unused, used or expired"))
		return
	}
	switch body.Sort {
	case "", "+createdAt", "-createdAt", "+usedAt", "-usedAt":
	default:
		writeAPIError(w, invalidParamError("sort must be one of +createdAt -createdAt +usedAt -usedAt"))
		return
	}

	invites, err := s.accounts.ListInvites(r.Context(), account.InviteFilter{
		Limit: body.Limit, Offset: body.Offset, Type: body.Type, Sort: body.Sort,
	})
	if err != nil {
		writeAPIError(w, invalidParamError(err.Error()))
		return
	}
	out := make([]map[string]any, 0, len(invites))
	for _, inv := range invites {
		out = append(out, packInvite(inv))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminInviteDelete is POST /api/admin/invite/delete. Upstream registers
// the user-facing path invite/delete; administrators reach it from the same
// console, and codes here are always admin-issued, so one path serves both.
func (s *Server) handleAdminInviteDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		InviteID string `json:"inviteId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.InviteID == "" {
		writeAPIError(w, invalidParamError("inviteId is required"))
		return
	}

	if err := s.accounts.DeleteInvite(r.Context(), body.InviteID); err != nil {
		if errors.Is(err, account.ErrInviteNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such invite code.", Code: "NO_SUCH_INVITE_CODE",
				ID: "cd4f9ae4-7854-4e3e-8df9-c296f051e634", Kind: "client",
				Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}
