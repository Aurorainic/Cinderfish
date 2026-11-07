package httpapi

import (
	"context"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/buildinfo"
	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/media"
	"github.com/Aurorainic/cinderfish/internal/note"
)

const repositoryURL = "https://github.com/Aurorainic/Cinderfish"

// themeColorOrDefault keeps the historical default when no colour is set.
func themeColorOrDefault(value string) string {
	if value == "" {
		return "#86b300"
	}
	return value
}

// handleMeta answers POST /api/meta. It mirrors Misskey's
// MetaEntityService.pack / packDetailed: `detail` (default true) selects
// the detailed variant, which additionally carries `features`,
// `requireSetup`, `proxyAccountName` and the remote-file cache flags.
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Detail *bool `json:"detail"`
	}
	decodeOptionalJSON(r, &body)

	detail := true
	if body.Detail != nil {
		detail = *body.Detail
	}

	settings := s.liveInstanceSettings(r.Context())
	mode := s.liveFederation(r.Context())
	meta := map[string]any{
		"maintainerName":               settings.MaintainerName,
		"maintainerEmail":              settings.MaintainerEmail,
		"version":                      buildinfo.Version,
		"providesTarball":              false,
		"name":                         settings.Name,
		"shortName":                    "cinderfish",
		"uri":                          "https://" + s.cfg.Domain,
		"description":                  settings.Description,
		"langs":                        []string{"en"},
		"tosUrl":                       nilOrString(settings.TosURL),
		"repositoryUrl":                repositoryURLOrDefault(settings.RepositoryURL),
		"feedbackUrl":                  settings.FeedbackURL,
		"impressumUrl":                 settings.ImpressumURL,
		"privacyPolicyUrl":             settings.PrivacyPolicyURL,
		"inquiryUrl":                   settings.InquiryURL,
		"disableRegistration":          settings.RegistrationMode != "open",
		"emailRequiredForSignup":       false,
		"enableHcaptcha":               false,
		"hcaptchaSiteKey":              "",
		"enableMcaptcha":               false,
		"mcaptchaSiteKey":              "",
		"mcaptchaInstanceUrl":          "",
		"enableRecaptcha":              false,
		"recaptchaSiteKey":             "",
		"enableTurnstile":              false,
		"turnstileSiteKey":             "",
		"enableTestcaptcha":            false,
		"googleAnalyticsMeasurementId": "",
		"swPublickey":                  "",
		"themeColor":                   themeColorOrDefault(settings.ThemeColor),
		"mascotImageUrl":               "",
		"bannerUrl":                    settings.BannerURL,
		"infoImageUrl":                 "",
		"serverErrorImageUrl":          "",
		"notFoundImageUrl":             "",
		"iconUrl":                      settings.IconURL,
		"backgroundImageUrl":           "",
		"logoImageUrl":                 "",
		"maxNoteTextLength":            note.MaxTextLength,
		"defaultLightTheme":            nil,
		"defaultDarkTheme":             nil,
		"clientOptions":                map[string]any{},
		"ads":                          []any{},
		"notesPerOneAd":                0,
		"enableEmail":                  s.cfg.Mail.Enabled(),
		"enableServiceWorker":          false,
		"translatorAvailable":          false,
		"serverRules":                  settings.ServerRules,
		"customCss":                    settings.CustomCSS,
		"policies":                     defaultPolicies(),
		"sentryForFrontend":            nil,
		"mediaProxy":                   "",
		// No URL preview endpoint exists; claiming one would make clients
		// request an API this instance does not answer.
		"enableUrlPreview":    false,
		"noteSearchableScope": "global",
		"maxFileSize":         media.DefaultMaxFileSize,
		// Misskey exposes the instance's federation mode to clients.
		"federation": mode.MisskeyMode(),
	}

	if !detail {
		writeJSON(w, http.StatusOK, meta)
		return
	}

	// The admin console shows whether a mail server is configured; the host is
	// public knowledge (it appears in the Received headers of every message),
	// the credentials are not and never leave config.toml.
	meta["emailServer"] = s.cfg.Mail.Host
	meta["cacheRemoteFiles"] = true
	meta["cacheRemoteSensitiveFiles"] = false
	meta["requireSetup"] = false
	meta["proxyAccountName"] = ""
	meta["features"] = map[string]bool{
		"localTimeline":  true,
		"globalTimeline": mode != federation.ModeNone,
		// Registration mode is runtime state (admin/update-meta writes it), so
		// this must read the same value as disableRegistration above.
		"registration":           settings.RegistrationMode == "open",
		"emailRequiredForSignup": false,
		"hcaptcha":               false,
		"recaptcha":              false,
		"turnstile":              false,
		"objectStorage":          s.cfg.Media.Backend == "s3",
		"serviceWorker":          false,
		"miauth":                 true,
	}
	writeJSON(w, http.StatusOK, meta)
}

func repositoryURLOrDefault(value string) string {
	if value != "" {
		return value
	}
	return repositoryURL
}

// liveFederation reads the runtime federation switch. The mode lives in
// instance_meta while the instance runs; configuration only seeds that row,
// and an administrator can change it without a restart, so reading cfg here
// would report the startup value after a switch.
func (s *Server) liveFederation(ctx context.Context) federation.Mode {
	mode, err := federation.ParseMode(s.liveFederationMode(ctx))
	if err != nil {
		return federation.ModeNone
	}
	return mode
}

// defaultPolicies mirrors Misskey's DEFAULT_POLICIES shape for the fields
// clients commonly read.
func defaultPolicies() map[string]any {
	return map[string]any{
		"canUseFederation":     true,
		"canCreateInstantNote": true,
	}
}
