package httpapi

import (
	"net/http"
)

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
	meta := map[string]any{
		"maintainerName":               settings.MaintainerName,
		"maintainerEmail":              settings.MaintainerEmail,
		"version":                      "0.1.0",
		"providesTarball":              false,
		"name":                         settings.Name,
		"shortName":                    "cinderfish",
		"uri":                          "https://" + s.cfg.Domain,
		"description":                  settings.Description,
		"langs":                        []string{"en"},
		"tosUrl":                       nilOrString(settings.TosURL),
		"repositoryUrl":                settings.RepositoryURL,
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
		"mascotImageUrl":               "/assets/ai.png",
		"bannerUrl":                    settings.BannerURL,
		"infoImageUrl":                 "",
		"serverErrorImageUrl":          "",
		"notFoundImageUrl":             "",
		"iconUrl":                      settings.IconURL,
		"backgroundImageUrl":           "",
		"logoImageUrl":                 "",
		"maxNoteTextLength":            3000,
		"defaultLightTheme":            nil,
		"defaultDarkTheme":             nil,
		"clientOptions":                map[string]any{},
		"ads":                          []any{},
		"notesPerOneAd":                0,
		"enableEmail":                  false,
		"enableServiceWorker":          false,
		"translatorAvailable":          false,
		"serverRules":                  settings.ServerRules,
		"policies":                     defaultPolicies(),
		"sentryForFrontend":            nil,
		"mediaProxy":                   "",
		"enableUrlPreview":             true,
		"noteSearchableScope":          "global",
		"maxFileSize":                  20 * 1024 * 1024,
		// Misskey exposes the instance's federation mode to clients.
		"federation": s.federationMode(),
	}

	if !detail {
		writeJSON(w, http.StatusOK, meta)
		return
	}

	meta["cacheRemoteFiles"] = true
	meta["cacheRemoteSensitiveFiles"] = false
	meta["requireSetup"] = false
	meta["proxyAccountName"] = ""
	meta["features"] = map[string]bool{
		"localTimeline":          true,
		"globalTimeline":         s.cfg.FederationMode == "public",
		"registration":           s.cfg.RegistrationMode == "open",
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

// federationMode maps the configured mode onto Misskey's own vocabulary
// (all | specified | none), so clients can read the instance's federation
// state the same way they read a real Misskey instance.
func (s *Server) federationMode() string {
	switch s.cfg.FederationMode {
	case "public":
		return "all"
	case "allowlist":
		return "specified"
	default:
		return "none"
	}
}

// defaultPolicies mirrors Misskey's DEFAULT_POLICIES shape for the fields
// clients commonly read.
func defaultPolicies() map[string]any {
	return map[string]any{
		"canUseFederation":     true,
		"canCreateInstantNote": true,
	}
}
