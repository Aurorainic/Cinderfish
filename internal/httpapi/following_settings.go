package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for the per-follow endpoints, copied from
// packages/backend/src/server/api/endpoints/following/{update,list,invalidate}.ts.
const (
	errIDFollowUpdateNoSuchUser = "14318698-f67e-492a-99da-5353a5ac52be"
	errIDFollowUpdateYourself   = "4c4cbaf9-962a-463b-8418-a5e365dbf2eb"
	errIDFollowUpdateNotFollow  = "b8dc75cf-1cb5-46c9-b14b-5f1ffbd782c9"
)

// handleFollowingList implements POST /api/following/list: the caller's own
// followings with their per-follow settings.
//
// Unlike users/following this endpoint is always the caller's own list (there
// is no userId parameter) and it can filter to the accounts that notify.
func (s *Server) handleFollowingList(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Notification bool   `json:"notification"`
		Limit        int    `json:"limit"`
		SinceID      string `json:"sinceId"`
		UntilID      string `json:"untilId"`
		SinceDate    *int64 `json:"sinceDate"`
		UntilDate    *int64 `json:"untilDate"`
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
	query := account.FollowQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	list, err := s.accounts.Following(r.Context(), info.User.ID, query)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(list))
	for _, f := range list {
		if body.Notification && f.Notify == "" {
			continue
		}
		out = append(out, packFollowingItem(r, s, f, info.User.ID))
	}
	writeJSON(w, http.StatusOK, out)
}

// packFollowingItem renders one Following row: the relation plus the followee
// as a UserLite, which is what both following/list and users/following return.
func packFollowingItem(r *http.Request, s *Server, f *account.Follow, followerID string) map[string]any {
	item := map[string]any{
		"id":         f.ID,
		"createdAt":  f.CreatedAt.UTC().Format(time.RFC3339Nano),
		"followerId": followerID,
		"followeeId": f.User.ID,
		"followee":   packUserLite(f.User, "", loadProfile(r.Context(), s, f.User.ID)),
	}
	if f.Notify != "" {
		item["notify"] = f.Notify
	}
	if !f.WithReplies {
		item["withReplies"] = false
	}
	return item
}

// handleFollowingUpdate implements POST /api/following/update: the notify and
// withReplies switches of one follow.
func (s *Server) handleFollowingUpdate(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		UserID      string  `json:"userId"`
		Notify      *string `json:"notify"`
		WithReplies *bool   `json:"withReplies"`
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
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	if body.UserID == info.User.ID {
		writeAPIError(w, &APIError{
			Message: "Followee is yourself.", Code: "FOLLOWEE_IS_YOURSELF",
			ID: errIDFollowUpdateYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDFollowUpdateNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if err := s.accounts.UpdateFollowing(r.Context(), info.User.ID, target.ID, body.Notify, body.WithReplies); err != nil {
		if errors.Is(err, account.ErrNotFollowing) {
			writeAPIError(w, &APIError{
				Message: "You are not following that user.", Code: "NOT_FOLLOWING",
				ID: errIDFollowUpdateNotFollow, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	// Misskey answers with the *caller*, not the followee.
	writeJSON(w, http.StatusOK, packUserLite(info.User, s.cfg.Domain, loadProfile(r.Context(), s, info.User.ID)))
}

// handleFollowingUpdateAll implements POST /api/following/update-all: the same
// switches applied to every account the caller follows.
func (s *Server) handleFollowingUpdateAll(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		Notify      *string `json:"notify"`
		WithReplies *bool   `json:"withReplies"`
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
	if err := s.accounts.UpdateAllFollowing(r.Context(), info.User.ID, body.Notify, body.WithReplies); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleFollowingInvalidate implements POST /api/following/invalidate: the
// caller (as followee) removes a follower without the follower's consent.
//
// The row is deleted locally. No Undo is delivered: that activity would have
// to be signed as the follower, whose key this instance does not hold. A local
// follower gets the streaming unfollow event so an open client updates.
func (s *Server) handleFollowingInvalidate(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil || s.federationStore == nil {
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
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	if body.UserID == info.User.ID {
		writeAPIError(w, &APIError{
			Message: "Follower is yourself.", Code: "FOLLOWER_IS_YOURSELF",
			ID: errIDFollowUpdateYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	follower, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || follower == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDFollowUpdateNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if !s.federationStore.HasFollowing(r.Context(), follower.ID, info.User.ID) {
		writeAPIError(w, &APIError{
			Message: "You are not following that user.", Code: "NOT_FOLLOWING",
			ID: errIDFollowUpdateNotFollow, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err := s.federationStore.Unfollow(r.Context(), follower.ID, info.User.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	s.publishUnfollowEvent(follower.ID, info.User.ID)
	writeJSON(w, http.StatusOK, packUserLite(follower, s.cfg.Domain, loadProfile(r.Context(), s, follower.ID)))
}
