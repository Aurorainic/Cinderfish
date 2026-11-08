package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// Misskey error ids for the users/lists endpoints, copied from
// packages/backend/src/server/api/endpoints/users/lists/*.ts. The same
// logical error often carries a different id per endpoint upstream, so the
// ids travel with the handler through userListErrorIDs.
const (
	errIDTooManyListsCreate     = "0cf21a28-7715-4f39-a20d-777bfdb8d138"
	errIDTooManyListsCopy       = "e9c105b2-c595-47de-97fb-7f7c2c33e92f"
	errIDNoSuchListCopy         = "9292f798-6175-4f7d-93f4-b6742279667d"
	errIDYouHaveBeenBlockedCopy = "a2497f2a-2389-439c-8626-5298540530f4"
	errIDTooManyUsersCopy       = "1845ea77-38d1-426e-8e4e-8b83b24f5bd7"
	errIDNoSuchListDelete       = "78436795-db79-42f5-b1e2-55ea2cf19166"
	errIDNoSuchUserListFavorite = "7dbaf3cf-7b42-4b8f-b431-b3919e580dbe"
	errIDAlreadyFavorited       = "6425bba0-985b-461e-af1b-518070e72081"
	errIDNoSuchUserListUnfav    = "baedb33e-76b8-4b0c-86a8-9375c0a7b94b"
	// Upstream unfavourite reuses the ALREADY_FAVORITED code for "you have
	// not favourited it" — a typo clients may match on, so it is kept.
	errIDAlreadyFavoritedUnfav  = "835c4b27-463d-4cfa-969b-a9058678d465"
	errIDNoSuchListMemberships  = "7bc05c21-1d7a-41ae-88f1-66820f4dc686"
	errIDNoSuchUserListsList    = "a8af4a82-0980-4cc4-a6af-8b0ffd54465e"
	errIDRemoteUserNotAllowed   = "53858f1b-3315-4a01-81b7-db9b48d4b79a"
	errIDNoSuchListPull         = "7f44670e-ab16-43b8-b4c1-ccd2ee89cc02"
	errIDNoSuchUserPull         = "588e7f72-c744-4a61-b180-d354e912bda2"
	errIDNoSuchListPush         = "2214501d-ac96-4049-b717-91e42272a711"
	errIDNoSuchUserPush         = "a89abd3d-f0bc-4cce-beb1-2f446f4f1e6a"
	errIDAlreadyAddedPush       = "1de7c884-1595-49e9-857e-61f12f4d4fc5"
	errIDYouHaveBeenBlockedPush = "990232c5-3f9d-4d83-9f3f-ef27b6332a4b"
	errIDTooManyUsersPush       = "2dd9752e-a338-413d-8eec-41814430989b"
	errIDNoSuchListUpdate       = "796666fe-3dff-4d39-becb-8a5932c1d5b7"
	errIDYourAccountMoved       = "56f20ec9-fd06-4fa5-841b-edd6d7d4fa31"
	errIDNoSuchListListTimeline = "8fb1fbd5-e476-4c37-9fb0-43d55b63a2ff"
)

// limitListsPush is Misskey's push rate cap (30 per hour).
var limitListsPush = rateLimit{bucket: "users/lists/push", max: 30, duration: time.Hour}

// userListErrorIDs carries the per-endpoint error ids of the list family.
type userListErrorIDs struct {
	noSuchList   string
	tooManyLists string
	tooManyUsers string
	alreadyAdded string
	blocked      string
}

// packUserList renders the UserList schema. userIds is always filled, as
// upstream's UserListEntityService does even though the schema marks it
// optional.
func (s *Server) packUserList(ctx context.Context, l *account.UserList) (map[string]any, error) {
	memberIDs, err := s.accounts.MemberIDs(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		ids[i] = id
	}
	return map[string]any{
		"id":        l.ID,
		"createdAt": l.CreatedAt.UTC().Format(time.RFC3339Nano),
		"name":      l.Name,
		"isPublic":  l.IsPublic,
		"userIds":   ids,
	}, nil
}

// writeUserListError maps the service sentinel errors of the list family
// onto Misskey's error envelope.
func writeUserListError(w http.ResponseWriter, err error, ids userListErrorIDs) {
	switch {
	case errors.Is(err, account.ErrUserListNotFound),
		errors.Is(err, account.ErrUserListNotPublic):
		writeAPIError(w, &APIError{
			Message: "No such list.", Code: "NO_SUCH_LIST",
			ID: ids.noSuchList, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrAccountMoved):
		writeAPIError(w, &APIError{
			Message: "Your account has moved.", Code: "YOUR_ACCOUNT_MOVED",
			ID: errIDYourAccountMoved, Kind: "permission", Status: http.StatusForbidden,
		})
	case errors.Is(err, account.ErrTooManyUserLists):
		writeAPIError(w, &APIError{
			Message: "You have too many lists.", Code: "TOO_MANY_USERLISTS",
			ID: ids.tooManyLists, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrTooManyMembers):
		writeAPIError(w, &APIError{
			Message: "You have too many users in that list.", Code: "TOO_MANY_USERS",
			ID: ids.tooManyUsers, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrAlreadyAdded):
		writeAPIError(w, &APIError{
			Message: "That user has already been added to that list.", Code: "ALREADY_ADDED",
			ID: ids.alreadyAdded, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrMemberBlocksYou):
		writeAPIError(w, &APIError{
			Message: "You have been blocked by that user.", Code: "YOU_HAVE_BEEN_BLOCKED",
			ID: ids.blocked, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrAlreadyFavorited):
		writeAPIError(w, &APIError{
			Message: "The list has already been favorited.", Code: "ALREADY_FAVORITED",
			ID: errIDAlreadyFavorited, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrNotFavorited):
		// Upstream's mislabelled code kept verbatim.
		writeAPIError(w, &APIError{
			Message: "You have not favorited that list.", Code: "ALREADY_FAVORITED",
			ID: errIDAlreadyFavoritedUnfav, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrListNameInvalid):
		writeAPIError(w, invalidParamError("name must be 1..100 characters"))
	case errors.Is(err, account.ErrNotAMember):
		// Upstream 500s here (UserListService.ts L161-163); we answer a
		// client error with pull's NO_SUCH_USER id. Decision recorded in
		// docs/HANDOVER.md.
		writeAPIError(w, &APIError{
			Message: "That user is not a member of the list.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserPull, Kind: "client", Status: http.StatusNotFound,
		})
	default:
		writeInternalServerError(w)
	}
}

// requireListCaller authenticates a mutating list request.
func (s *Server) requireListCaller(w http.ResponseWriter, r *http.Request) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	return info
}

// handleUserListsCreate implements POST /api/users/lists/create.
func (s *Server) handleUserListsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	l, err := s.accounts.CreateList(r.Context(), info.User.ID, body.Name)
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{tooManyLists: errIDTooManyListsCreate})
		return
	}
	packed, err := s.packUserList(r.Context(), l)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleUserListsCreateFromPublic implements POST
// /api/users/lists/create-from-public.
func (s *Server) handleUserListsCreateFromPublic(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		ListID string `json:"listId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}

	l, err := s.accounts.CopyPublicList(r.Context(), info.User.ID, body.ListID, body.Name,
		func(memberID string) (bool, error) {
			return s.accounts.IsBlocked(r.Context(), memberID, info.User.ID)
		})
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{
			noSuchList:   errIDNoSuchListCopy,
			tooManyLists: errIDTooManyListsCopy,
			tooManyUsers: errIDTooManyUsersCopy,
			blocked:      errIDYouHaveBeenBlockedCopy,
		})
		return
	}
	packed, err := s.packUserList(r.Context(), l)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleUserListsDelete implements POST /api/users/lists/delete.
func (s *Server) handleUserListsDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID string `json:"listId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}
	if err := s.accounts.DeleteList(r.Context(), body.ListID, info.User.ID); err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListDelete})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUserListsFavorite implements POST /api/users/lists/favorite.
func (s *Server) handleUserListsFavorite(w http.ResponseWriter, r *http.Request) {
	s.serveListFavorite(w, r, true)
}

// handleUserListsUnfavorite implements POST /api/users/lists/unfavorite.
func (s *Server) handleUserListsUnfavorite(w http.ResponseWriter, r *http.Request) {
	s.serveListFavorite(w, r, false)
}

// serveListFavorite answers favourite and unfavourite. Both target a
// public list by id alone and answer NO_SUCH_USER_LIST (not NO_SUCH_LIST)
// when the list is missing or private.
func (s *Server) serveListFavorite(w http.ResponseWriter, r *http.Request, favorite bool) {
	var body struct {
		ListID string `json:"listId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}

	notFoundID := errIDNoSuchUserListUnfav
	if favorite {
		notFoundID = errIDNoSuchUserListFavorite
	}
	l, err := s.accounts.GetList(r.Context(), body.ListID, "", false)
	if err != nil || !l.IsPublic {
		writeAPIError(w, &APIError{
			Message: "No such user list.", Code: "NO_SUCH_USER_LIST",
			ID: notFoundID, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	if favorite {
		err = s.accounts.FavoriteList(r.Context(), info.User.ID, l.ID)
	} else {
		err = s.accounts.UnfavoriteList(r.Context(), info.User.ID, l.ID)
	}
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUserListsGetMemberships implements POST
// /api/users/lists/get-memberships.
func (s *Server) handleUserListsGetMemberships(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID    string `json:"listId"`
		ForPublic bool   `json:"forPublic"`
		Limit     int    `json:"limit"`
		SinceID   string `json:"sinceId"`
		UntilID   string `json:"untilId"`
		SinceDate *int64 `json:"sinceDate"`
		UntilDate *int64 `json:"untilDate"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}

	// Misskey's lookup rule: the owner reads their own list, everyone
	// else (including anonymous callers) reads public lists.
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	var l *account.UserList
	var err error
	if !body.ForPublic && viewerID != "" {
		l, err = s.accounts.GetList(r.Context(), body.ListID, viewerID, true)
	} else {
		l, err = s.accounts.GetList(r.Context(), body.ListID, "", false)
	}
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListMemberships})
		return
	}

	query := account.UserListMembershipQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	memberships, err := s.accounts.ListMemberships(r.Context(), l.ID, query)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(memberships))
	for _, m := range memberships {
		u, err := s.accounts.GetByID(r.Context(), m.UserID)
		if err != nil || u == nil {
			continue
		}
		out = append(out, map[string]any{
			"id":          m.ID,
			"createdAt":   m.CreatedAt.UTC().Format(time.RFC3339Nano),
			"userId":      m.UserID,
			"user":        packUserLite(u, "", loadProfile(r.Context(), s, m.UserID)),
			"withReplies": m.WithReplies,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUserListsList implements POST /api/users/lists/list.
func (s *Server) handleUserListsList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}

	var lists []*account.UserList
	var err error
	if body.UserID != "" {
		// Listing another account's lists shows public ones only, and
		// remote accounts have no local lists to show.
		target, targetErr := s.accounts.GetByID(r.Context(), body.UserID)
		if targetErr != nil || target == nil {
			writeAPIError(w, &APIError{
				Message: "No such user.", Code: "NO_SUCH_USER",
				ID: errIDNoSuchUserListsList, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		if target.Host != nil {
			writeAPIError(w, &APIError{
				Message: "Remote user not allowed.", Code: "REMOTE_USER_NOT_ALLOWED",
				ID: errIDRemoteUserNotAllowed, Kind: "permission", Status: http.StatusForbidden,
			})
			return
		}
		lists, err = s.accounts.ListListsOf(r.Context(), target.ID, true)
	} else {
		if viewerID == "" {
			writeAPIError(w, invalidParamError("userId is required"))
			return
		}
		lists, err = s.accounts.ListListsOf(r.Context(), viewerID, false)
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(lists))
	for _, l := range lists {
		packed, err := s.packUserList(r.Context(), l)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUserListsShow implements POST /api/users/lists/show.
func (s *Server) handleUserListsShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID    string `json:"listId"`
		ForPublic bool   `json:"forPublic"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}

	var l *account.UserList
	var err error
	if !body.ForPublic && viewerID != "" {
		l, err = s.accounts.GetList(r.Context(), body.ListID, viewerID, true)
	} else {
		l, err = s.accounts.GetList(r.Context(), body.ListID, "", false)
	}
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListMemberships})
		return
	}

	packed, err := s.packUserList(r.Context(), l)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	// likedCount/isLiked only travel on the forPublic branch, as upstream
	// packs them; isLiked is false for an anonymous viewer.
	if body.ForPublic && l.IsPublic {
		count, err := s.accounts.CountListFavorites(r.Context(), l.ID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		liked := false
		if viewerID != "" {
			liked, err = s.accounts.HasListFavorite(r.Context(), viewerID, l.ID)
			if err != nil {
				writeInternalServerError(w)
				return
			}
		}
		packed["likedCount"] = count
		packed["isLiked"] = liked
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleUserListsUpdate implements POST /api/users/lists/update.
func (s *Server) handleUserListsUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID   string  `json:"listId"`
		Name     *string `json:"name"`
		IsPublic *bool   `json:"isPublic"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}
	l, err := s.accounts.UpdateList(r.Context(), body.ListID, info.User.ID, body.Name, body.IsPublic)
	if err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListUpdate})
		return
	}
	// Upstream re-packs after the update, re-reading the membership ids.
	packed, err := s.packUserList(r.Context(), l)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleUserListsPush implements POST /api/users/lists/push.
func (s *Server) handleUserListsPush(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID string `json:"listId"`
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	if !s.checkRateLimit(w, info.User.ID, limitListsPush) {
		return
	}
	if body.ListID == "" || body.UserID == "" {
		writeAPIError(w, invalidParamError("listId and userId are required"))
		return
	}

	ids := userListErrorIDs{noSuchList: errIDNoSuchListPush, tooManyUsers: errIDTooManyUsersPush, alreadyAdded: errIDAlreadyAddedPush, blocked: errIDYouHaveBeenBlockedPush}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserPush, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	blocked, err := s.accounts.IsBlocked(r.Context(), target.ID, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if blocked {
		writeUserListError(w, account.ErrMemberBlocksYou, ids)
		return
	}
	// New memberships follow the upstream default: replies excluded until
	// update-membership says otherwise.
	if err := s.accounts.AddMember(r.Context(), body.ListID, info.User.ID, target.ID, false); err != nil {
		writeUserListError(w, err, ids)
		return
	}
	s.publishListMemberEvent(r.Context(), body.ListID, target.ID, "userAdded")
	w.WriteHeader(http.StatusNoContent)
}

// handleUserListsPull implements POST /api/users/lists/pull. Pulling a
// non-member silently succeeds.
func (s *Server) handleUserListsPull(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID string `json:"listId"`
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	if body.ListID == "" || body.UserID == "" {
		writeAPIError(w, invalidParamError("listId and userId are required"))
		return
	}

	// Resolve first so pull of a non-user still answers NO_SUCH_USER, as
	// upstream does; then the silent remove.
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserPull, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if err := s.accounts.PullMember(r.Context(), body.ListID, info.User.ID, target.ID); err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListPull})
		return
	}
	s.publishListMemberEvent(r.Context(), body.ListID, target.ID, "userRemoved")
	w.WriteHeader(http.StatusNoContent)
}

// handleUserListsUpdateMembership implements POST
// /api/users/lists/update-membership.
func (s *Server) handleUserListsUpdateMembership(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ListID      string `json:"listId"`
		UserID      string `json:"userId"`
		WithReplies *bool  `json:"withReplies"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireListCaller(w, r)
	if info == nil {
		return
	}
	if info.User.MovedToURI != "" {
		writeUserListError(w, account.ErrAccountMoved, userListErrorIDs{})
		return
	}
	if body.ListID == "" || body.UserID == "" {
		writeAPIError(w, invalidParamError("listId and userId are required"))
		return
	}
	if err := s.accounts.UpdateMembership(r.Context(), body.ListID, info.User.ID, body.UserID, body.WithReplies); err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListPull})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUserListTimeline implements POST /api/notes/user-list-timeline.
func (s *Server) handleUserListTimeline(w http.ResponseWriter, r *http.Request) {
	if s.timeline == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		ListID                string `json:"listId"`
		Limit                 int    `json:"limit"`
		SinceID               string `json:"sinceId"`
		UntilID               string `json:"untilId"`
		SinceDate             *int64 `json:"sinceDate"`
		UntilDate             *int64 `json:"untilDate"`
		WithFiles             bool   `json:"withFiles"`
		WithRenotes           *bool  `json:"withRenotes"`
		IncludeMyRenotes      *bool  `json:"includeMyRenotes"`
		IncludeRenotedMyNotes *bool  `json:"includeRenotedMyNotes"`
		IncludeLocalRenotes   *bool  `json:"includeLocalRenotes"`
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
	if body.ListID == "" {
		writeAPIError(w, invalidParamError("listId is required"))
		return
	}
	// The list timeline is always the caller's own list, never public.
	if _, err := s.accounts.OwnList(r.Context(), body.ListID, info.User.ID); err != nil {
		writeUserListError(w, err, userListErrorIDs{noSuchList: errIDNoSuchListListTimeline})
		return
	}

	params := note.TimelineParams{
		Kind:                  note.TimelineList,
		ListID:                body.ListID,
		ViewerID:              info.User.ID,
		Limit:                 body.Limit,
		SinceID:               body.SinceID,
		UntilID:               body.UntilID,
		WithFiles:             body.WithFiles,
		WithRenotes:           true,
		IncludeMyRenotes:      true,
		IncludeRenotedMyNotes: true,
		IncludeLocalRenotes:   true,
	}
	if body.WithRenotes != nil {
		params.WithRenotes = *body.WithRenotes
	}
	if body.IncludeMyRenotes != nil {
		params.IncludeMyRenotes = *body.IncludeMyRenotes
	}
	if body.IncludeRenotedMyNotes != nil {
		params.IncludeRenotedMyNotes = *body.IncludeRenotedMyNotes
	}
	if body.IncludeLocalRenotes != nil {
		params.IncludeLocalRenotes = *body.IncludeLocalRenotes
	}
	if params.SinceID == "" && body.SinceDate != nil {
		id, err := s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.SinceDate))
		if err != nil {
			writeInternalServerError(w)
			return
		}
		params.SinceID = id
	}
	if params.UntilID == "" && body.UntilDate != nil {
		id, err := s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.UntilDate))
		if err != nil {
			writeInternalServerError(w)
			return
		}
		params.UntilID = id
	}

	notes, err := s.timeline.Timeline(r.Context(), params)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, info.User.ID)
		out = append(out, packNoteDetail(r.Context(), s, n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}
