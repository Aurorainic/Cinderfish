package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/auth"
)

// Error ids used by the account endpoints. Misskey's signin flow returns
// errors that carry only an id, so the id is what clients match on.
const (
	errIDNoSuchUser      = "6cc579cc-885d-43d8-95c2-b8c7fc963280"
	errIDUserSuspended   = "e03a5f46-d309-4865-9b69-56282d94e1eb"
	errIDInvalidPassword = "932c904e-9460-45b7-9ce6-7ed33be7eb2c"
	errIDSignupFailed    = "4d371c9a-3a0f-4c8b-9b8e-2f9c0a1d3e45"
)

// writeIDError emits Misskey's id-only error envelope, used by signin.
func writeIDError(w http.ResponseWriter, status int, id string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"id":"` + id + `"}}`))
}

// handleSignup implements POST /api/signup.
//
// Registration policy comes from the configuration:
//   - open: the account is created directly;
//   - invite: a valid invitationCode is required;
//   - admin: self-service signup is refused.
//
// On success the response is the MeDetailed entity plus a `token` field,
// matching Misskey's SignupApiService.
func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAPIError(w, &APIError{
			Message: "Signup is not available.", Code: "SIGNUP_UNAVAILABLE",
			ID: errIDSignupFailed, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		InvitationCode string `json:"invitationCode"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	mode := s.registrationMode()
	user, token, err := s.accounts.Register(r.Context(), mode, body.Username, body.Password, body.InvitationCode)
	if err != nil {
		var regErr *account.RegistrationError
		switch {
		case errors.As(err, &regErr):
			writeAPIError(w, &APIError{
				Message: regErr.Message, Code: regErr.Code,
				ID: errIDSignupFailed, Kind: "client", Status: http.StatusBadRequest,
			})
		case errors.Is(err, account.ErrUsernameTaken):
			writeAPIError(w, &APIError{
				Message: "Username is already taken.", Code: "DUPLICATED_USERNAME",
				ID: errIDSignupFailed, Kind: "client", Status: http.StatusBadRequest,
			})
		case errors.Is(err, account.ErrInvalidInvite):
			writeAPIError(w, &APIError{
				Message: "The invitation code is invalid.", Code: "INVALID_INVITE",
				ID: errIDSignupFailed, Kind: "client", Status: http.StatusBadRequest,
			})
		default:
			writeAPIError(w, &APIError{
				Message: "Signup failed.", Code: "SIGNUP_FAILED",
				ID: errIDSignupFailed, Kind: "server", Status: http.StatusInternalServerError,
			})
		}
		return
	}

	packed := packMeDetailed(user, user.CreatedAt, true, loadProfile(r.Context(), s, user.ID))
	s.applyUserCounts(r.Context(), user.ID, packed)
	packed["token"] = token
	writeJSON(w, http.StatusOK, packed)
}

// loadProfile fetches a user's profile for the packers, tolerating a
// missing profile row or a transient read failure by passing nothing.
func loadProfile(ctx context.Context, s *Server, userID string) *account.Profile {
	if s.accounts == nil {
		return nil
	}
	prof, err := s.accounts.GetProfile(ctx, userID)
	if err != nil {
		return nil
	}
	return prof
}

// registrationMode maps the configured mode onto the account service.
func (s *Server) registrationMode() account.RegistrationMode {
	switch s.cfg.RegistrationMode {
	case "open":
		return account.RegistrationOpen
	case "invite":
		return account.RegistrationInvite
	default:
		return account.RegistrationAdmin
	}
}

// handleSignin implements POST /api/signin.
//
// Success returns {"finished":true,"id":...,"i":<token>}; failures use
// Misskey's id-only error envelope.
func (s *Server) handleSignin(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAPIError(w, &APIError{
			Message: "Signin is not available.", Code: "SIGNIN_UNAVAILABLE",
			ID: errIDInvalidPassword, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Username == "" {
		writeIDError(w, http.StatusBadRequest, errIDNoSuchUser)
		return
	}

	user, err := s.accounts.Authenticate(r.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeIDError(w, http.StatusNotFound, errIDNoSuchUser)
			return
		}
		if errors.Is(err, account.ErrInvalidCredentials) {
			writeIDError(w, http.StatusForbidden, errIDInvalidPassword)
			return
		}
		writeAPIError(w, &APIError{
			Message: "Signin failed.", Code: "SIGNIN_FAILED",
			ID: errIDInvalidPassword, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	if user.IsSuspended {
		writeIDError(w, http.StatusForbidden, errIDUserSuspended)
		return
	}

	_ = s.accounts.RecordSignin(r.Context(), user.ID, clientIP(r), true)

	writeJSON(w, http.StatusOK, map[string]any{
		"finished": true,
		"id":       user.ID,
		"i":        user.Token,
	})
}

// handleUsernameAvailable implements POST /api/username/available.
func (s *Server) handleUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	available, err := s.accounts.UsernameAvailable(r.Context(), body.Username)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not check the username.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"available": available})
}

// handleI implements POST /api/i and requires a credential.
func (s *Server) handleI(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = decodeOptionalJSON(r, &body)

	info, err := s.authenticate(r.Context(), r, body)
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	// Misskey only includes secrets for native-token requests.
	includeSecrets := info.Token == nil
	packed := packMeDetailed(info.User, info.User.CreatedAt, includeSecrets, loadProfile(r.Context(), s, info.User.ID))
	s.applyUserCounts(r.Context(), info.User.ID, packed)
	s.applyUnreadNotifications(r.Context(), info.User.ID, packed)
	s.applyPinnedNotes(r.Context(), info.User.ID, info.User.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// handleUsersShow implements POST /api/users/show.
func (s *Server) handleUsersShow(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		UserID   string `json:"userId"`
		Username string `json:"username"`
		Host     string `json:"host"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	var user *account.User
	var err error
	switch {
	case body.UserID != "":
		user, err = s.accounts.GetByID(r.Context(), body.UserID)
	case body.Username != "" && body.Host == "":
		user, err = s.accounts.GetByUsername(r.Context(), body.Username)
	default:
		writeAPIError(w, &APIError{
			Message: "Invalid param.", Code: "INVALID_PARAM",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeAPIError(w, &APIError{
			Message: "Could not load the user.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	packed := packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID))
	// users/show does not require credentials, but an authenticated viewer
	// gets their relationship with the target filled in. A missing or bad
	// token simply means anonymous, never a 401.
	viewerID := ""
	if info, authErr := s.authenticate(r.Context(), r, bodyAsMap(r)); authErr == nil && info != nil {
		viewerID = info.User.ID
		if s.federationStore != nil && info.User.ID != user.ID {
			packed["isFollowing"] = s.federationStore.HasFollowing(r.Context(), info.User.ID, user.ID)
			packed["isFollowed"] = s.federationStore.HasFollowing(r.Context(), user.ID, info.User.ID)
			packed["hasPendingFollowRequestFromYou"] = s.federationStore.HasPendingFollowRequest(r.Context(), info.User.ID, user.ID)
			packed["hasPendingFollowRequestToYou"] = s.federationStore.HasPendingFollowRequest(r.Context(), user.ID, info.User.ID)
			if muted, muteErr := s.accounts.IsMuted(r.Context(), info.User.ID, user.ID); muteErr == nil {
				packed["isMuted"] = muted
			}
			if blocked, blockErr := s.accounts.IsBlocked(r.Context(), info.User.ID, user.ID); blockErr == nil {
				packed["isBlocked"] = blocked
			}
		}
	}
	s.applyUserCounts(r.Context(), user.ID, packed)
	s.applyPinnedNotes(r.Context(), user.ID, viewerID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// handleMiAuthGenToken implements POST /api/miauth/gen-token: it issues a
// MiAuth access token for the authenticated user.
func (s *Server) handleMiAuthGenToken(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body map[string]any
	_ = decodeOptionalJSON(r, &body)

	info, err := s.authenticate(r.Context(), r, body)
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	session, _ := body["session"].(string)
	if session == "" {
		writeAPIError(w, &APIError{
			Message: "Invalid param.", Code: "INVALID_PARAM",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	name, _ := body["name"].(string)
	description, _ := body["description"].(string)
	iconURL, _ := body["iconUrl"].(string)

	var scopes []string
	if raw, ok := body["permission"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				scopes = append(scopes, s)
			}
		}
	}

	token := auth.GenerateMiAuthToken()
	if _, err := s.tokens.Issue(r.Context(), info.User.ID, token, session, name, description, iconURL, scopes); err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not issue the token.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}
