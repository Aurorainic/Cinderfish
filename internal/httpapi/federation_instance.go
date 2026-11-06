package httpapi

import (
	"context"
	"net/http"
	"strings"
)

// instanceInfo is the FederationInstance shape returned by
// federation/show-instance. Cinderfish has no `instances` table, so most
// software fields stay null; the counters come from the rows already stored.
type instanceInfo struct {
	Host              string  `json:"host"`
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	SoftwareName      *string `json:"softwareName"`
	SoftwareVersion   *string `json:"softwareVersion"`
	OpenRegistrations *bool   `json:"openRegistrations"`
	UsersCount        int     `json:"usersCount"`
	NotesCount        int     `json:"notesCount"`
	LatestStatus      *int    `json:"latestStatus"`
	IsBlocked         bool    `json:"isBlocked"`
	IsSuspended       bool    `json:"isSuspended"`
	IsNotResponding   bool    `json:"isNotResponding"`
}

// handleFederationShowInstance implements POST /api/federation/show-instance.
//
// The local host is answered from the runtime settings and live counts. Any
// other host is a row aggregated from the users/notes this instance already
// stores; no network request is made, so the endpoint cannot be used to probe
// arbitrary addresses. It returns null for a host this instance has never seen.
func (s *Server) handleFederationShowInstance(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil || s.accounts == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		Host string `json:"host"`
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
	if body.Host == "" {
		writeAPIError(w, invalidParamError("host is required"))
		return
	}

	host := normalizeHost(body.Host)
	if host == "" {
		writeAPIError(w, invalidParamError("host is invalid"))
		return
	}

	localHost := normalizeHost(s.cfg.Host)
	if localHost == "" {
		localHost = "localhost"
	}
	if host == localHost {
		settings := s.liveInstanceSettings(r.Context())
		usersCount, _ := s.accounts.CountUsers(r.Context())
		notesCount, _ := s.countLocalNotes(r.Context())
		name := settings.Name
		var description *string
		if settings.Description != "" {
			description = &settings.Description
		}
		software := "cinderfish"
		open := settings.RegistrationMode == "open"
		writeJSON(w, http.StatusOK, instanceInfo{
			Host:              host,
			Name:              &name,
			Description:       description,
			SoftwareName:      &software,
			SoftwareVersion:   nil,
			OpenRegistrations: &open,
			UsersCount:        usersCount,
			NotesCount:        notesCount,
		})
		return
	}

	var usersCount int
	if err := s.pool.QueryRow(r.Context(), `SELECT count(*) FROM users WHERE host = $1`, host).Scan(&usersCount); err != nil {
		writeInternalServerError(w)
		return
	}
	if usersCount == 0 {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	var notesCount int
	if err := s.pool.QueryRow(r.Context(), `
SELECT count(*) FROM notes WHERE user_host = $1 AND deleted_at IS NULL
`, host).Scan(&notesCount); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, instanceInfo{
		Host:       host,
		UsersCount: usersCount,
		NotesCount: notesCount,
	})
}

// countLocalNotes counts the instance's readable local notes.
func (s *Server) countLocalNotes(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM notes
WHERE user_host IS NULL AND deleted_at IS NULL
  AND visibility IN ('public','home')
`).Scan(&n)
	return n, err
}

// normalizeHost lowercases and trims a host, dropping a scheme or trailing
// slash so `https://例.example/` and `例.example` name the same instance.
func normalizeHost(value string) string {
	host := strings.ToLower(strings.TrimSpace(value))
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(host, prefix) {
			host = strings.TrimPrefix(host, prefix)
		}
	}
	host = strings.TrimSuffix(host, "/")
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if host == "" || strings.ContainsAny(host, " @\\") {
		return ""
	}
	return host
}

// handleFederationInstances implements POST /api/federation/instances: the
// remote hosts this instance already knows, aggregated from users/notes.
// Cinderfish keeps no `instances` table, so software/registration fields stay
// null; only the counters are meaningful.
func (s *Server) handleFederationInstances(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		Host   string `json:"host"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
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
	limit := body.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if body.Offset < 0 {
		body.Offset = 0
	}
	host := normalizeHost(body.Host)

	rows, err := s.pool.Query(r.Context(), `
SELECT u.host, count(DISTINCT u.id) AS users,
       (SELECT count(*) FROM notes n WHERE n.user_host = u.host AND n.deleted_at IS NULL) AS notes
FROM users u
WHERE u.host IS NOT NULL AND ($1 = '' OR u.host = $1)
GROUP BY u.host
ORDER BY users DESC, u.host ASC
LIMIT $2 OFFSET $3
`, host, limit, body.Offset)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()

	out := make([]instanceInfo, 0, limit)
	for rows.Next() {
		var item instanceInfo
		if err := rows.Scan(&item.Host, &item.UsersCount, &item.NotesCount); err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
