package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// IP ban and signin-audit administration. These are Cinderfish extensions:
// neither Misskey nor Firefish exposes IP bans through the API, so the
// endpoint shapes are ours (documented in docs/HANDOVER.md §12GV). Bans are
// enforced on signin and signup only; they do not affect browsing, media or
// federation, where blocking belongs to the federation/host policy.

// packIPBan renders one ban rule.
func packIPBan(ban account.IPBan) map[string]any {
	out := map[string]any{
		"id":        ban.ID,
		"ip":        ban.IP,
		"reason":    ban.Reason,
		"createdAt": ban.CreatedAt.UTC().Format(time.RFC3339Nano),
		"expiresAt": nil,
	}
	if ban.ExpiresAt != nil {
		out["expiresAt"] = ban.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	if ban.CreatedByID != "" {
		out["createdBy"] = map[string]any{"id": ban.CreatedByID, "username": ban.CreatedBy}
	} else {
		out["createdBy"] = nil
	}
	return out
}

// handleAdminIPBans implements POST /api/admin/ip-bans/list.
func (s *Server) handleAdminIPBans(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	bans, err := s.accounts.ListIPBans(r.Context(), body.Limit, body.Offset)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]map[string]any, 0, len(bans))
	for _, ban := range bans {
		out = append(out, packIPBan(ban))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminIPBansCreate implements POST /api/admin/ip-bans/create.
func (s *Server) handleAdminIPBansCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		IP        string  `json:"ip"`
		Reason    string  `json:"reason"`
		ExpiresAt *string `json:"expiresAt"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		moment, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			writeAPIError(w, invalidParamError("expiresAt must be an RFC3339 date-time"))
			return
		}
		expiresAt = &moment
	}

	info, _ := s.authenticate(r.Context(), r, bodyAsMap(r))
	creator := ""
	if info != nil {
		creator = info.User.ID
	}
	ban, err := s.accounts.CreateIPBan(r.Context(), body.IP, body.Reason, expiresAt, creator)
	if err != nil {
		writeAPIError(w, invalidParamError(err.Error()))
		return
	}
	// The INSERT returns the bare row; attach the issuer like the list does.
	if creator != "" {
		if user, err := s.accounts.GetByID(r.Context(), creator); err == nil {
			ban.CreatedBy = user.Username
		}
	}
	writeJSON(w, http.StatusOK, packIPBan(*ban))
}

// handleAdminIPBansDelete implements POST /api/admin/ip-bans/delete.
func (s *Server) handleAdminIPBansDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
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
	if err := s.accounts.DeleteIPBan(r.Context(), body.ID); err != nil {
		if errors.Is(err, account.ErrIPBanNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such IP ban.", Code: "NO_SUCH_IP_BAN",
				ID: errIDNotFound, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleAdminSigninHistory implements POST /api/admin/signin-history: the
// audit view across accounts (the per-account list is i/signin-history).
func (s *Server) handleAdminSigninHistory(w http.ResponseWriter, r *http.Request) {
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
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	records, err := s.accounts.AdminSigninHistory(r.Context(), body.UserID, body.Limit, body.Offset)
	if err != nil {
		writeAPIError(w, internalError("Could not read the signin history."))
		return
	}
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, map[string]any{
			"id":        rec.ID,
			"createdAt": rec.CreatedAt.UTC().Format(time.RFC3339Nano),
			"ip":        rec.IP,
			"success":   rec.Success,
			"user":      map[string]any{"id": rec.UserID, "username": rec.Username},
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// rejectBannedIP answers 403 for a banned source address. It reports false
// (and writes nothing) when the address is allowed.
func (s *Server) rejectBannedIP(w http.ResponseWriter, r *http.Request) bool {
	if s.accounts == nil {
		return false
	}
	banned, err := s.accounts.IsIPBanned(r.Context(), clientIP(r))
	if err != nil || !banned {
		return false
	}
	writeAPIError(w, &APIError{
		Message: "Your IP address is banned from this instance.",
		Code:    "IP_BANNED", ID: errIDIPBanned, Kind: "permission",
		Status: http.StatusForbidden,
	})
	return true
}
