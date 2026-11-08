package httpapi

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/federation"
)

// Misskey error ids for the account migration endpoints, copied from
// packages/backend/src/server/api/endpoints/i/{known-as,move}.ts.
const (
	errIDMigrationNoSuchUser     = "fcd2eef9-a9b2-4c4f-8624-038099e90aa5"
	errIDMigrationNotRemote      = "4362f8dc-731f-4ad8-a694-be2a88922a24"
	errIDMigrationURINull        = "bf326f31-d430-4f97-9933-5d61e4d48a23"
	errIDMigrationLocalURINull   = "95ba11b9-90e8-43a5-ba16-7acc1ab32e71"
	errIDMigrationAlreadyMoved   = "b234a14e-9ebe-4581-8000-074b3c215962"
	errIDMigrationAlreadyAliased = "56f20ec9-fd06-4fa5-841b-edd6d7d4fa31"
	errIDMigrationYourself       = "25c90186-4ab0-49c8-9bba-a1fa6c202ba4"
	errIDMigrationNoMoveTarget   = "b5c90186-4ab0-49c8-9bba-a1f76c202ba4"
	errIDMigrationTargetForbids  = "b5c90186-4ab0-49c8-9bba-a1f766282ba4"
	errIDMigrationAdminForbidden = "4362e8dc-731f-4ad8-a694-be2a88922a24"
)

// handleIKnownAs implements POST /api/i/known-as: it records the actor URIs
// the caller declares as aliases of itself.
//
// The list is what the target of a Move advertises, so a peer can check that
// the moving account really is the one this account names. Only remote
// accounts, or a local one by username, may be named; the account itself is
// refused. Misskey accepts at most ten entries.
func (s *Server) handleIKnownAs(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil || s.fedRenderer == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		AlsoKnownAs []string `json:"alsoKnownAs"`
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
	if len(body.AlsoKnownAs) > 10 {
		writeAPIError(w, invalidParamError("alsoKnownAs accepts at most 10 entries"))
		return
	}
	if info.User.MovedToURI != "" {
		writeAPIError(w, &APIError{
			Message: "You have already moved your account.", Code: "ALREADY_MOVED",
			ID: errIDMigrationAlreadyAliased, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	uris := make([]string, 0, len(body.AlsoKnownAs))
	seen := map[string]bool{}
	for _, line := range body.AlsoKnownAs {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "@"))
		if line == "" {
			writeAPIError(w, s.migrationNoSuchUser())
			return
		}
		uri, err := s.resolveAlias(r.Context(), line)
		if err != nil {
			writeAPIError(w, s.migrationNoSuchUser())
			return
		}
		if self := s.fedRenderer.ActorRef(info.User.ID); uri == self {
			writeAPIError(w, &APIError{
				Message: "You can't set yourself as your own alias.", Code: "FORBIDDEN_TO_SET_YOURSELF",
				ID: errIDMigrationYourself, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		if seen[uri] {
			continue
		}
		seen[uri] = true
		uris = append(uris, uri)
	}

	if err := s.accounts.SetKnownAs(r.Context(), info.User.ID, uris); err != nil {
		writeInternalServerError(w)
		return
	}
	updated, err := s.accounts.GetByID(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writeMe(w, r, updated, info.Token == nil)
}

// resolveAlias turns one alias line into an actor URI. A local username
// resolves from the database; a user@host handle goes through WebFinger and
// the actor document.
func (s *Server) resolveAlias(ctx context.Context, line string) (string, error) {
	user, host := line, ""
	if i := strings.Index(line, "@"); i >= 0 {
		user, host = line[:i], strings.ToLower(line[i+1:])
	}
	if host == "" || (s.cfg != nil && strings.EqualFold(host, s.cfg.Domain)) {
		u, err := s.accounts.GetByUsername(ctx, user)
		if err != nil || u == nil {
			return "", account.ErrUserNotFound
		}
		return s.fedRenderer.ActorRef(u.ID), nil
	}
	if s.federationInbox == nil {
		return "", federation.ErrRemoteNotFound
	}
	remote, err := s.federationInbox.ResolveAcct(ctx, user, host)
	if err != nil {
		return "", err
	}
	if remote.URI == "" {
		return "", federation.ErrRemoteNotFound
	}
	return remote.URI, nil
}

// migrationNoSuchUser is the NO_SUCH_USER envelope the alias resolver returns.
func (s *Server) migrationNoSuchUser() *APIError {
	return &APIError{
		Message: "No such user.", Code: "NO_SUCH_USER",
		ID: errIDMigrationNoSuchUser, Kind: "client", Status: http.StatusNotFound,
	}
}

// handleIMove implements POST /api/i/move: the caller migrates to a remote
// account that already lists the caller's actor URI in its alsoKnownAs.
//
// The move is recorded locally, announced to the caller's followers as a
// Move activity, and the local followers are moved across to the target, so
// no one is left following an account that no longer posts here.
func (s *Server) handleIMove(w http.ResponseWriter, r *http.Request) {
	if s.accounts == nil || s.federationStore == nil || s.federationInbox == nil || s.fedRenderer == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		MoveToAccount string `json:"moveToAccount"`
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
	if info.User.IsAdmin {
		writeAPIError(w, &APIError{
			Message: "Admins cant migrate.", Code: "NOT_ADMIN_FORBIDDEN",
			ID: errIDMigrationAdminForbidden, Kind: "client", Status: http.StatusForbidden,
		})
		return
	}
	if info.User.MovedToURI != "" {
		writeAPIError(w, &APIError{
			Message: "Account was already moved to another account.", Code: "ALREADY_MOVED",
			ID: errIDMigrationAlreadyMoved, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body.MoveToAccount), "@"))
	user, host, ok := federation.SplitAcct(line)
	if !ok {
		// A bare username is a local account, and Firefish refuses those
		// separately from an unparsable handle.
		if strings.Contains(line, "@") || line == "" {
			writeAPIError(w, &APIError{
				Message: "No such move target.", Code: "NO_SUCH_MOVE_TARGET",
				ID: errIDMigrationNoMoveTarget, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeAPIError(w, &APIError{
			Message: "User is not remote. You can only migrate to other instances.", Code: "NOT_REMOTE",
			ID: errIDMigrationNotRemote, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	target, err := s.federationInbox.ResolveAcct(r.Context(), user, host)
	if err != nil {
		log.Printf("httpapi: i/move resolve %s: %v", line, err)
		writeAPIError(w, &APIError{
			Message: "No such move target.", Code: "NO_SUCH_MOVE_TARGET",
			ID: errIDMigrationNoMoveTarget, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if target.URI == "" {
		writeAPIError(w, &APIError{
			Message: "User ActivityPup URI is null.", Code: "URI_NULL",
			ID: errIDMigrationURINull, Kind: "client", Status: http.StatusInternalServerError,
		})
		return
	}

	// The target must name this account among its aliases. Re-fetch so a
	// list that was just changed is what gets checked.
	fresh, err := s.federationInbox.FetchActor(r.Context(), target.URI)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	selfURI := s.fedRenderer.ActorRef(info.User.ID)
	allowed := false
	for _, alias := range fresh.AlsoKnownAs {
		if alias == selfURI || alias == info.User.ID {
			allowed = true
			break
		}
	}
	if !allowed {
		writeAPIError(w, &APIError{
			Message: "Remote account doesn't have proper 'Known As' alias. Did you remember to set it?",
			Code:    "REMOTE_ACCOUNT_FORBIDS",
			ID:      errIDMigrationTargetForbids, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	if err := s.accounts.MarkMoved(r.Context(), info.User.ID, target.URI); err != nil {
		writeInternalServerError(w)
		return
	}

	// Announce the move to everyone who follows the old account.
	activity := s.fedRenderer.RenderMove(
		s.fedRenderer.ActivityRef(info.User.ID)+"/move",
		selfURI, selfURI, target.URI,
	)
	if inboxes, err := s.federationStore.InboxesForUser(r.Context(), info.User.ID, false); err == nil && s.federationDeliver != nil {
		seen := map[string]bool{}
		for _, inbox := range inboxes {
			if seen[inbox] {
				continue
			}
			seen[inbox] = true
			if err := s.federationDeliver.Enqueue(r.Context(), inbox, "", s.fedRenderer.KeyID(info.User.ID), info.User.ID, activity); err != nil {
				log.Printf("httpapi: enqueue move to %s: %v", inbox, err)
			}
		}
	} else if err != nil {
		log.Printf("httpapi: follower inboxes for move: %v", err)
	}

	s.migrateFollowersOf(r.Context(), info.User.ID, target.ID)

	// The response carries the updated account so the client can drop the
	// migration form without a second round trip.
	updated, err := s.accounts.GetByID(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writeMe(w, r, updated, info.Token == nil)
}

// writeMe answers with the packed MeDetailed entity, applying the derived
// counters the i endpoints fill.
func (s *Server) writeMe(w http.ResponseWriter, r *http.Request, u *account.User, includeSecrets bool) {
	writeJSON(w, http.StatusOK, s.meDetailed(r.Context(), u, includeSecrets))
}

// meDetailed packs a user as MeDetailed, with the counters that only the
// owner's own view carries. Every i endpoint and the meUpdated stream event
// go through it so the two can never drift apart.
func (s *Server) meDetailed(ctx context.Context, u *account.User, includeSecrets bool) map[string]any {
	packed := packMeDetailed(u, u.CreatedAt, includeSecrets, loadProfile(ctx, s, u.ID))
	s.applyPinnedPage(ctx, u.ID, packed)
	s.packMovedTo(ctx, u, packed)
	s.applyUserCounts(ctx, u.ID, packed)
	s.applyUnreadNotifications(ctx, u.ID, packed)
	s.applyUnreadAnnouncements(ctx, u.ID, packed)
	s.applyPinnedNotes(ctx, u.ID, u.ID, packed)
	// The Firefish client reads the legacy field name; hasUnreadChatMessages
	// stays false for clients keyed to the newer Misskey field.
	if s.accounts != nil {
		if hasUnread, err := s.accounts.HasUnreadMessages(ctx, u.ID); err == nil {
			packed["hasUnreadMessagingMessage"] = hasUnread
		}
	} else {
		packed["hasUnreadMessagingMessage"] = false
	}
	return packed
}

// migrateFollowersOf moves the local followers of oldID to newID. It is the
// same step Firefish performs both when a local account moves away and when
// an inbound Move arrives for a remote account: unfollow the old account,
// then follow the new one (through the outbound queue when it is remote).
func (s *Server) migrateFollowersOf(ctx context.Context, oldID, newID string) {
	if s.federationStore == nil || s.accounts == nil || oldID == newID {
		return
	}
	followers, err := s.federationStore.LocalFollowerIDs(ctx, oldID)
	if err != nil {
		log.Printf("httpapi: local followers of %s: %v", oldID, err)
		return
	}
	if len(followers) == 0 {
		return
	}
	target, err := s.accounts.GetByID(ctx, newID)
	if err != nil || target == nil {
		log.Printf("httpapi: move target %s: %v", newID, err)
		return
	}
	for _, followerID := range followers {
		if err := s.federationStore.Unfollow(ctx, followerID, oldID); err != nil {
			log.Printf("httpapi: unfollow %s from %s: %v", followerID, oldID, err)
			continue
		}
		s.publishUnfollowEvent(followerID, oldID)
		if target.Host == nil {
			if err := s.federationStore.Follow(ctx, "", followerID, newID); err != nil {
				log.Printf("httpapi: follow %s on %s: %v", followerID, newID, err)
				continue
			}
			s.publishFollowEvent(followerID, newID)
			continue
		}
		s.sendRemoteFollow(ctx, followerID, newID, *target.Host)
	}
}
