package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/federation"
)

// Misskey error ids for the follow request endpoints, copied from
// packages/backend/src/server/api/endpoints/following/requests/*.ts.
const (
	errIDNoSuchUserAccept      = "66ce1645-d66c-46bb-8b79-96739af885bd"
	errIDNoFollowRequest       = "bcde4f8b-0913-4614-8881-614e522fb041"
	errIDNoSuchUserReject      = "abc2ffa6-25b2-4380-ba99-321ff3a94555"
	errIDNoSuchUserCancel      = "4e68c551-fc4c-4e46-bb41-7d4a37bf9dab"
	errIDFollowRequestNotFound = "089b125b-d338-482a-9a09-e2622ac9f8d4"
)

// followRequestParams is the parameter set shared by
// following/requests/list and following/requests/sent. Misskey types the
// date cursors as integer milliseconds.
type followRequestParams struct {
	Limit     int    `json:"limit"`
	SinceID   string `json:"sinceId"`
	UntilID   string `json:"untilId"`
	SinceDate *int64 `json:"sinceDate"`
	UntilDate *int64 `json:"untilDate"`
}

// handleFollowingRequestsList is POST /api/following/requests/list: the
// pending requests other users sent to the caller.
func (s *Server) handleFollowingRequestsList(w http.ResponseWriter, r *http.Request) {
	s.serveFollowRequests(w, r, true)
}

// handleFollowingRequestsSent is POST /api/following/requests/sent: the
// pending requests the caller sent out.
func (s *Server) handleFollowingRequestsSent(w http.ResponseWriter, r *http.Request) {
	s.serveFollowRequests(w, r, false)
}

// serveFollowRequests answers the two request lists. They share their
// parameters and differ only in which side of the row has to be the
// caller.
func (s *Server) serveFollowRequests(w http.ResponseWriter, r *http.Request, incoming bool) {
	var body followRequestParams
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if s.federationStore == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	query := federation.FollowRequestQuery{
		Limit:   body.Limit,
		SinceID: body.SinceID,
		UntilID: body.UntilID,
	}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	if incoming {
		query.FolloweeID = info.User.ID
	} else {
		query.FollowerID = info.User.ID
	}

	requests, err := s.federationStore.FollowRequests(r.Context(), query)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(requests))
	for _, req := range requests {
		item, err := s.packFollowRequest(r.Context(), req)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

// packFollowRequest renders one row the way Misskey's
// FollowRequestEntityService does (packages/backend/src/core/entities/
// FollowRequestEntityService.ts): id, follower and followee, both packed
// as UserLite, and nothing else — the row carries no createdAt.
func (s *Server) packFollowRequest(ctx context.Context, req *federation.FollowRequest) (map[string]any, error) {
	follower, err := s.accounts.GetByID(ctx, req.FollowerID)
	if err != nil {
		return nil, err
	}
	followee, err := s.accounts.GetByID(ctx, req.FolloweeID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":       req.ID,
		"follower": packUserLite(follower, "", loadProfile(ctx, s, follower.ID)),
		"followee": packUserLite(followee, "", loadProfile(ctx, s, followee.ID)),
	}, nil
}

// handleFollowingRequestsAccept is POST /api/following/requests/accept:
// the caller approves a request they received, which turns it into a
// following. A remote follower is answered with an ActivityPub Accept.
func (s *Server) handleFollowingRequestsAccept(w http.ResponseWriter, r *http.Request) {
	follower, info, ok := s.followRequestTarget(w, r, errIDNoSuchUserAccept)
	if !ok {
		return
	}

	req, err := s.federationStore.AcceptFollowRequest(r.Context(), follower.ID, info.User.ID)
	if errors.Is(err, federation.ErrRemoteNotFound) {
		writeAPIError(w, &APIError{
			Message: "No follow request.", Code: "NO_FOLLOW_REQUEST",
			ID: errIDNoFollowRequest, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	s.publishFollowEvent(follower.ID, info.User.ID)
	s.deliverFollowRequestAnswer(r.Context(), true, info.User.ID, req)
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleFollowingRequestsReject is POST /api/following/requests/reject:
// the caller drops a request they received. A remote follower is answered
// with an ActivityPub Reject.
func (s *Server) handleFollowingRequestsReject(w http.ResponseWriter, r *http.Request) {
	follower, info, ok := s.followRequestTarget(w, r, errIDNoSuchUserReject)
	if !ok {
		return
	}

	req, err := s.federationStore.RemoveFollowRequest(r.Context(), follower.ID, info.User.ID)
	if err != nil && !errors.Is(err, federation.ErrRemoteNotFound) {
		writeInternalServerError(w)
		return
	}
	// Misskey treats a missing request as a no-op here
	// (UserFollowingService.rejectFollowRequest ->
	// removeFollowRequest returns silently), so a stale button still
	// succeeds and no Reject is sent for a request that was already gone.
	if req != nil {
		s.deliverFollowRequestAnswer(r.Context(), false, info.User.ID, req)
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleFollowingRequestsCancel is POST /api/following/requests/cancel:
// the caller withdraws a request they sent. Misskey answers with the
// followee packed as UserLite.
func (s *Server) handleFollowingRequestsCancel(w http.ResponseWriter, r *http.Request) {
	followee, info, ok := s.followRequestTarget(w, r, errIDNoSuchUserCancel)
	if !ok {
		return
	}

	_, err := s.federationStore.RemoveFollowRequest(r.Context(), info.User.ID, followee.ID)
	if errors.Is(err, federation.ErrRemoteNotFound) {
		writeAPIError(w, &APIError{
			Message: "Follow request not found.", Code: "FOLLOW_REQUEST_NOT_FOUND",
			ID: errIDFollowRequestNotFound, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	// A remote followee still holds the Follow, so the Undo follows the
	// local delete. Failure is logged, not returned: the request is
	// already gone on this side.
	if followee.Host != nil {
		s.sendRemoteUndoFollow(r.Context(), info.User.ID, followee.ID, *followee.Host)
	}
	writeJSON(w, http.StatusOK, packUserLite(followee, "", loadProfile(r.Context(), s, followee.ID)))
}

// followRequestTarget decodes and authenticates one of the
// following/requests writes and resolves the userId it names. It answers
// the error responses itself and reports ok=false once it has; noSuchUserID
// is the calling endpoint's own Misskey error id.
func (s *Server) followRequestTarget(w http.ResponseWriter, r *http.Request, noSuchUserID string) (*account.User, *AuthInfo, bool) {
	var body struct {
		UserID string `json:"userId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return nil, nil, false
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil, nil, false
	}
	if s.accounts == nil || s.federationStore == nil {
		writeInternalServerError(w)
		return nil, nil, false
	}
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return nil, nil, false
	}

	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		if err != nil && !errors.Is(err, account.ErrUserNotFound) {
			writeInternalServerError(w)
			return nil, nil, false
		}
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: noSuchUserID, Kind: "client", Status: http.StatusNotFound,
		})
		return nil, nil, false
	}
	return target, info, true
}

// deliverFollowRequestAnswer sends an Accept or Reject for a request a
// remote actor sent. Delivery goes through the same queue as every other
// outbound activity, and a failure is logged rather than returned: the
// local relationship already changed, which is what inbox.follow does
// when its own Accept cannot be delivered.
func (s *Server) deliverFollowRequestAnswer(ctx context.Context, accept bool, meID string, req *federation.FollowRequest) {
	if s.federationDeliver == nil || s.fedRenderer == nil || s.federationStore == nil {
		return
	}

	actor, err := s.federationStore.UserByID(ctx, req.FollowerID)
	if err != nil {
		log.Printf("httpapi: follow request %s: load follower %s: %v", req.ID, req.FollowerID, err)
		return
	}
	// A local follower needs no answer: it learns the outcome from its
	// own requests/sent list.
	if actor.Host == "" {
		return
	}
	inbox := actor.SharedInbox
	if inbox == "" {
		inbox = actor.Inbox
	}
	if actor.URI == "" || inbox == "" {
		log.Printf("httpapi: follow request %s: remote follower %s has no actor id or inbox", req.ID, req.FollowerID)
		return
	}

	// The Accept/Reject has to embed the original Follow, so the peer can
	// match the answer to its request. The stored activity id is the one
	// it sent; a row without one falls back to the id we would have used.
	followID := req.URI
	if followID == "" {
		followID = s.fedRenderer.ActorRef(actor.ID) + "#follows/" + meID
	}
	follow := s.fedRenderer.RenderFollow(followID, actor.URI, s.fedRenderer.ActorRef(meID))

	activity := s.fedRenderer.RenderReject(s.fedRenderer.ActorRef(meID), follow)
	if accept {
		activity = s.fedRenderer.RenderAccept(s.fedRenderer.ActorRef(meID), follow)
	}
	if err := s.federationDeliver.Enqueue(ctx, inbox, activity.Ref("id"), s.fedRenderer.KeyID(meID), meID, activity); err != nil {
		log.Printf("httpapi: enqueue %s for follow request %s to %s: %v", activity.Type(), req.ID, inbox, err)
	}
}
