package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// Misskey error ids for the gallery endpoints, copied from
// packages/backend/src/server/api/endpoints/gallery/*.ts.
const (
	errIDNoSuchGalleryPostDelete  = "ae52f367-4bd7-4ecd-afc6-5672fff427f5"
	errIDNoSuchGalleryPostShow    = "1137bf14-c5b0-4604-85bb-5b5371b1cd45"
	errIDNoSuchGalleryPostLike    = "56c06af3-1287-442f-9701-c93f7c4a62ff"
	errIDGalleryYourPost          = "f78f1511-5ebc-4478-a888-1198d752da68"
	errIDGalleryAlreadyLiked      = "40e9ed56-a59c-473a-bf3f-f289c54fb5a7"
	errIDNoSuchGalleryPostUnlike  = "c32e6dd0-b555-4413-925e-b3757d19ed84"
	errIDGalleryNotLiked          = "e3e8e06e-be37-41f7-a5b4-87a8250288f0"
)

// channelErrorIDs-style carrier for the gallery family.
type galleryErrorIDs struct {
	noSuchPost string
}

// limitGalleryCreate is Misskey's gallery/posts/create rate cap.
var limitGalleryCreate = rateLimit{bucket: "gallery/posts/create", max: 20, duration: time.Hour}

// packGalleryPost renders the GalleryPost schema. tags is dead data
// upstream never writes, so it stays absent; isLiked travels only with a
// viewer; files arrive in fileIds order via the ordered drive lookup.
func (s *Server) packGalleryPost(ctx context.Context, p *account.GalleryPost, viewerID string) (map[string]any, error) {
	packed := map[string]any{
		"id":          p.ID,
		"createdAt":   p.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":   nilOrTime(p.UpdatedAt),
		"userId":      p.UserID,
		"title":       p.Title,
		"description": p.Description,
		"fileIds":     p.FileIDs,
		"isSensitive": p.IsSensitive,
		"likedCount":  p.LikedCount,
	}
	files, err := s.orderedDriveFiles(ctx, p.UserID, p.FileIDs)
	if err != nil {
		return nil, err
	}
	fileList := make([]any, 0, len(files))
	for _, f := range files {
		fileList = append(fileList, packDriveFile(f))
	}
	packed["files"] = fileList
	if u, err := s.accounts.GetByID(ctx, p.UserID); err == nil && u != nil {
		packed["user"] = packUserLite(u, "", loadProfile(ctx, s, p.UserID))
	}
	if viewerID != "" {
		liked, err := s.accounts.HasGalleryLike(ctx, viewerID, p.ID)
		if err != nil {
			return nil, err
		}
		packed["isLiked"] = liked
	}
	return packed, nil
}

// orderedDriveFiles loads drive files in the requested id order, skipping
// ids that no longer resolve. DriveService.GetMany does not preserve
// order, hence the explicit resequencing.
func (s *Server) orderedDriveFiles(ctx context.Context, ownerID string, ids []string) ([]*media.File, error) {
	if len(ids) == 0 || s.drive == nil {
		return []*media.File{}, nil
	}
	files, err := s.drive.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*media.File, len(files))
	for _, f := range files {
		byID[f.ID] = f
	}
	out := make([]*media.File, 0, len(ids))
	for _, id := range ids {
		if f := byID[id]; f != nil && f.UserID == ownerID {
			out = append(out, f)
		}
	}
	return out, nil
}

// writeGalleryError maps the gallery service sentinels onto Misskey's
// envelope with the verbatim ids.
func writeGalleryError(w http.ResponseWriter, err error, ids galleryErrorIDs) {
	switch {
	case errors.Is(err, account.ErrGalleryPostNotFound):
		writeAPIError(w, &APIError{
			Message: "No such post.", Code: "NO_SUCH_POST",
			ID: ids.noSuchPost, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrGalleryYourPost):
		writeAPIError(w, &APIError{
			Message: "You cannot like your own post.", Code: "YOUR_POST",
			ID: errIDGalleryYourPost, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrGalleryAlreadyLiked):
		writeAPIError(w, &APIError{
			Message: "The post has already been liked.", Code: "ALREADY_LIKED",
			ID: errIDGalleryAlreadyLiked, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrGalleryNotLiked):
		writeAPIError(w, &APIError{
			Message: "You have not liked that post.", Code: "NOT_LIKED",
			ID: errIDGalleryNotLiked, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrGalleryNoFiles):
		// Upstream has no declared error and 500s; answering INVALID_PARAM
		// is the recorded divergence.
		writeAPIError(w, invalidParamError("at least one owned file is required"))
	case errors.Is(err, account.ErrGalleryTitleInvalid):
		writeAPIError(w, invalidParamError("title must not be empty"))
	case errors.Is(err, account.ErrAccountMoved):
		writeAPIError(w, &APIError{
			Message: "Your account has moved.", Code: "YOUR_ACCOUNT_MOVED",
			ID: errIDYourAccountMoved, Kind: "permission", Status: http.StatusForbidden,
		})
	default:
		writeInternalServerError(w)
	}
}

// requireGalleryCaller authenticates a gallery mutation.
func (s *Server) requireGalleryCaller(w http.ResponseWriter, r *http.Request, prohibitMoved bool) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	if prohibitMoved && info.User.MovedToURI != "" {
		writeGalleryError(w, account.ErrAccountMoved, galleryErrorIDs{})
		return nil
	}
	return info
}

// handleGalleryPostsCreate implements POST /api/gallery/posts/create.
func (s *Server) handleGalleryPostsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string   `json:"title"`
		Description *string  `json:"description"`
		FileIDs     []string `json:"fileIds"`
		IsSensitive bool     `json:"isSensitive"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireGalleryCaller(w, r, true)
	if info == nil {
		return
	}
	if !s.checkRateLimit(w, info.User.ID, limitGalleryCreate) {
		return
	}
	p, err := s.accounts.CreatePost(r.Context(), info.User.ID, account.GalleryInput{
		Title:       body.Title,
		Description: body.Description,
		FileIDs:     body.FileIDs,
		IsSensitive: body.IsSensitive,
	})
	if err != nil {
		writeGalleryError(w, err, galleryErrorIDs{})
		return
	}
	packed, err := s.packGalleryPost(r.Context(), p, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleGalleryPostsUpdate implements POST /api/gallery/posts/update.
func (s *Server) handleGalleryPostsUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PostID      string   `json:"postId"`
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		FileIDs     []string `json:"fileIds"`
		IsSensitive *bool    `json:"isSensitive"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireGalleryCaller(w, r, true)
	if info == nil {
		return
	}
	raw := bodyAsMap(r)
	_, descriptionPresent := raw["description"]
	_, fileIDsPresent := raw["fileIds"]
	p, err := s.accounts.UpdatePost(r.Context(), body.PostID, info.User.ID, &account.GalleryUpdatePatch{
		Title:              body.Title,
		Description:        body.Description,
		DescriptionPresent: descriptionPresent,
		FileIDs:            body.FileIDs,
		FileIDsPresent:     fileIDsPresent,
		IsSensitive:        body.IsSensitive,
	})
	if err != nil {
		writeGalleryError(w, err, galleryErrorIDs{noSuchPost: errIDNoSuchGalleryPostShow})
		return
	}
	packed, err := s.packGalleryPost(r.Context(), p, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleGalleryPostsDelete implements POST /api/gallery/posts/delete.
func (s *Server) handleGalleryPostsDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PostID string `json:"postId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireGalleryCaller(w, r, false)
	if info == nil {
		return
	}
	if body.PostID == "" {
		writeAPIError(w, invalidParamError("postId is required"))
		return
	}
	err := s.accounts.DeletePost(r.Context(), body.PostID, info.User.ID, info.User.IsAdmin)
	if errors.Is(err, account.ErrGalleryPostNotFound) {
		// A foreign post reports not-found through the owner-scoped lookup;
		// upstream's ACCESS_DENIED only fires on the moderator path's
		// explicit ownership probe Cinderfish does not need.
		writeAPIError(w, &APIError{
			Message: "No such post.", Code: "NO_SUCH_POST",
			ID: errIDNoSuchGalleryPostDelete, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	if err != nil {
		writeGalleryError(w, err, galleryErrorIDs{})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGalleryPostsShow implements POST /api/gallery/posts/show.
func (s *Server) handleGalleryPostsShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PostID string `json:"postId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	p, err := s.accounts.GetGalleryPost(r.Context(), body.PostID)
	if err != nil {
		writeGalleryError(w, err, galleryErrorIDs{noSuchPost: errIDNoSuchGalleryPostShow})
		return
	}
	packed, err := s.packGalleryPost(r.Context(), p, viewerID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleGalleryPosts implements POST /api/gallery/posts.
func (s *Server) handleGalleryPosts(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryList(w, r, "all")
}

// handleGalleryFeatured implements POST /api/gallery/featured.
func (s *Server) handleGalleryFeatured(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryList(w, r, "featured")
}

// handleGalleryPopular implements POST /api/gallery/popular.
func (s *Server) handleGalleryPopular(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryList(w, r, "popular")
}

// handleUsersGalleryPosts implements POST /api/users/gallery/posts.
func (s *Server) handleUsersGalleryPosts(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryList(w, r, "user")
}

// handleIGalleryPosts implements POST /api/i/gallery/posts.
func (s *Server) handleIGalleryPosts(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryList(w, r, "my")
}

// handleIGalleryLikes implements POST /api/i/gallery/likes.
func (s *Server) handleIGalleryLikes(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	var body struct {
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
	query := account.GalleryQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	likes, err := s.accounts.ListGalleryLikes(r.Context(), info.User.ID, query)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(likes))
	for _, e := range likes {
		packed, err := s.packGalleryPost(r.Context(), e.Post, info.User.ID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, map[string]any{"id": e.ID, "post": packed})
	}
	writeJSON(w, http.StatusOK, out)
}

// serveGalleryList fans the five gallery listings out over the shared
// pagination shape.
func (s *Server) serveGalleryList(w http.ResponseWriter, r *http.Request, which string) {
	var body struct {
		UserID    string `json:"userId"`
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
	viewerID := ""
	authed := false
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
		authed = true
	}
	query := account.GalleryQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}

	var posts []*account.GalleryPost
	var err error
	switch which {
	case "all":
		posts, err = s.accounts.ListGalleryPosts(r.Context(), query)
	case "featured":
		posts, err = s.accounts.FeaturedGallery(r.Context(), body.Limit, body.UntilID)
	case "popular":
		posts, err = s.accounts.PopularGallery(r.Context())
	case "user":
		if body.UserID == "" {
			writeAPIError(w, invalidParamError("userId is required"))
			return
		}
		posts, err = s.accounts.ListUserGalleryPosts(r.Context(), body.UserID, query)
	case "my":
		if !authed {
			writeAuthenticationError(w)
			return
		}
		posts, err = s.accounts.ListMyGalleryPosts(r.Context(), viewerID, query)
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(posts))
	for _, p := range posts {
		packed, err := s.packGalleryPost(r.Context(), p, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGalleryPostsLike implements POST /api/gallery/posts/like.
func (s *Server) handleGalleryPostsLike(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryLike(w, r, true)
}

// handleGalleryPostsUnlike implements POST /api/gallery/posts/unlike.
func (s *Server) handleGalleryPostsUnlike(w http.ResponseWriter, r *http.Request) {
	s.serveGalleryLike(w, r, false)
}

func (s *Server) serveGalleryLike(w http.ResponseWriter, r *http.Request, like bool) {
	var body struct {
		PostID string `json:"postId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireGalleryCaller(w, r, true)
	if info == nil {
		return
	}
	if body.PostID == "" {
		writeAPIError(w, invalidParamError("postId is required"))
		return
	}
	var err error
	ids := galleryErrorIDs{noSuchPost: errIDNoSuchGalleryPostUnlike}
	if like {
		ids.noSuchPost = errIDNoSuchGalleryPostLike
		err = s.accounts.LikeGalleryPost(r.Context(), info.User.ID, body.PostID)
	} else {
		err = s.accounts.UnlikeGalleryPost(r.Context(), info.User.ID, body.PostID)
	}
	if err != nil {
		writeGalleryError(w, err, ids)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
