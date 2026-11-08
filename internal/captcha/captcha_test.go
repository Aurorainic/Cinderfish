package captcha_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/captcha"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func respond(body string, status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestVerifySuccess(t *testing.T) {
	var seenURL, seenBody string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		seenURL = r.URL.String()
		raw, _ := io.ReadAll(r.Body)
		seenBody = string(raw)
		return respond(`{"success":true}`, 200), nil
	})}

	if err := captcha.Verify(context.Background(), client, captcha.ProviderHcaptcha, "sec", "tok", "192.0.2.1"); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !strings.HasPrefix(seenURL, "https://hcaptcha.com/siteverify") {
		t.Errorf("url = %q, want the fixed hcaptcha endpoint", seenURL)
	}
	for _, want := range []string{"secret=sec", "response=tok", "remoteip=192.0.2.1"} {
		if !strings.Contains(seenBody, want) {
			t.Errorf("form body %q missing %q", seenBody, want)
		}
	}
}

func TestVerifyFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return respond(`{"success":false,"error-codes":["invalid-input-response"]}`, 200), nil
	})}
	err := captcha.Verify(context.Background(), client, captcha.ProviderRecaptcha, "sec", "tok", "")
	var captchaErr *captcha.Error
	if err == nil || !asCaptchaError(err, &captchaErr) || captchaErr.Code != "VERIFICATION_FAILED" {
		t.Fatalf("error = %v, want VERIFICATION_FAILED", err)
	}
}

func TestVerifyNoResponse(t *testing.T) {
	called := false
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		return respond(`{"success":true}`, 200), nil
	})}
	err := captcha.Verify(context.Background(), client, captcha.ProviderTurnstile, "sec", "", "")
	var captchaErr *captcha.Error
	if err == nil || !asCaptchaError(err, &captchaErr) || captchaErr.Code != "NO_RESPONSE" {
		t.Fatalf("error = %v, want NO_RESPONSE", err)
	}
	if called {
		t.Error("an empty token must not reach the network")
	}
}

func TestVerifyInvalidProvider(t *testing.T) {
	err := captcha.Verify(context.Background(), http.DefaultClient, captcha.Provider("bogus"), "sec", "tok", "")
	var captchaErr *captcha.Error
	if err == nil || !asCaptchaError(err, &captchaErr) || captchaErr.Code != "INVALID_PROVIDER" {
		t.Fatalf("error = %v, want INVALID_PROVIDER", err)
	}
}

func TestVerifyProviderErrorStatus(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return respond(`oops`, 500), nil
	})}
	err := captcha.Verify(context.Background(), client, captcha.ProviderHcaptcha, "sec", "tok", "")
	var captchaErr *captcha.Error
	if err == nil || !asCaptchaError(err, &captchaErr) || captchaErr.Code != "REQUEST_FAILED" {
		t.Fatalf("error = %v, want REQUEST_FAILED", err)
	}
}

func asCaptchaError(err error, target **captcha.Error) bool {
	e, ok := err.(*captcha.Error)
	if ok {
		*target = e
	}
	return ok
}
