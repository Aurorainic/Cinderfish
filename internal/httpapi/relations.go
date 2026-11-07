package httpapi

import (
	"errors"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for mute/* and blocking/*, copied from
// packages/backend/src/server/api/endpoints/{mute,blocking}/*.ts.
const (
	errIDMuteNoSuchUser    = "6fef56f3-e765-4957-88e5-c6f65329b8a5"
	errIDMuteeIsYourself   = "a4619cb2-5f23-484b-9301-94c903074e10"
	errIDAlreadyMuting     = "7e7359cb-160c-4956-b08f-4d1c653cd007"
	errIDUnmuteNoSuchUser  = "b851d00b-8ab1-4a56-8b1b-e24187cb48ef"
	errIDBlockNoSuchUser   = "7cc4f851-e2f1-4621-9633-ec9e1d00c01e"
	errIDBlockeeIsYourself = "88b19138-f28d-42c0-8499-6a31bbd0fdc6"
	errIDAlreadyBlocking   = "787fed64-acb9-464a-82eb-afbd745b9614"
	errIDUnblockNoSuchUser = "8621d8bf-c358-4303-a066-5ea78610eb3f"
)

// handleMuteCreate is POST /api/mute/create.
func (s *Server) handleMuteCreate(w http.ResponseWriter, r *http.Request) {
	body := struct {
		UserID string `json:"userId"`
	}{}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDMuteNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	if target.ID == info.User.ID {
		writeAPIError(w, &APIError{
			Message: "Mutee is yourself.", Code: "MUTEE_IS_YOURSELF",
			ID: errIDMuteeIsYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	muted, err := s.accounts.IsMuted(r.Context(), info.User.ID, target.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if muted {
		writeAPIError(w, &APIError{
			Message: "You are already muting that user.", Code: "ALREADY_MUTING",
			ID: errIDAlreadyMuting, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.Mute(r.Context(), info.User.ID, target.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleMuteDelete is POST /api/mute/delete. Deleting a mute that does not
// exist is a no-op, so the settings page can retry safely.
func (s *Server) handleMuteDelete(w http.ResponseWriter, r *http.Request) {
	body := struct {
		UserID string `json:"userId"`
	}{}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDUnmuteNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	if err := s.accounts.Unmute(r.Context(), info.User.ID, target.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleMuteList is POST /api/mute/list.
func (s *Server) handleMuteList(w http.ResponseWriter, r *http.Request) {
	s.serveRelationList(w, r, false)
}

// handleBlockingCreate is POST /api/blocking/create.
func (s *Server) handleBlockingCreate(w http.ResponseWriter, r *http.Request) {
	body := struct {
		UserID string `json:"userId"`
	}{}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDBlockNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	if target.ID == info.User.ID {
		writeAPIError(w, &APIError{
			Message: "Blockee is yourself.", Code: "BLOCKEE_IS_YOURSELF",
			ID: errIDBlockeeIsYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	blocked, err := s.accounts.IsBlocked(r.Context(), info.User.ID, target.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if blocked {
		writeAPIError(w, &APIError{
			Message: "You are already blocking that user.", Code: "ALREADY_BLOCKING",
			ID: errIDAlreadyBlocking, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.accounts.Block(r.Context(), info.User.ID, target.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	packed := packUserDetailed(target, target.CreatedAt, false, loadProfile(r.Context(), s, target.ID))
	// Misskey answers with the freshly blocked user; the viewer's own state
	// is what `isBlocked` reports, so it is true here even though the generic
	// packer defaults it to false.
	packed["isBlocked"] = true
	s.publishMeUpdated(r.Context(), info.User.ID)
	writeJSON(w, http.StatusOK, packed)
}

// handleBlockingDelete is POST /api/blocking/delete.
func (s *Server) handleBlockingDelete(w http.ResponseWriter, r *http.Request) {
	body := struct {
		UserID string `json:"userId"`
	}{}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil {
		if errors.Is(err, account.ErrUserNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDUnblockNoSuchUser, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	if err := s.accounts.Unblock(r.Context(), info.User.ID, target.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	s.publishMeUpdated(r.Context(), info.User.ID)
	w.WriteHeader(http.StatusNoContent)
}

// handleBlockingList is POST /api/blocking/list.
func (s *Server) handleBlockingList(w http.ResponseWriter, r *http.Request) {
	s.serveRelationList(w, r, true)
}

// serveRelationList answers mute/list and blocking/list. The two responses
// differ only in the key holding the other user: `mutee` or `blockee`.
func (s *Server) serveRelationList(w http.ResponseWriter, r *http.Request, blocking bool) {
	body := struct {
		Limit   int    `json:"limit"`
		SinceID string `json:"sinceId"`
		UntilID string `json:"untilId"`
	}{}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	var (
		rows []*account.Relation
		key  string
	)
	if blocking {
		rows, err = s.accounts.Blocks(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID)
		key = "blockee"
	} else {
		rows, err = s.accounts.Mutes(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID)
		key = "mutee"
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(rows))
	for _, rel := range rows {
		out = append(out, map[string]any{
			"id":        rel.ID,
			"createdAt": rel.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
			key:         packUserDetailed(rel.User, rel.User.CreatedAt, false, loadProfile(r.Context(), s, rel.User.ID)),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
