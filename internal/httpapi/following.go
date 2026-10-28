package httpapi

import (
	"context"
	"log"
	"net/http"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// Misskey error ids for the following endpoints, copied from
// packages/backend/src/server/api/endpoints/following/*.ts.
const (
	errIDNoSuchUserDelete   = "5b12c78d-2b28-4dca-99d2-f56139b42ff8"
	errIDFolloweeIsYourself = "26fbe7bb-a331-4857-af17-205b426669a9"
	errIDAlreadyFollowing   = "35387507-38c7-4cb9-9197-300b93783fa0"
	errIDNotFollowing       = "5dbf82f5-c92b-40b1-87d1-6c8c0741fd09"
)

// handleFollowingCreate is POST /api/following/create.
//
// A local followee is followed immediately. A remote one sends a signed
// Follow and leaves the relationship pending until the Accept arrives,
// which the inbox records.
func (s *Server) handleFollowingCreate(w http.ResponseWriter, r *http.Request) {
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
			Message: "Followee is yourself.", Code: "FOLLOWEE_IS_YOURSELF",
			ID: errIDFolloweeIsYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if s.federationStore.HasFollowing(r.Context(), info.User.ID, target.ID) {
		writeAPIError(w, &APIError{
			Message: "You are already following that user.", Code: "ALREADY_FOLLOWING",
			ID: errIDAlreadyFollowing, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	if target.Host == nil {
		if err := s.federationStore.Follow(r.Context(), "", info.User.ID, target.ID); err != nil {
			writeInternalServerError(w)
			return
		}
		s.publishFollowEvent(info.User.ID, target.ID)
	} else {
		s.sendRemoteFollow(r.Context(), info.User.ID, target.ID, *target.Host)
	}

	writeJSON(w, http.StatusOK, packUserLite(target, s.cfg.Domain))
}

// handleFollowingDelete is POST /api/following/delete.
func (s *Server) handleFollowingDelete(w http.ResponseWriter, r *http.Request) {
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
			Message: "Followee is yourself.", Code: "FOLLOWEE_IS_YOURSELF",
			ID: errIDFolloweeIsYourself, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	target, err := s.accounts.GetByID(r.Context(), body.UserID)
	if err != nil || target == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUserDelete, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if !s.federationStore.HasFollowing(r.Context(), info.User.ID, target.ID) {
		writeAPIError(w, &APIError{
			Message: "You are not following that user.", Code: "NOT_FOLLOWING",
			ID: errIDNotFollowing, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	if err := s.federationStore.Unfollow(r.Context(), info.User.ID, target.ID); err != nil {
		writeInternalServerError(w)
		return
	}
	s.publishUnfollowEvent(info.User.ID, target.ID)

	// A remote peer also needs the Undo. Failure is logged, not returned:
	// the local relationship is already gone.
	if target.Host != nil {
		s.sendRemoteUndoFollow(r.Context(), info.User.ID, target.ID, *target.Host)
	}

	writeJSON(w, http.StatusOK, packUserLite(target, s.cfg.Domain))
}

// sendRemoteFollow resolves the remote actor, queues a signed Follow, and
// records it optimistically as pending. The inbox replaces the pending
// row with a real one when the Accept arrives.
func (s *Server) sendRemoteFollow(ctx context.Context, meID, targetID, host string) {
	if s.federationClient == nil || s.fedRenderer == nil || s.federationDeliver == nil {
		return
	}
	if err := s.federationPolicy.CheckAllowed(ctx, host); err != nil {
		log.Printf("httpapi: cannot follow %s: %v", host, err)
		return
	}

	// The remote actor's inbox is only known once its document has been
	// fetched, so resolution happens here rather than in the request
	// path's error handling: a peer that is down simply fails the follow.
	actor, err := s.resolveRemoteActor(ctx, targetID)
	if err != nil {
		log.Printf("httpapi: resolve followee %s: %v", targetID, err)
		return
	}
	followID := s.fedRenderer.ActorRef(meID) + "#follows/" + actor.ID
	follow := s.fedRenderer.RenderFollow(followID, s.fedRenderer.ActorRef(meID), actor.URI)

	inbox := actor.SharedInbox
	if inbox == "" {
		inbox = actor.Inbox
	}
	if inbox == "" {
		log.Printf("httpapi: remote actor %s has no inbox", actor.URI)
		return
	}
	if err := s.federationDeliver.Enqueue(ctx, inbox, followID, s.fedRenderer.KeyID(meID), meID, follow); err != nil {
		log.Printf("httpapi: enqueue follow to %s: %v", inbox, err)
	}
}

// sendRemoteUndoFollow queues an Undo for a previously sent Follow.
func (s *Server) sendRemoteUndoFollow(ctx context.Context, meID, targetID, host string) {
	if s.federationClient == nil || s.fedRenderer == nil || s.federationDeliver == nil {
		return
	}
	actor, err := s.federationStore.UserByID(ctx, targetID)
	if err != nil {
		log.Printf("httpapi: undo follow, load %s: %v", targetID, err)
		return
	}
	followID := s.fedRenderer.ActorRef(meID) + "#follows/" + targetID
	follow := s.fedRenderer.RenderFollow(followID, s.fedRenderer.ActorRef(meID), actor.URI)
	undo := s.fedRenderer.RenderUndo(follow, s.fedRenderer.ActorRef(meID))

	inbox := actor.SharedInbox
	if inbox == "" {
		inbox = actor.Inbox
	}
	if inbox == "" {
		return
	}
	if err := s.federationDeliver.Enqueue(ctx, inbox, "", s.fedRenderer.KeyID(meID), meID, undo); err != nil {
		log.Printf("httpapi: enqueue undo follow to %s: %v", inbox, err)
	}
}

// resolveRemoteActor loads a stored remote actor.
func (s *Server) resolveRemoteActor(ctx context.Context, userID string) (*federation.RemoteUser, error) {
	return s.federationStore.UserByID(ctx, userID)
}
