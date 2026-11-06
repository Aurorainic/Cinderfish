package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for the application and auth-session endpoints, copied
// from packages/backend/src/server/api/endpoints/{app,auth}.
const (
	errIDAppShowNoSuchApp      = "dce83913-2dc6-4093-8a7b-71dbb11718a3"
	errIDSessionNoSuchApp      = "92f93e63-428e-4f2f-a5a4-39e1407fe998"
	errIDSessionNoSuchSession  = "bd72c97d-eba7-4adb-a467-f171b8847250"
	errIDAcceptNoSuchSession   = "9c72d8de-391a-43c1-9d06-08d29efde8df"
	errIDUserKeyNoSuchApp      = "fcab192a-2c5a-43b7-8ad8-9b7054d8d40d"
	errIDUserKeyNoSuchSession  = "5b5a1503-8bc8-4bd0-8054-dc189e8cdcb3"
	errIDUserKeyPendingSession = "8c8a4145-02cc-4cca-8e66-29ba60445a8e"
)

// packApp renders a registered application. The secret is only included when
// the caller owns the app and authenticated with a native token, which is
// Misskey's rule (appEntityService pack with includeSecret).
func packApp(app *account.AppRecord, includeSecret, isAuthorized bool) map[string]any {
	out := map[string]any{
		"id":           app.ID,
		"name":         app.Name,
		"callbackUrl":  app.CallbackURL,
		"permission":   app.Permission,
		"isAuthorized": isAuthorized,
	}
	if includeSecret {
		out["secret"] = app.Secret
	}
	return out
}

// handleAppCreate implements POST /api/app/create. It needs no credential:
// Misskey lets an anonymous client register an application and receive its
// secret, which is what the legacy authorization flow starts with.
func (s *Server) handleAppCreate(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permission  []string `json:"permission"`
		CallbackURL *string  `json:"callbackUrl"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.Name == "" {
		writeAPIError(w, invalidParamError("name is required"))
		return
	}

	userID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		userID = info.User.ID
	}
	callback := ""
	if body.CallbackURL != nil {
		callback = *body.CallbackURL
	}
	app, err := s.tokens.CreateApp(r.Context(), userID, body.Name, body.Description, body.Permission, callback)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packApp(app, true, userID != ""))
}

// handleAppShow implements POST /api/app/show.
func (s *Server) handleAppShow(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		AppID string `json:"appId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	app, err := s.tokens.AppByID(r.Context(), body.AppID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such app.", Code: "NO_SUCH_APP",
			ID: errIDAppShowNoSuchApp, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil {
		info = nil
	}
	isAuthorized := false
	if info != nil {
		isAuthorized, _ = s.tokens.AppAuthorized(r.Context(), app.ID, info.User.ID)
	}
	includeSecret := info != nil && info.Token == nil && app.UserID != nil && *app.UserID == info.User.ID
	writeJSON(w, http.StatusOK, packApp(app, includeSecret, isAuthorized))
}

// handleAuthSessionGenerate implements POST /api/auth/session/generate: the
// app exchanges its secret for a one-time approval URL.
func (s *Server) handleAuthSessionGenerate(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		AppSecret string `json:"appSecret"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	app, err := s.tokens.AppBySecret(r.Context(), body.AppSecret)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such app.", Code: "NO_SUCH_APP",
			ID: errIDSessionNoSuchApp, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	session, err := s.tokens.CreateAuthSession(r.Context(), app.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": session.Token,
		"url":   s.baseURL() + "/auth/" + session.Token,
	})
}

// handleAuthSessionShow implements POST /api/auth/session/show: what the
// approval page reads to display the app and its permissions.
func (s *Server) handleAuthSessionShow(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	session, err := s.tokens.SessionByToken(r.Context(), body.Token)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such session.", Code: "NO_SUCH_SESSION",
			ID: errIDSessionNoSuchSession, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	app, err := s.tokens.AppByID(r.Context(), session.AppID)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil {
		info = nil
	}
	isAuthorized := false
	if info != nil {
		isAuthorized, _ = s.tokens.AppAuthorized(r.Context(), app.ID, info.User.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    session.ID,
		"app":   packApp(app, false, isAuthorized),
		"token": session.Token,
	})
}

// handleAuthAccept implements POST /api/auth/accept: the signed-in user
// approves the session, which issues the app's access token.
func (s *Server) handleAuthAccept(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
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
	if _, err := s.tokens.AcceptSession(r.Context(), body.Token, info.User.ID); err != nil {
		if errors.Is(err, account.ErrSessionNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such session.", Code: "NO_SUCH_SESSION",
				ID: errIDAcceptNoSuchSession, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAuthSessionUserKey implements POST /api/auth/session/userkey: the app
// polls with its secret to collect the token the user approved. The session is
// consumed by a successful call.
func (s *Server) handleAuthSessionUserKey(w http.ResponseWriter, r *http.Request) {
	if s.tokens == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		AppSecret string `json:"appSecret"`
		Token     string `json:"token"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	app, err := s.tokens.AppBySecret(r.Context(), body.AppSecret)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such app.", Code: "NO_SUCH_APP",
			ID: errIDUserKeyNoSuchApp, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	at, userID, err := s.tokens.SessionUserKey(r.Context(), app.ID, body.Token)
	switch {
	case errors.Is(err, account.ErrSessionNotFound):
		writeAPIError(w, &APIError{
			Message: "No such session.", Code: "NO_SUCH_SESSION",
			ID: errIDUserKeyNoSuchSession, Kind: "client", Status: http.StatusNotFound,
		})
		return
	case errors.Is(err, account.ErrSessionPending):
		writeAPIError(w, &APIError{
			Message: "This session is not completed yet.", Code: "PENDING_SESSION",
			ID: errIDUserKeyPendingSession, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	case err != nil:
		writeInternalServerError(w)
		return
	}

	user, err := s.accounts.GetByID(r.Context(), userID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	packed := packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID))
	s.packMovedTo(r.Context(), user, packed)
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": at.Token,
		"user":        packed,
	})
}
