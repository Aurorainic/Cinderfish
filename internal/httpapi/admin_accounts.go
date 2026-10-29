package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// handleAdminAccountsCreate implements POST /api/admin/accounts/create. It
// is how an administrator adds a user while self-service registration is
// closed (registration_mode = "admin").
func (s *Server) handleAdminAccountsCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
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

	user, err := s.accounts.Create(r.Context(), body.Username, body.Password)
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
		default:
			writeInternalServerError(w)
		}
		return
	}

	packed := packUserDetailed(user, user.CreatedAt, false, loadProfile(r.Context(), s, user.ID))
	s.applyUserCounts(r.Context(), user.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// handleAdminShowUsers implements POST /api/admin/show-users.
func (s *Server) handleAdminShowUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	users, err := s.accounts.ListLocal(r.Context(), body.Limit, body.Offset)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(users))
	for _, u := range users {
		packed := packUserDetailed(u, u.CreatedAt, false, loadProfile(r.Context(), s, u.ID))
		s.applyUserCounts(r.Context(), u.ID, packed)
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminSuspendUser implements POST /api/admin/suspend-user and
// /api/admin/unsuspend-user.
func (s *Server) handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	s.suspendUser(w, r, true)
}

func (s *Server) handleAdminUnsuspendUser(w http.ResponseWriter, r *http.Request) {
	s.suspendUser(w, r, false)
}

func (s *Server) suspendUser(w http.ResponseWriter, r *http.Request, suspend bool) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}

	// Suspending yourself would lock the instance's own administration
	// out; Misskey refuses it too.
	if info, _ := s.authenticate(r.Context(), r, bodyAsMap(r)); info != nil && info.User.ID == body.UserID {
		writeAPIError(w, &APIError{
			Message: "You cannot suspend yourself.", Code: "CANNOT_SUSPEND_YOURSELF",
			ID: errIDForbidden, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	if err := s.accounts.SetSuspended(r.Context(), body.UserID, suspend); err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}
