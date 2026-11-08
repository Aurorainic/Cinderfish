package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/captcha"
)

// verifySignupCaptcha checks the active captcha provider's token. It answers
// the request itself and reports false when verification fails. With no
// provider configured (or a provider without a secret, which Misskey treats
// as disabled) signup proceeds unchecked.
func (s *Server) verifySignupCaptcha(w http.ResponseWriter, r *http.Request, hcaptchaToken, recaptchaToken, turnstileToken string) bool {
	settings := s.liveInstanceSettings(r.Context())

	var provider captcha.Provider
	var secret, token string
	switch settings.CaptchaProvider {
	case "hcaptcha":
		provider, secret, token = captcha.ProviderHcaptcha, settings.CaptchaSecret, hcaptchaToken
	case "recaptcha":
		provider, secret, token = captcha.ProviderRecaptcha, settings.CaptchaSecret, recaptchaToken
	case "turnstile":
		provider, secret, token = captcha.ProviderTurnstile, settings.CaptchaSecret, turnstileToken
	default:
		return true
	}
	if strings.TrimSpace(secret) == "" {
		return true
	}

	if err := captcha.Verify(r.Context(), s.captchaHTTP, provider, secret, token, clientIP(r)); err != nil {
		writeAPIError(w, &APIError{
			Message: "Captcha verification failed.", Code: "CAPTCHA_FAILED",
			ID: errIDCaptchaFailed, Kind: "client", Status: http.StatusBadRequest,
		})
		return false
	}
	return true
}

// autoBanIfNeeded creates an IP ban once a source address has produced the
// configured number of failed signins inside the window. The current attempt
// still answers "invalid password"; the next one hits the ban.
func (s *Server) autoBanIfNeeded(ctx context.Context, ip string) {
	if s.accounts == nil || strings.TrimSpace(ip) == "" {
		return
	}
	cfg := s.liveInstanceSettings(ctx).SigninAutoBan
	if !cfg.Enabled {
		return
	}
	since := time.Now().Add(-time.Duration(cfg.WindowMinutes) * time.Minute)
	count, err := s.accounts.CountRecentSigninFailures(ctx, ip, since)
	if err != nil || count < cfg.Threshold {
		return
	}
	if banned, _ := s.accounts.IsIPBanned(ctx, ip); banned {
		return
	}
	until := time.Now().Add(time.Duration(cfg.BanMinutes) * time.Minute)
	_, _ = s.accounts.CreateIPBan(ctx, ip,
		fmt.Sprintf("自动封禁：%d 分钟内 %d 次登录失败", cfg.WindowMinutes, count), &until, "")
}

// recordSigninFailure counts one failed attempt and applies the automatic
// ban rule. Both unknown usernames and wrong passwords count: the counter
// table does not need an account row.
func (s *Server) recordSigninFailure(ctx context.Context, ip string) {
	if s.accounts == nil {
		return
	}
	_ = s.accounts.RecordSigninFailure(ctx, ip)
	s.autoBanIfNeeded(ctx, ip)
}
