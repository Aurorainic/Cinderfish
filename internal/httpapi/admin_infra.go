package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// objectStorageMetaKey holds the runtime media configuration, stored as one
// JSON object. It overrides [media] from config.toml once an administrator
// saves the object-storage page; before that the configuration applies.
const objectStorageMetaKey = "objectStorage"

// storedObjectStorage is the persisted shape. The secret is stored in
// plaintext like any other instance secret (the database is the trust
// boundary, as it is for the config file); the API never returns it.
type storedObjectStorage struct {
	Backend   string            `json:"backend"`
	LocalPath string            `json:"localPath"`
	S3        config.S3Settings `json:"s3"`
}

// liveMediaSettings returns the effective media configuration and whether a
// runtime override row exists.
func (s *Server) liveMediaSettings(ctx context.Context) (config.Media, bool) {
	if s.pool == nil {
		return s.cfg.Media, false
	}
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM instance_meta WHERE key = $1`, objectStorageMetaKey).Scan(&raw)
	if err != nil {
		return s.cfg.Media, false
	}
	var stored storedObjectStorage
	if json.Unmarshal(raw, &stored) != nil || stored.Backend == "" {
		return s.cfg.Media, false
	}
	return config.Media{Backend: stored.Backend, LocalPath: stored.LocalPath, S3: stored.S3}, true
}

// buildStorage constructs the backend for a media configuration.
func buildStorage(cfg config.Media) (media.Storage, error) {
	return media.BuildStorage(cfg.Backend, cfg.LocalPath, media.S3Options{
		Endpoint:      cfg.S3.Endpoint,
		Bucket:        cfg.S3.Bucket,
		Region:        cfg.S3.Region,
		AccessKey:     cfg.S3.AccessKey,
		SecretKey:     cfg.S3.SecretKey,
		UseSSL:        cfg.S3.UseSSL,
		PublicBaseURL: cfg.S3.PublicBaseURL,
		CACertPath:    cfg.S3.CACert,
	})
}

// switchStorage points the switchable backend at a new inner storage.
func (s *Server) switchStorage(inner media.Storage) {
	if sw, ok := s.storage.(*media.SwitchableStorage); ok {
		sw.Switch(inner)
	}
}

// applyStoredStorage activates a persisted object-storage override at
// startup. It does nothing when no override row exists, so a fresh instance
// keeps using config.toml.
func (s *Server) applyStoredStorage() {
	if s.pool == nil || s.storage == nil {
		return
	}
	ctx := context.Background()
	cfg, stored := s.liveMediaSettings(ctx)
	if !stored {
		return
	}
	inner, err := buildStorage(cfg)
	if err != nil {
		// Keep config.toml's backend rather than serving without storage.
		return
	}
	s.switchStorage(inner)
}

// handleAdminObjectStorage is POST /api/admin/object-storage: the effective
// media configuration (secrets masked) plus whether it comes from the
// runtime override or config.toml.
func (s *Server) handleAdminObjectStorage(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, s.packObjectStorage(r.Context()))
}

// packObjectStorage renders the effective media configuration for the admin
// console. Access and secret keys are reported as booleans only.
func (s *Server) packObjectStorage(ctx context.Context) map[string]any {
	mediaCfg, stored := s.liveMediaSettings(ctx)
	out := map[string]any{
		"backend":   mediaCfg.Backend,
		"localPath": nil,
		"s3":        nil,
		"stored":    stored,
	}
	if mediaCfg.Backend == "s3" {
		out["s3"] = map[string]any{
			"endpoint":       mediaCfg.S3.Endpoint,
			"bucket":         mediaCfg.S3.Bucket,
			"region":         mediaCfg.S3.Region,
			"useSSL":         mediaCfg.S3.UseSSL,
			"publicBaseUrl":  nilIfEmpty(mediaCfg.S3.PublicBaseURL),
			"caCert":         nilIfEmpty(mediaCfg.S3.CACert),
			"accessKeySet":   mediaCfg.S3.AccessKey != "",
			"secretKeySet":   mediaCfg.S3.SecretKey != "",
			"hasCredentials": mediaCfg.S3.AccessKey != "" && mediaCfg.S3.SecretKey != "",
		}
	} else {
		out["localPath"] = mediaCfg.LocalPath
	}
	return out
}

// handleAdminObjectStorageUpdate is POST /api/admin/object-storage/update.
// It validates the submitted configuration, builds the backend, persists the
// override and switches the live storage. Existing files are not migrated:
// the page says so, and cinderfish-media-migrate moves them.
func (s *Server) handleAdminObjectStorageUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Backend   string `json:"backend"`
		LocalPath string `json:"localPath"`
		S3        struct {
			Endpoint      string `json:"endpoint"`
			Bucket        string `json:"bucket"`
			Region        string `json:"region"`
			AccessKey     string `json:"accessKey"`
			SecretKey     string `json:"secretKey"`
			UseSSL        bool   `json:"useSSL"`
			PublicBaseURL string `json:"publicBaseUrl"`
			CACert        string `json:"caCert"`
		} `json:"s3"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	current, _ := s.liveMediaSettings(r.Context())
	stored := storedObjectStorage{
		Backend:   strings.TrimSpace(body.Backend),
		LocalPath: strings.TrimSpace(body.LocalPath),
	}
	if stored.Backend == "" {
		stored.Backend = "local"
	}

	switch stored.Backend {
	case "local":
		if stored.LocalPath == "" {
			writeAPIError(w, invalidParamError("localPath is required for the local backend"))
			return
		}
		// Keep the S3 settings so switching back does not lose them.
		stored.S3 = current.S3
	case "s3":
		stored.S3 = config.S3Settings{
			Endpoint:      strings.TrimSpace(body.S3.Endpoint),
			Bucket:        strings.TrimSpace(body.S3.Bucket),
			Region:        strings.TrimSpace(body.S3.Region),
			AccessKey:     strings.TrimSpace(body.S3.AccessKey),
			SecretKey:     strings.TrimSpace(body.S3.SecretKey),
			UseSSL:        body.S3.UseSSL,
			PublicBaseURL: strings.TrimSpace(body.S3.PublicBaseURL),
			CACert:        strings.TrimSpace(body.S3.CACert),
		}
		if stored.S3.Endpoint == "" || stored.S3.Bucket == "" {
			writeAPIError(w, invalidParamError("s3 endpoint and bucket are required"))
			return
		}
		// An empty secret keeps the stored one, so the form can be saved
		// without re-entering credentials it never received.
		if stored.S3.SecretKey == "" {
			stored.S3.SecretKey = current.S3.SecretKey
		}
		if stored.S3.AccessKey == "" {
			stored.S3.AccessKey = current.S3.AccessKey
		}
	default:
		writeAPIError(w, invalidParamError("backend must be local or s3"))
		return
	}

	mediaCfg := config.Media{Backend: stored.Backend, LocalPath: stored.LocalPath, S3: stored.S3}
	inner, err := buildStorage(mediaCfg)
	if err != nil {
		writeAPIError(w, invalidParamError(err.Error()))
		return
	}

	payload, err := json.Marshal(stored)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
INSERT INTO instance_meta (key, value) VALUES ($1, $2::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, objectStorageMetaKey, payload); err != nil {
		writeInternalServerError(w)
		return
	}
	s.switchStorage(inner)
	writeJSON(w, http.StatusOK, s.packObjectStorage(r.Context()))
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
