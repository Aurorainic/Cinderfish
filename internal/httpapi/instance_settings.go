package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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
	BlockedHosts     []string
	TosURL           string
	PrivacyPolicyURL string
	RepositoryURL    string
	FeedbackURL      string
	InquiryURL       string
	ImpressumURL     string
	ServerRules      []string
	HiddenTags       []string
	IconURL          string
	BannerURL        string
	ThemeColor       string
	// CustomCSS is instance-wide stylesheet text injected into every page.
	CustomCSS string
}

// instanceMetaKey maps a settings field to its instance_meta key.
var instanceMetaKeys = map[string]string{
	"name":             "instanceName",
	"description":      "instanceDescription",
	"maintainerName":   "maintainerName",
	"maintainerEmail":  "maintainerEmail",
	"registrationMode": "registrationMode",
	"tosUrl":           "tosUrl",
	"privacyPolicyUrl": "privacyPolicyUrl",
	"repositoryUrl":    "repositoryUrl",
	"feedbackUrl":      "feedbackUrl",
	"inquiryUrl":       "inquiryUrl",
	"impressumUrl":     "impressumUrl",
	"serverRules":      "serverRules",
	"hiddenTags":       "hiddenTags",
	"iconUrl":          "iconUrl",
	"bannerUrl":        "bannerUrl",
	"themeColor":       "themeColor",
	"customCss":        "customCss",
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
SELECT key, value FROM instance_meta WHERE key = ANY($1)
`, []string{
		instanceMetaKeys["name"], instanceMetaKeys["description"],
		instanceMetaKeys["maintainerName"], instanceMetaKeys["maintainerEmail"],
		instanceMetaKeys["registrationMode"],
		instanceMetaKeys["tosUrl"], instanceMetaKeys["privacyPolicyUrl"],
		instanceMetaKeys["repositoryUrl"], instanceMetaKeys["feedbackUrl"],
		instanceMetaKeys["inquiryUrl"], instanceMetaKeys["impressumUrl"],
		instanceMetaKeys["serverRules"],
		instanceMetaKeys["hiddenTags"],
		instanceMetaKeys["iconUrl"], instanceMetaKeys["bannerUrl"],
		instanceMetaKeys["themeColor"],
		instanceMetaKeys["customCss"],
	})
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			continue
		}
		if key == instanceMetaKeys["serverRules"] {
			_ = json.Unmarshal(raw, &out.ServerRules)
			continue
		}
		if key == instanceMetaKeys["hiddenTags"] {
			_ = json.Unmarshal(raw, &out.HiddenTags)
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
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
		case instanceMetaKeys["tosUrl"]:
			out.TosURL = value
		case instanceMetaKeys["privacyPolicyUrl"]:
			out.PrivacyPolicyURL = value
		case instanceMetaKeys["repositoryUrl"]:
			out.RepositoryURL = value
		case instanceMetaKeys["feedbackUrl"]:
			out.FeedbackURL = value
		case instanceMetaKeys["inquiryUrl"]:
			out.InquiryURL = value
		case instanceMetaKeys["impressumUrl"]:
			out.ImpressumURL = value
		case instanceMetaKeys["iconUrl"]:
			out.IconURL = value
		case instanceMetaKeys["bannerUrl"]:
			out.BannerURL = value
		case instanceMetaKeys["themeColor"]:
			out.ThemeColor = value
		case instanceMetaKeys["customCss"]:
			out.CustomCSS = value
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

// maxInstanceCustomCSS bounds the instance stylesheet. 100 KiB is far more
// than any sane theme override and keeps one admin from bloating every page
// response.
const maxInstanceCustomCSS = 100 * 1024

// nilOrString maps an empty settings string to JSON null, which is what
// Misskey's nullable URL fields expect.
func nilOrString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// liveBlockedHosts reads the blocked-host list. It prefers the policy cache
// (which the inbox and the deliverer use) and falls back to the table when no
// federation policy is attached, as in some tests.
func (s *Server) liveBlockedHosts(ctx context.Context) []string {
	if s.federationPolicy != nil {
		return s.federationPolicy.BlockedHosts(ctx)
	}
	if s.pool == nil {
		return []string{}
	}
	var raw []byte
	if err := s.pool.QueryRow(ctx,
		`SELECT value FROM instance_meta WHERE key = 'blockedHosts'`).Scan(&raw); err != nil {
		return []string{}
	}
	var hosts []string
	if err := json.Unmarshal(raw, &hosts); err != nil {
		return []string{}
	}
	return hosts
}

// saveBlockedHosts writes the list through the policy when it exists, so the
// cache is invalidated immediately.
func (s *Server) saveBlockedHosts(ctx context.Context, hosts []string) error {
	clean := make([]string, 0, len(hosts))
	seen := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		h := strings.ToLower(strings.TrimSpace(host))
		if h == "" || seen[h] {
			continue
		}
		if len(h) > 253 || strings.ContainsAny(h, "/ \\@") {
			return fmt.Errorf("blocked host is not a hostname: %s", host)
		}
		seen[h] = true
		clean = append(clean, h)
	}
	if len(clean) > 200 {
		return fmt.Errorf("blockedHosts has too many entries")
	}
	sort.Strings(clean)
	if s.federationPolicy != nil {
		return s.federationPolicy.SetBlockedHosts(ctx, clean)
	}
	payload, err := json.Marshal(clean)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO instance_meta (key, value) VALUES ('blockedHosts', $1::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, payload)
	return err
}

// handleAdminMeta is POST /api/admin/meta. It returns only the subset this
// instance lets an administrator change.
func (s *Server) handleAdminMeta(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	settings := s.liveInstanceSettings(r.Context())
	blocked := s.liveBlockedHosts(r.Context())
	if blocked == nil {
		blocked = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":             settings.Name,
		"description":      settings.Description,
		"maintainerName":   settings.MaintainerName,
		"maintainerEmail":  settings.MaintainerEmail,
		"registrationMode": settings.RegistrationMode,
		"blockedHosts":     blocked,
		"tosUrl":           nilOrString(settings.TosURL),
		"privacyPolicyUrl": settings.PrivacyPolicyURL,
		"repositoryUrl":    settings.RepositoryURL,
		"feedbackUrl":      settings.FeedbackURL,
		"inquiryUrl":       settings.InquiryURL,
		"impressumUrl":     settings.ImpressumURL,
		"serverRules":      settings.ServerRules,
		"hiddenTags":       settings.HiddenTags,
		"iconUrl":          nilIfEmpty(settings.IconURL),
		"bannerUrl":        nilIfEmpty(settings.BannerURL),
		"themeColor":       nilIfEmpty(settings.ThemeColor),
		"customCss":        settings.CustomCSS,
		// The admin console only needs to know whether mail is configured and
		// which host it goes through; credentials stay in config.toml.
		"emailServer": s.cfg.Mail.Host,
	})
}

// handleAdminUpdateMeta is POST /api/admin/update-meta.
//
// Misskey's endpoint accepts dozens of fields; this port accepts the ones it
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
		Name             *string   `json:"name"`
		Description      *string   `json:"description"`
		MaintainerName   *string   `json:"maintainerName"`
		MaintainerEmail  *string   `json:"maintainerEmail"`
		RegistrationMode *string   `json:"registrationMode"`
		BlockedHosts     *[]string `json:"blockedHosts"`
		TosURL           *string   `json:"tosUrl"`
		PrivacyPolicyURL *string   `json:"privacyPolicyUrl"`
		RepositoryURL    *string   `json:"repositoryUrl"`
		FeedbackURL      *string   `json:"feedbackUrl"`
		InquiryURL       *string   `json:"inquiryUrl"`
		ImpressumURL     *string   `json:"impressumUrl"`
		ServerRules      *[]string `json:"serverRules"`
		HiddenTags       *[]string `json:"hiddenTags"`
		// The three branding fields accept an explicit null to clear them, the
		// way Misskey's clients send it; nullableField tells null from absent.
		IconURL    nullableField `json:"iconUrl"`
		BannerURL  nullableField `json:"bannerUrl"`
		ThemeColor nullableField `json:"themeColor"`
		CustomCSS  *string       `json:"customCss"`
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

	urlFields := []struct {
		field string
		value *string
	}{
		{"tosUrl", body.TosURL},
		{"privacyPolicyUrl", body.PrivacyPolicyURL},
		{"repositoryUrl", body.RepositoryURL},
		{"feedbackUrl", body.FeedbackURL},
		{"inquiryUrl", body.InquiryURL},
		{"impressumUrl", body.ImpressumURL},
	}
	for _, item := range urlFields {
		if item.value == nil {
			continue
		}
		value := strings.TrimSpace(*item.value)
		if value != "" {
			if len(value) > 300 || (!strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "http://")) {
				writeAPIError(w, invalidParamError(item.field+" must be an http(s) URL of at most 300 characters"))
				return
			}
		}
		updates[item.field] = value
	}
	if body.ServerRules != nil {
		rules := make([]string, 0, len(*body.ServerRules))
		for _, rule := range *body.ServerRules {
			trimmed := strings.TrimSpace(rule)
			if trimmed == "" {
				continue
			}
			if utf8.RuneCountInString(trimmed) > 300 {
				writeAPIError(w, invalidParamError("each server rule is at most 300 characters"))
				return
			}
			rules = append(rules, trimmed)
		}
		if len(rules) > 20 {
			writeAPIError(w, invalidParamError("serverRules has at most 20 entries"))
			return
		}
		payload, err := json.Marshal(rules)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		if _, err := s.pool.Exec(r.Context(), `
INSERT INTO instance_meta (key, value) VALUES ('serverRules', $1::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, payload); err != nil {
			writeInternalServerError(w)
			return
		}
	}

	for _, item := range []struct {
		field string
		value nullableField
		url   bool
	}{
		{"iconUrl", body.IconURL, true},
		{"bannerUrl", body.BannerURL, true},
	} {
		if !item.value.provided() {
			continue
		}
		value := ""
		if v := item.value.asUpdatePointer(); v != nil {
			value = strings.TrimSpace(*v)
		}
		if value != "" {
			if len(value) > 300 || (!strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "http://")) {
				writeAPIError(w, invalidParamError(item.field+" must be an http(s) URL of at most 300 characters"))
				return
			}
		}
		updates[item.field] = value
	}
	if body.ThemeColor.provided() {
		color := ""
		if v := body.ThemeColor.asUpdatePointer(); v != nil {
			color = strings.TrimSpace(*v)
		}
		if color != "" && !validThemeColor(color) {
			writeAPIError(w, invalidParamError("themeColor must be #rrggbb"))
			return
		}
		updates["themeColor"] = color
	}

	if body.CustomCSS != nil {
		css := *body.CustomCSS
		if len(css) > maxInstanceCustomCSS {
			writeAPIError(w, invalidParamError("customCss is at most 100 KiB"))
			return
		}
		updates["customCss"] = css
	}

	if body.HiddenTags != nil {
		tags := make([]string, 0, len(*body.HiddenTags))
		seen := map[string]struct{}{}
		for _, tag := range *body.HiddenTags {
			trimmed := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(tag), "#")))
			if trimmed == "" {
				continue
			}
			if utf8.RuneCountInString(trimmed) > 100 {
				writeAPIError(w, invalidParamError("each hidden tag is at most 100 characters"))
				return
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			tags = append(tags, trimmed)
		}
		if len(tags) > 200 {
			writeAPIError(w, invalidParamError("hiddenTags has at most 200 entries"))
			return
		}
		payload, err := json.Marshal(tags)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		if _, err := s.pool.Exec(r.Context(), `
INSERT INTO instance_meta (key, value) VALUES ('hiddenTags', $1::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, payload); err != nil {
			writeInternalServerError(w)
			return
		}
	}

	if body.BlockedHosts != nil {
		if err := s.saveBlockedHosts(r.Context(), *body.BlockedHosts); err != nil {
			writeAPIError(w, invalidParamError(err.Error()))
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

// validThemeColor accepts a six-digit hex colour.
func validThemeColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, r := range value[1:] {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
