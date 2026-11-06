package httpapi

import (
	"net/http"
	"time"
)

// handleApps is POST /api/i/apps: the third-party applications the caller has
// authorized. Misskey returns a plain array with no pagination.
func (s *Server) handleApps(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Sort string `json:"sort"`
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
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}

	apps, err := s.tokens.Apps(r.Context(), info.User.ID, body.Sort)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		var lastUsed any
		if app.LastUsedAt != nil {
			lastUsed = app.LastUsedAt.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, map[string]any{
			"id":          app.ID,
			"name":        app.Name,
			"description": app.Description,
			"iconUrl":     nilIfEmpty(app.IconURL),
			"permission":  app.Scopes,
			"createdAt":   app.CreatedAt.UTC().Format(time.RFC3339Nano),
			"lastUsedAt":  lastUsed,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleRevokeToken is POST /api/i/revoke-token. Revoking a token that is not
// the caller's (or does not exist) is a silent no-op, like Misskey.
func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TokenID string `json:"tokenId"`
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
	if body.TokenID == "" {
		writeAPIError(w, invalidParamError("tokenId is required"))
		return
	}
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	if _, err := s.tokens.RevokeOwned(r.Context(), info.User.ID, body.TokenID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
