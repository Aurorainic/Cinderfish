package httpapi

import (
	"net/http"
	"net/url"
	"time"
)

// handleAdminObjectStorage is POST /api/admin/object-storage: a read-only view
// of the media configuration. Secrets (access key, secret key) are never
// returned; the page tells the operator to edit config.toml instead.
func (s *Server) handleAdminObjectStorage(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	media := s.cfg.Media
	out := map[string]any{
		"backend": media.Backend,
	}
	if media.Backend == "s3" {
		endpointHost := media.S3.Endpoint
		if parsed, err := url.Parse(media.S3.Endpoint); err == nil && parsed.Host != "" {
			endpointHost = parsed.Host
		}
		out["localPath"] = nil
		out["s3"] = map[string]any{
			"endpointHost":   endpointHost,
			"bucket":         media.S3.Bucket,
			"region":         media.S3.Region,
			"useSSL":         media.S3.UseSSL,
			"publicBaseUrl":  nilIfEmpty(media.S3.PublicBaseURL),
			"hasCredentials": media.S3.AccessKey != "" && media.S3.SecretKey != "",
		}
	} else {
		out["localPath"] = media.LocalPath
		out["s3"] = nil
	}
	writeJSON(w, http.StatusOK, out)
}

// adminDeliverQueue is the shape of `admin/deliver-queue`.
type adminDeliverQueue struct {
	Pending      int64   `json:"pending"`
	DueNow       int64   `json:"dueNow"`
	Failed       int64   `json:"failed"`
	OldestNextAt *string `json:"oldestNextAt"`
	Recent       []any   `json:"recent"`
}

// handleAdminDeliverQueue is POST /api/admin/deliver-queue: the PostgreSQL
// delivery queue's health. `failed` counts rows that reached the retry cap
// (attempts >= 12), which the worker no longer picks up.
func (s *Server) handleAdminDeliverQueue(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	out := adminDeliverQueue{Recent: []any{}}
	if err := s.pool.QueryRow(r.Context(), `
SELECT
  count(*) FILTER (WHERE attempts < 12),
  count(*) FILTER (WHERE attempts < 12 AND next_attempt_at <= now()),
  count(*) FILTER (WHERE attempts >= 12)
FROM deliver_queue
`).Scan(&out.Pending, &out.DueNow, &out.Failed); err != nil {
		writeInternalServerError(w)
		return
	}

	var oldest *time.Time
	_ = s.pool.QueryRow(r.Context(), `
SELECT min(next_attempt_at) FROM deliver_queue WHERE attempts < 12
`).Scan(&oldest)
	if oldest != nil {
		formatted := oldest.UTC().Format(time.RFC3339Nano)
		out.OldestNextAt = &formatted
	}

	rows, err := s.pool.Query(r.Context(), `
SELECT id, inbox_url, attempts, next_attempt_at, COALESCE(last_error, '')
FROM deliver_queue ORDER BY id DESC LIMIT 20
`)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var inbox string
		var attempts int
		var nextAt time.Time
		var lastError string
		if err := rows.Scan(&id, &inbox, &attempts, &nextAt, &lastError); err != nil {
			writeInternalServerError(w)
			return
		}
		host := ""
		if parsed, err := url.Parse(inbox); err == nil {
			host = parsed.Host
		}
		out.Recent = append(out.Recent, map[string]any{
			"id":            id,
			"host":          host,
			"attempts":      attempts,
			"nextAttemptAt": nextAt.UTC().Format(time.RFC3339Nano),
			"lastError":     nilIfEmpty(lastError),
		})
	}
	if err := rows.Err(); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
