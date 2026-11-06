package httpapi

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// instanceSettings are the instance fields an administrator can change at
// runtime. Everything else still comes from config.toml; the admin page says
// so instead of pretending to own the whole configuration.
type instanceSettings struct {
	Name             string
	Description      string
	MaintainerName   string
	MaintainerEmail  string
	RegistrationMode string
}

// instanceMetaKey maps a settings field to its instance_meta key.
var instanceMetaKeys = map[string]string{
	"name":             "instanceName",
	"description":      "instanceDescription",
	"maintainerName":   "maintainerName",
	"maintainerEmail":  "maintainerEmail",
	"registrationMode": "registrationMode",
}

// liveInstanceSettings reads the runtime overrides, falling back to the
// configuration for every field without a row.
func (s *Server) liveInstanceSettings(ctx context.Context) instanceSettings {
	out := instanceSettings{
		Name:             "Cinderfish",
		Description:      "",
		MaintainerName:   "",
		MaintainerEmail:  "",
		RegistrationMode: s.cfg.RegistrationMode,
	}
	if out.RegistrationMode == "" {
		out.RegistrationMode = "admin"
	}
	if s.pool == nil {
		return out
	}
	rows, err := s.pool.Query(ctx, `
SELECT key, value #>> '{}' FROM instance_meta
WHERE key = ANY($1)
`, []string{
		instanceMetaKeys["name"], instanceMetaKeys["description"],
		instanceMetaKeys["maintainerName"], instanceMetaKeys["maintainerEmail"],
		instanceMetaKeys["registrationMode"],
	})
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case instanceMetaKeys["name"]:
			out.Name = value
		case instanceMetaKeys["description"]:
			out.Description = value
		case instanceMetaKeys["maintainerName"]:
			out.MaintainerName = value
		case instanceMetaKeys["maintainerEmail"]:
			out.MaintainerEmail = value
		case instanceMetaKeys["registrationMode"]:
			if value != "" {
				out.RegistrationMode = value
			}
		}
	}
	return out
}

// liveRegistrationMode resolves the mode the signup endpoint must enforce.
func (s *Server) liveRegistrationMode(ctx context.Context) account.RegistrationMode {
	switch s.liveInstanceSettings(ctx).RegistrationMode {
	case "open":
		return account.RegistrationOpen
	case "invite":
		return account.RegistrationInvite
	default:
		return account.RegistrationAdmin
	}
}

// saveInstanceSetting upserts one key.
func (s *Server) saveInstanceSetting(ctx context.Context, field, value string) error {
	key := instanceMetaKeys[field]
	_, err := s.pool.Exec(ctx, `
INSERT INTO instance_meta (key, value) VALUES ($1, to_jsonb($2::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, key, value)
	return err
}

// handleAdminMeta is POST /api/admin/meta. It returns only the subset this
// instance lets an administrator change.
func (s *Server) handleAdminMeta(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	settings := s.liveInstanceSettings(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"name":             settings.Name,
		"description":      settings.Description,
		"maintainerName":   settings.MaintainerName,
		"maintainerEmail":  settings.MaintainerEmail,
		"registrationMode": settings.RegistrationMode,
	})
}

// handleAdminUpdateMeta is POST /api/admin/update-meta.
//
// Misskey's endpoint accepts dozens of fields; this port accepts the five it
// actually stores and ignores the rest, which the admin page documents. A
// value of null or an absent key leaves the field unchanged.
func (s *Server) handleAdminUpdateMeta(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.pool == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Name             *string `json:"name"`
		Description      *string `json:"description"`
		MaintainerName   *string `json:"maintainerName"`
		MaintainerEmail  *string `json:"maintainerEmail"`
		RegistrationMode *string `json:"registrationMode"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	updates := map[string]string{}
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" || utf8.RuneCountInString(name) > 100 {
			writeAPIError(w, invalidParamError("name is required and at most 100 characters"))
			return
		}
		updates["name"] = name
	}
	if body.Description != nil {
		description := strings.TrimSpace(*body.Description)
		if utf8.RuneCountInString(description) > 500 {
			writeAPIError(w, invalidParamError("description is at most 500 characters"))
			return
		}
		updates["description"] = description
	}
	if body.MaintainerName != nil {
		value := strings.TrimSpace(*body.MaintainerName)
		if utf8.RuneCountInString(value) > 100 {
			writeAPIError(w, invalidParamError("maintainerName is at most 100 characters"))
			return
		}
		updates["maintainerName"] = value
	}
	if body.MaintainerEmail != nil {
		value := strings.TrimSpace(*body.MaintainerEmail)
		if value != "" && (!strings.Contains(value, "@") || utf8.RuneCountInString(value) > 254) {
			writeAPIError(w, invalidParamError("maintainerEmail must look like an address"))
			return
		}
		updates["maintainerEmail"] = value
	}
	if body.RegistrationMode != nil {
		value := *body.RegistrationMode
		switch value {
		case "open", "invite", "admin":
			updates["registrationMode"] = value
		default:
			writeAPIError(w, invalidParamError("registrationMode must be open, invite or admin"))
			return
		}
	}

	for field, value := range updates {
		if err := s.saveInstanceSetting(r.Context(), field, value); err != nil {
			writeInternalServerError(w)
			return
		}
	}

	s.handleAdminMeta(w, r)
}
