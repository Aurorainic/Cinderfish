package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/mail"
)

// mailSettingsMetaKey holds the runtime SMTP configuration. Like the
// object-storage override it supersedes config.toml once saved from the
// admin console; before that the [mail] section applies.
const mailSettingsMetaKey = "mailServer"

// storedMailSettings is the persisted shape. The password is stored in
// plaintext like any other instance secret; the API reports only whether it
// is set.
type storedMailSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLS      string `json:"tls"`
}

// liveMailSettings returns the effective SMTP settings and whether a runtime
// override row exists.
func (s *Server) liveMailSettings(ctx context.Context) (mail.Settings, bool) {
	fallback := mail.Settings{}
	if s.cfg != nil {
		fallback = mail.Settings{
			Host:     s.cfg.Mail.Host,
			Port:     s.cfg.Mail.Port,
			Username: s.cfg.Mail.Username,
			Password: s.cfg.Mail.Password,
			From:     s.cfg.Mail.From,
			TLS:      s.cfg.Mail.TLS,
		}
	}
	if s.pool == nil {
		return fallback, false
	}
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT value FROM instance_meta WHERE key = $1`, mailSettingsMetaKey).Scan(&raw); err != nil {
		return fallback, false
	}
	var stored storedMailSettings
	if json.Unmarshal(raw, &stored) != nil {
		return fallback, false
	}
	return mail.Settings{
		Host:     stored.Host,
		Port:     stored.Port,
		Username: stored.Username,
		Password: stored.Password,
		From:     stored.From,
		TLS:      stored.TLS,
	}, true
}

// switchMail points the switchable sender at new settings.
func (s *Server) switchMail(settings mail.Settings) {
	if sw, ok := s.mail.(*mail.SwitchableSender); ok {
		sw.Switch(mail.New(settings))
	}
}

// applyStoredMail activates a persisted SMTP override at startup.
func (s *Server) applyStoredMail() {
	if s.pool == nil || s.mail == nil {
		return
	}
	settings, stored := s.liveMailSettings(context.Background())
	if stored {
		s.switchMail(settings)
	}
}

// handleAdminEmailSettings is POST /api/admin/email-settings: the effective
// SMTP configuration with the password masked.
func (s *Server) handleAdminEmailSettings(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, s.packMailSettings(r.Context()))
}

// packMailSettings renders the effective SMTP settings for the admin page.
func (s *Server) packMailSettings(ctx context.Context) map[string]any {
	settings, stored := s.liveMailSettings(ctx)
	port := settings.Port
	if port == 0 {
		port = 587
	}
	tlsMode := settings.TLS
	if tlsMode == "" {
		tlsMode = "starttls"
	}
	return map[string]any{
		"host":        settings.Host,
		"port":        port,
		"username":    settings.Username,
		"passwordSet": settings.Password != "",
		"from":        nilIfEmpty(settings.From),
		"tls":         tlsMode,
		"configured":  strings.TrimSpace(settings.Host) != "",
		"stored":      stored,
	}
}

// handleAdminEmailSettingsUpdate is POST /api/admin/email-settings/update.
func (s *Server) handleAdminEmailSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		TLS      string `json:"tls"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	stored := storedMailSettings{
		Host:     strings.TrimSpace(body.Host),
		Port:     body.Port,
		Username: strings.TrimSpace(body.Username),
		Password: strings.TrimSpace(body.Password),
		From:     strings.TrimSpace(body.From),
		TLS:      strings.TrimSpace(body.TLS),
	}
	if stored.Port == 0 {
		stored.Port = 587
	}
	if stored.Port < 1 || stored.Port > 65535 {
		writeAPIError(w, invalidParamError("port must be between 1 and 65535"))
		return
	}
	switch stored.TLS {
	case "":
		stored.TLS = "starttls"
	case "starttls", "tls", "none":
	default:
		writeAPIError(w, invalidParamError("tls must be starttls, tls or none"))
		return
	}

	// An empty password keeps the stored one; the page never receives it.
	current, _ := s.liveMailSettings(r.Context())
	if stored.Password == "" {
		stored.Password = current.Password
	}

	payload, err := json.Marshal(stored)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
INSERT INTO instance_meta (key, value) VALUES ($1, $2::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, mailSettingsMetaKey, payload); err != nil {
		writeInternalServerError(w)
		return
	}
	s.switchMail(mail.Settings{
		Host:     stored.Host,
		Port:     stored.Port,
		Username: stored.Username,
		Password: stored.Password,
		From:     stored.From,
		TLS:      stored.TLS,
	})
	writeJSON(w, http.StatusOK, s.packMailSettings(r.Context()))
}
