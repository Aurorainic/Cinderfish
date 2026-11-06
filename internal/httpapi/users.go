package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for the follower and following list endpoints, copied
// from packages/backend/src/server/api/endpoints/users/{followers,following}.ts.
const (
	errIDNoSuchUserFollowers = "27fa5435-88ab-43de-9360-387de88727cd"
	errIDForbiddenFollowers  = "3c6a84db-d619-26af-ca14-06232a21df8a"
	errIDNoSuchUserFollowing = "63e4aba4-4156-4e53-be25-c9559e42d71b"
	errIDForbiddenFollowing  = "f6cdb0df-c19f-ec5c-7dbb-0ba84a1f92ba"
)

// handleUsersNotes implements POST /api/users/notes: the public and home
// notes of one local user, newest first. Followers-only notes are not
// included, matching the endpoint's anonymous scope.
func (s *Server) handleUsersNotes(w http.ResponseWriter, r *http.Request) {
	if s.timeline == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		UserID      string `json:"userId"`
		Username    string `json:"username"`
		Limit       int    `json:"limit"`
		SinceID     string `json:"sinceId"`
		UntilID     string `json:"untilId"`
		WithFiles   *bool  `json:"withFiles"`
		WithReplies *bool  `json:"withReplies"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	var (
		user *account.User
		err  error
	)
	switch {
	case body.UserID != "":
		user, err = s.accounts.GetByID(r.Context(), body.UserID)
	case body.Username != "":
		user, err = s.accounts.GetByUsername(r.Context(), body.Username)
	default:
		writeAPIError(w, invalidParamError("userId or username is required"))
		return
	}
	if err != nil || user == nil {
		if err != nil && !errors.Is(err, account.ErrUserNotFound) {
			writeInternalServerError(w)
			return
		}
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	viewerID := ""
	if info, _ := s.authenticate(r.Context(), r, bodyAsMap(r)); info != nil {
		viewerID = info.User.ID
	}

	// Misskey's users/notes defaults: withFiles is false and withReplies
	// is true unless the request says otherwise.
	withFiles := body.WithFiles != nil && *body.WithFiles
	withReplies := body.WithReplies == nil || *body.WithReplies

	notes, err := s.timeline.UserNotes(r.Context(), user.ID, body.Limit, body.SinceID, body.UntilID, withFiles, withReplies)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNoteDetail(r.Context(), s, n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}

// followListRequest is the parameter set shared by users/followers and
// users/following.
type followListRequest struct {
	UserID  string `json:"userId"`
	Limit   int    `json:"limit"`
	SinceID string `json:"sinceId"`
	UntilID string `json:"untilId"`
}

// handleUsersFollowers implements POST /api/users/followers: the users
// who follow one account.
func (s *Server) handleUsersFollowers(w http.ResponseWriter, r *http.Request) {
	s.serveFollowList(w, r, true)
}

// handleUsersFollowing implements POST /api/users/following: the users
// one account follows.
func (s *Server) handleUsersFollowing(w http.ResponseWriter, r *http.Request) {
	s.serveFollowList(w, r, false)
}

// serveFollowList answers users/followers and users/following, which share
// their parameters and differ only in which side of the followings table
// they page.
//
// The response is Misskey's Following object, not a bare user list:
// Firefish's follow-list page reads x.follower / x.followee
// (packages/client/src/pages/user/follow-list.vue L11-13) and the
// json-schema/following.ts packed schema names the same fields. Each
// embedded user is packed as UserLite.
func (s *Server) serveFollowList(w http.ResponseWriter, r *http.Request, followers bool) {
	if s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body followListRequest
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	// Both endpoints are readable without a credential; the target's
	// privacy settings decide whether an anonymous viewer gets a list.
	var viewer *account.User
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewer = info.User
	}

	noSuchUserID, forbiddenID := errIDNoSuchUserFollowing, errIDForbiddenFollowing
	if followers {
		noSuchUserID, forbiddenID = errIDNoSuchUserFollowers, errIDForbiddenFollowers
	}

	// userId defaults to the caller, which is how a profile page asks for
	// its own lists.
	targetID := body.UserID
	if targetID == "" {
		if viewer == nil {
			writeAPIError(w, invalidParamError("userId is required"))
			return
		}
		targetID = viewer.ID
	}

	target, err := s.accounts.GetByID(r.Context(), targetID)
	if err != nil || target == nil {
		if err != nil && !errors.Is(err, account.ErrUserNotFound) {
			writeInternalServerError(w)
			return
		}
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: noSuchUserID, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	allowed, err := s.followListAllowed(r.Context(), target, viewer, followers)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	if !allowed {
		writeAPIError(w, &APIError{
			Message: "Forbidden.", Code: "FORBIDDEN",
			ID: forbiddenID, Kind: "permission", Status: http.StatusForbidden,
		})
		return
	}

	var list []*account.Follow
	if followers {
		list, err = s.accounts.Followers(r.Context(), target.ID, body.Limit, body.SinceID, body.UntilID)
	} else {
		list, err = s.accounts.Following(r.Context(), target.ID, body.Limit, body.SinceID, body.UntilID)
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]any, 0, len(list))
	for _, f := range list {
		followerID, followeeID := f.User.ID, target.ID
		if !followers {
			followerID, followeeID = target.ID, f.User.ID
		}
		item := map[string]any{
			"id":         f.ID,
			"createdAt":  f.CreatedAt.UTC().Format(time.RFC3339Nano),
			"followerId": followerID,
			"followeeId": followeeID,
		}
		packed := packUserLite(f.User, "", loadProfile(r.Context(), s, f.User.ID))
		if followers {
			item["follower"] = packed
		} else {
			item["followee"] = packed
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

// followListAllowed mirrors the visibility gate of Misskey's
// users/followers and users/following
// (packages/backend/src/server/api/endpoints/users/followers.ts
// L104-121): the owner reads their own lists, a "private" list belongs to
// the owner alone, a "followers" list to the owner and the owner's
// followers, and a moderator reads every list. Cinderfish's nearest
// moderator role is the administrator flag.
func (s *Server) followListAllowed(ctx context.Context, target, viewer *account.User, followers bool) (bool, error) {
	if viewer != nil && (viewer.ID == target.ID || viewer.IsAdmin) {
		return true, nil
	}

	followersVisibility, followingVisibility, err := s.accounts.FollowVisibility(ctx, target.ID)
	if err != nil {
		return false, err
	}
	visibility := followingVisibility
	if followers {
		visibility = followersVisibility
	}

	switch visibility {
	case "", "public":
		return true, nil
	case "private":
		return false, nil
	case "followers":
		if viewer == nil {
			return false, nil
		}
		return s.accounts.IsFollowing(ctx, viewer.ID, target.ID)
	default:
		return false, nil
	}
}

// applyPinnedNotes fills pinnedNoteIds and pinnedNotes with the user's
// pin list in stored order, at most five notes packed for viewerID.
func (s *Server) applyPinnedNotes(ctx context.Context, userID, viewerID string, packed map[string]any) {
	if s.accounts == nil || s.notes == nil {
		return
	}
	ids, err := s.accounts.PinnedNoteIDs(ctx, userID)
	if err != nil {
		return
	}
	// pinnedNoteIds and pinnedNotes must agree, so both are capped at
	// the same limit.
	if len(ids) > account.MaxPinnedNotes {
		ids = ids[:account.MaxPinnedNotes]
	}
	packed["pinnedNoteIds"] = ids
	notes := make([]any, 0, len(ids))
	for i, id := range ids {
		if i >= 5 {
			break
		}
		n, err := s.notes.Get(ctx, id)
		if err != nil {
			continue
		}
		notes = append(notes, packNoteDetail(ctx, s, n, s.notePackContext(ctx, n, viewerID)))
	}
	packed["pinnedNotes"] = notes
}

// applyUserCounts fills the profile counters that the UserDetailed schema
// always carries. The follows are counted live so they stay correct even
// for rows written before the counters existed.
func (s *Server) applyUserCounts(ctx context.Context, userID string, packed map[string]any) {
	if s.pool == nil {
		return
	}
	var followers, following int
	if err := s.pool.QueryRow(ctx,
		`SELECT followers_count, following_count FROM users WHERE id = $1`, userID).
		Scan(&followers, &following); err == nil {
		packed["followersCount"] = followers
		packed["followingCount"] = following
	}
	var notes int
	if err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM notes
WHERE user_id = $1 AND deleted_at IS NULL AND local_only = FALSE
  AND visibility IN ('public', 'home')`, userID).Scan(&notes); err == nil {
		packed["notesCount"] = notes
	}
}

// handleUsersStats implements POST /api/users/stats: the activity counters of
// one account.
//
// Upstream requires userId; this port also accepts an empty userId and answers
// for the caller, which is what the account-info page needs for itself. The
// endpoint is public: every number it returns is already on the profile.
func (s *Server) handleUsersStats(w http.ResponseWriter, r *http.Request) {
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

	userID := body.UserID
	if userID == "" {
		info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
		if err != nil || info == nil {
			writeAPIError(w, invalidParamError("userId is required"))
			return
		}
		userID = info.User.ID
	}

	user, err := s.accounts.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		writeAPIError(w, &APIError{
			Message: "No such user.", Code: "NO_SUCH_USER",
			ID: errIDNoSuchUser, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	stats, err := s.accounts.Stats(r.Context(), user.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notesCount":     stats.NotesCount,
		"repliesCount":   stats.RepliesCount,
		"renotesCount":   stats.RenotesCount,
		"repliedCount":   stats.RepliedCount,
		"renotedCount":   stats.RenotedCount,
		"pollVotesCount": stats.PollVotesCount,
		"pollVotedCount": stats.PollVotedCount,
	})
}
