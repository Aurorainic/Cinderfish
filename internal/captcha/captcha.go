// Package captcha verifies hCaptcha, reCAPTCHA and Cloudflare Turnstile
// tokens for signup. The verification endpoints are fixed, allowlisted
// hosts; no caller-supplied URL is ever fetched, which keeps this path clear
// of SSRF concerns.
package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Provider names a supported captcha service.
type Provider string

const (
	ProviderHcaptcha  Provider = "hcaptcha"
	ProviderRecaptcha Provider = "recaptcha"
	ProviderTurnstile Provider = "turnstile"
)

// verifyURLs are the only endpoints this package contacts.
var verifyURLs = map[Provider]string{
	ProviderHcaptcha:  "https://hcaptcha.com/siteverify",
	ProviderRecaptcha: "https://www.recaptcha.net/recaptcha/api/siteverify",
	ProviderTurnstile: "https://challenges.cloudflare.com/turnstile/v0/siteverify",
}

// Error describes a failed verification. Code is stable for logging.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

// Verify checks a response token with the provider. An empty token fails
// with NO_RESPONSE without a network call. The client may be nil, in which
// case http.DefaultClient is used with no timeout (callers pass one).
func Verify(ctx context.Context, client *http.Client, provider Provider, secret, response, remoteIP string) error {
	if strings.TrimSpace(response) == "" {
		return &Error{Code: "NO_RESPONSE", Message: "captcha: no response token provided"}
	}
	endpoint, ok := verifyURLs[provider]
	if !ok {
		return &Error{Code: "INVALID_PROVIDER", Message: fmt.Sprintf("captcha: unknown provider %q", provider)}
	}
	if client == nil {
		client = http.DefaultClient
	}

	form := url.Values{"secret": {secret}, "response": {response}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return &Error{Code: "REQUEST_FAILED", Message: "captcha: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return &Error{Code: "REQUEST_FAILED", Message: "captcha: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &Error{Code: "REQUEST_FAILED", Message: fmt.Sprintf("captcha: provider answered %d", resp.StatusCode)}
	}

	var body struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return &Error{Code: "REQUEST_FAILED", Message: "captcha: " + err.Error()}
	}
	if !body.Success {
		return &Error{
			Code:    "VERIFICATION_FAILED",
			Message: "captcha: verification failed " + strings.Join(body.ErrorCodes, ", "),
		}
	}
	return nil
}
