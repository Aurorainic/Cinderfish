package httpapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/mail"
)

// errIDRateLimited is Cinderfish's own id: Misskey reports rate limits
// through its Fastify plugin without a stable endpoint-level id.
const errIDRateLimited = "7c4f1a62-9e58-4b3d-8f0a-1d6c2b9e5a73"

// Link lifetimes. Verification links only have to outlive a mail queue, so
// an hour is generous; reset links are the account's recovery path and get a
// day, which is what Mailcatcher-hosted Misskey instances use in practice.
const (
	emailVerifyTTL = time.Hour
	emailResetTTL  = 24 * time.Hour
	// emailSendInterval throttles repeated sends per account and purpose so
	// the endpoint cannot be used to flood a mailbox.
	emailSendInterval = time.Minute
)

// requireUser resolves the caller for a handler that needs a signed-in
// account, writing the standard authentication error when there is none.
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (*AuthInfo, bool) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil, false
	}
	return info, true
}

// sendEmail delivers one message, mapping a transport failure to a server
// error the caller reports without leaking the SMTP details.
func (s *Server) sendEmail(r *http.Request, to, subject, body string) error {
	if s.mail == nil {
		return errors.New("httpapi: no mail sender")
	}
	return s.mail.Send(r.Context(), mail.Message{To: to, Subject: subject, Body: body})
}

// handleIEmailVerification implements POST /api/i/email-verification: it
// mails a verification link to the address currently on file.
//
// The optional `email` parameter lets a client assert which address it thinks
// is stored; a mismatch is rejected instead of silently sending elsewhere.
func (s *Server) handleIEmailVerification(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	state, err := s.accounts.EmailState(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if state.Email == "" {
		writeAPIError(w, invalidParamError("no email address is registered"))
		return
	}
	if body.Email != "" && !strings.EqualFold(strings.TrimSpace(body.Email), state.Email) {
		writeAPIError(w, invalidParamError("email does not match the registered address"))
		return
	}
	if !s.emailSendAllowed(w, r, info.User.ID, account.EmailPurposeVerify) {
		return
	}

	challenge, err := s.accounts.IssueEmailChallenge(r.Context(), info.User.ID,
		account.EmailPurposeVerify, state.Email, emailVerifyTTL)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if err := s.sendEmail(r, state.Email, "验证你的邮箱地址",
		verifyEmailBody(s.baseURL(), challenge.Code)); err != nil {
		writeAPIError(w, internalError("Could not send the verification email."))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleVerifyEmail implements POST /api/verify-email: the link target of a
// verification mail marks the stored address verified.
func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	code := strings.TrimSpace(body.Code)
	if code == "" {
		writeAPIError(w, invalidParamError("code is required"))
		return
	}

	challenge, err := s.accounts.ConsumeEmailChallenge(r.Context(), account.EmailPurposeVerify, code)
	if errors.Is(err, account.ErrInvalidEmailChallenge) {
		writeAPIError(w, &APIError{
			Message: "The verification link is invalid or has expired.",
			Code:    "INVALID_EMAIL_CODE",
			ID:      errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if challenge.UserID != info.User.ID {
		// A link minted for another account must not verify this one. The
		// code has already been spent, which is harmless.
		writeAPIError(w, &APIError{
			Message: "The verification link belongs to another account.",
			Code:    "INVALID_EMAIL_CODE",
			ID:      errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	state, err := s.accounts.EmailState(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !strings.EqualFold(state.Email, challenge.Email) {
		// The address changed after the link was sent; the old link proves
		// nothing about the new mailbox.
		writeAPIError(w, &APIError{
			Message: "The email address changed after this link was sent.",
			Code:    "INVALID_EMAIL_CODE",
			ID:      errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.MarkEmailVerified(r.Context(), info.User.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleIUpdateEmail implements POST /api/i/update-email: it replaces the
// address after confirming the password, then mails a link to the new
// mailbox. A wrong password answers the same id envelope as signin so the
// client's existing prompt keeps working.
func (s *Server) handleIUpdateEmail(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	addr := strings.TrimSpace(body.Email)
	if !account.ValidateEmail(addr) {
		writeAPIError(w, invalidParamError("email must be a valid address"))
		return
	}

	okPassword, err := s.accounts.CheckPassword(r.Context(), info.User.ID, body.Password)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !okPassword {
		writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
		return
	}
	if !s.emailSendAllowed(w, r, info.User.ID, account.EmailPurposeVerify) {
		return
	}

	if err := s.accounts.SetEmail(r.Context(), info.User.ID, addr); err != nil {
		if errors.Is(err, account.ErrEmailTaken) {
			writeAPIError(w, &APIError{
				Message: "That email address is already in use.",
				Code:    "EMAIL_ALREADY_REGISTERED",
				ID:      errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeInternalServerError(w)
		return
	}

	challenge, err := s.accounts.IssueEmailChallenge(r.Context(), info.User.ID,
		account.EmailPurposeVerify, addr, emailVerifyTTL)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	// The address is saved at this point; a mail failure is reported but
	// must not roll the change back, so the client can retry the send.
	if err := s.sendEmail(r, addr, "验证你的邮箱地址",
		verifyEmailBody(s.baseURL(), challenge.Code)); err != nil {
		writeAPIError(w, internalError("The address was saved but the verification email could not be sent."))
		return
	}

	packed := packMeDetailed(info.User, info.User.CreatedAt, true, loadProfile(r.Context(), s, info.User.ID))
	s.applyUserCounts(r.Context(), info.User.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// handleEmailAddressAvailable implements POST /api/email-address/available.
// It is public, like the username check the signup form uses.
func (s *Server) handleEmailAddressAvailable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	addr := strings.TrimSpace(body.Email)
	if !account.ValidateEmail(addr) {
		writeJSON(w, http.StatusOK, map[string]any{
			"available": false,
			"reason":    "format",
		})
		return
	}
	if s.accounts == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": true})
		return
	}
	available, err := s.accounts.EmailAvailable(r.Context(), addr, "")
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := map[string]any{"available": available}
	if !available {
		out["reason"] = "used"
	}
	writeJSON(w, http.StatusOK, out)
}

// handleRequestResetPassword implements POST /api/request-reset-password.
//
// The answer is always 204: whether an account, address or neither matched
// is not observable, so the endpoint cannot enumerate registrations.
func (s *Server) handleRequestResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if s.accounts == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	username := strings.TrimSpace(body.Username)
	target := strings.TrimSpace(body.Email)
	if username == "" || target == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	user, err := s.accounts.GetByUsername(r.Context(), username)
	if err != nil || user == nil || user.Host != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	state, err := s.accounts.EmailState(r.Context(), user.ID)
	if err != nil || state.Email == "" || !strings.EqualFold(state.Email, target) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	recent, hasRecent, err := s.accounts.LatestEmailChallengeTime(r.Context(), user.ID, account.EmailPurposeReset)
	if err == nil && hasRecent && time.Since(recent) < emailSendInterval {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	challenge, err := s.accounts.IssueEmailChallenge(r.Context(), user.ID,
		account.EmailPurposeReset, state.Email, emailResetTTL)
	if err != nil {
		// A storage failure on a matching account would otherwise answer
		// differently from the miss path, so it is logged and swallowed too.
		log.Printf("email: issue reset challenge for %s: %v", user.Username, err)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// A dead SMTP host must not turn into an enumeration oracle, so a send
	// failure is logged and still answered with 204.
	if err := s.sendEmail(r, state.Email, "重置你的密码",
		resetPasswordBody(s.baseURL(), challenge.Code)); err != nil {
		log.Printf("email: send reset link for %s: %v", user.Username, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleResetPassword implements POST /api/reset-password: the public
// counterpart of the mail link. Redeeming it also marks the address verified,
// since the link reaching the mailbox is the proof.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	token := strings.TrimSpace(body.Token)
	if token == "" {
		writeAPIError(w, invalidParamError("token is required"))
		return
	}
	if body.Password == "" {
		writeAPIError(w, invalidParamError("password is required"))
		return
	}

	challenge, err := s.accounts.ConsumeEmailChallenge(r.Context(), account.EmailPurposeReset, token)
	if errors.Is(err, account.ErrInvalidEmailChallenge) {
		writeAPIError(w, &APIError{
			Message: "The reset link is invalid or has expired.",
			Code:    "INVALID_TOKEN",
			ID:      errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if err := s.accounts.ResetPassword(r.Context(), challenge.UserID, body.Password); err != nil {
		writeInternalServerError(w)
		return
	}
	if err := s.accounts.MarkEmailVerified(r.Context(), challenge.UserID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// emailSendAllowed enforces the per-purpose throttle and writes the error
// response when it rejects the request.
func (s *Server) emailSendAllowed(w http.ResponseWriter, r *http.Request, userID, purpose string) bool {
	recent, ok, err := s.accounts.LatestEmailChallengeTime(r.Context(), userID, purpose)
	if err != nil {
		return true
	}
	if ok && time.Since(recent) < emailSendInterval {
		writeAPIError(w, &APIError{
			Message: "A link was just sent. Wait a minute before requesting another.",
			Code:    "RATE_LIMIT_EXCEEDED",
			ID:      errIDRateLimited, Kind: "client", Status: http.StatusTooManyRequests,
		})
		return false
	}
	return true
}

func verifyEmailBody(base, code string) string {
	return fmt.Sprintf(`你好：

请打开下面的链接完成邮箱验证：

%s/verify-email/%s

链接一小时内有效。如果不是你本人操作，忽略这封邮件即可。

-- Cinderfish`, base, code)
}

func resetPasswordBody(base, code string) string {
	return fmt.Sprintf(`你好：

请打开下面的链接设置新密码：

%s/reset-password/%s

链接 24 小时内有效。如果不是你本人操作，请忽略这封邮件，你的密码不会被改动。

-- Cinderfish`, base, code)
}
