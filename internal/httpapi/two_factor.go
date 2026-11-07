package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/auth"
)

// Error ids for the TOTP endpoints. Misskey clients match on the id, so the
// invalid-token id is Firefish's own.
const (
	errIDInvalidTOTP    = "cdf1235b-ac71-46d4-a3a6-84ccce48df6f"
	errIDTOTPNotStarted = "7d1f0c55-5a4a-4f4b-a3f3-4c6d4a9b2e01"
)

// handleI2FARegister implements POST /api/i/2fa/register.
//
// It verifies the caller's password, stores a pending secret and returns the
// otpauth URI plus the raw secret. Firefish also returns a `qr` data URL;
// rendering a QR code would need an image encoder dependency, so the field
// is present but empty and the page tells the user to enter the secret
// manually.
func (s *Server) handleI2FARegister(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	ok, err := s.accounts.CheckPassword(r.Context(), info.User.ID, body.Password)
	if err != nil {
		writeAPIError(w, internalError("Could not verify the password."))
		return
	}
	if !ok {
		writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
		return
	}

	secret := auth.GenerateTOTPSecret()
	if err := s.accounts.BeginTwoFactor(r.Context(), info.User.ID, secret); err != nil {
		writeAPIError(w, internalError("Could not start two-factor registration."))
		return
	}

	// The issuer is what the user sees in their authenticator app, so it has
	// to be the public domain rather than the listen address in cfg.Host.
	issuer := s.cfg.Domain
	if issuer == "" {
		issuer = "localhost"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"qr":     "",
		"url":    auth.TOTPURL(secret, issuer, info.User.Username),
		"secret": secret,
		"label":  info.User.Username,
		"issuer": issuer,
	})
}

// handleI2FADone implements POST /api/i/2fa/done: the pending secret becomes
// active once the caller proves it can produce a code.
func (s *Server) handleI2FADone(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	state, err := s.accounts.GetTwoFactorState(r.Context(), info.User.ID)
	if err != nil {
		writeAPIError(w, internalError("Could not read the two-factor state."))
		return
	}
	if state.TempSecret == "" {
		writeAPIError(w, &APIError{
			Message: "Two-factor registration has not been started.",
			Code:    "TWO_FACTOR_NOT_STARTED", ID: errIDTOTPNotStarted,
			Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if !auth.ValidateTOTP(state.TempSecret, body.Token) {
		writeAPIError(w, &APIError{
			Message: "The token is invalid.", Code: "TOTP_TOKEN_INVALID",
			ID: errIDInvalidTOTP, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.EnableTwoFactor(r.Context(), info.User.ID); err != nil {
		if errors.Is(err, account.ErrTwoFactorNotStarted) {
			writeAPIError(w, &APIError{
				Message: "Two-factor registration has not been started.",
				Code:    "TWO_FACTOR_NOT_STARTED", ID: errIDTOTPNotStarted,
				Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeAPIError(w, internalError("Could not enable two-factor authentication."))
		return
	}
	// Other tabs show the two-factor state from `$i`; Misskey publishes
	// meUpdated from i/2fa/done for the same reason.
	s.publishMeUpdated(r.Context(), info.User.ID)
	w.WriteHeader(http.StatusNoContent)
}

// handleI2FAUnregister implements POST /api/i/2fa/unregister. The password
// check keeps a stolen session token from removing the second factor.
func (s *Server) handleI2FAUnregister(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	ok, err := s.accounts.CheckPassword(r.Context(), info.User.ID, body.Password)
	if err != nil {
		writeAPIError(w, internalError("Could not verify the password."))
		return
	}
	if !ok {
		writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
		return
	}
	if err := s.accounts.DisableTwoFactor(r.Context(), info.User.ID); err != nil {
		writeAPIError(w, internalError("Could not disable two-factor authentication."))
		return
	}
	s.publishMeUpdated(r.Context(), info.User.ID)
	w.WriteHeader(http.StatusNoContent)
}

// handleI2FAPasswordLess implements POST /api/i/2fa/password-less. Security
// keys are not supported, so enabling password-less login answers
// NO_SECURITY_KEY (as Firefish does when the user has no key) and disabling
// it is a no-op. The switch in the settings page stays disabled.
func (s *Server) handleI2FAPasswordLess(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Value bool `json:"value"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.Value {
		writeAPIError(w, &APIError{
			Message: "No security key.", Code: "NO_SECURITY_KEY",
			ID:   "f9c54d7f-d4c2-4d3c-9a8f-a70daac86512",
			Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// twoFactorFields adds the three TOTP-related UserDetailed fields. Both
// users/show and POST /api/i carry them; the signin dialog needs the first
// one to decide whether to ask for a code.
func twoFactorFields(packed map[string]any, enabled bool) {
	packed["twoFactorEnabled"] = enabled
	packed["usePasswordLessLogin"] = false
	packed["securityKeys"] = false
}

func internalError(message string) *APIError {
	return &APIError{
		Message: message, Code: "INTERNAL_ERROR", ID: errIDInternal,
		Kind: "server", Status: http.StatusInternalServerError,
	}
}
