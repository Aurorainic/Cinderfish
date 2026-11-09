package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for the pages endpoints, copied from
// packages/backend/src/server/api/endpoints/pages/*.ts.
const (
	errIDPageNoSuchFileCreate = "b7b97489-0f66-4b12-a5ff-b21bd63f6e1c"
	errIDPageNameTakenCreate  = "4650348e-301c-499a-83c9-6aa988c66bc1"
	errIDNoSuchPageUpdate     = "21149b9e-3616-4778-9592-c4ce89f5a864"
	errIDPageAccessDenied     = "3c15cd52-3b4b-4274-967d-6456fc4f792b"
	errIDPageNoSuchFileUpdate = "cfc23c7c-3887-490e-af30-0ed576703c82"
	errIDPageNameTakenUpdate  = "2298a392-d4a1-44c5-9ebb-ac1aeaa5a9ab"
	errIDNoSuchPageDelete     = "eb0c6e1d-d519-4764-9486-52a7e1c6392a"
	errIDPageAccessDeniedDel  = "8b741b3e-2c22-44b3-a15f-29949aa1601e"
	errIDNoSuchPageShow       = "222120c0-3ead-4528-811b-b96f233388d7"
	errIDNoSuchPageLike       = "cc98a8a2-0dc3-4123-b198-62c71df18ed3"
	errIDPageYourPage         = "28800466-e6db-40f2-8fae-bf9e82aa92b8"
	errIDPageAlreadyLiked     = "d4c1edbe-7da2-4eae-8714-1acfd2d63941"
	errIDNoSuchPageUnlike     = "a0d41e20-1993-40bd-890e-f6e560ae648e"
	errIDPageNotLiked         = "f5e586b0-ce93-4050-b0e3-7f31af5259ee"
	// i/update's pinnedPageId ownership error.
	errIDPageOtherProfile = "8e01b590-7eb9-431b-a239-860e086c408e"
)

// limitPagesCreate is Misskey's pages/create rate cap.
var limitPagesCreate = rateLimit{bucket: "pages/create", max: 10, duration: time.Hour}

// pageErrorIDs carries the per-endpoint error ids of the page family.
type pageErrorIDs struct {
	noSuchPage string
	noSuchFile string
	nameTaken  string
	access     string
}

// packPage renders the Page schema. attachedFiles walks the content for
// image blocks and loads the author's drive files; isLiked travels only
// with a viewer.
func (s *Server) packPage(ctx context.Context, p *account.Page, viewerID string) (map[string]any, error) {
	packed := map[string]any{
		"id":                  p.ID,
		"createdAt":           p.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":           p.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"userId":              p.UserID,
		"title":               p.Title,
		"name":                p.Name,
		"summary":             p.Summary,
		"content":             jsonVal(p.Content),
		"variables":           jsonVal(p.Variables),
		"script":              p.Script,
		"hideTitleWhenPinned": p.HideTitleWhenPinned,
		"alignCenter":         p.AlignCenter,
		"font":                p.Font,
		"eyeCatchingImageId":  p.EyeCatchingImageID,
		"likedCount":          p.LikedCount,
	}
	if u, err := s.accounts.GetByID(ctx, p.UserID); err == nil && u != nil {
		packed["user"] = packUserLite(u, "", loadProfile(ctx, s, p.UserID))
	}
	if p.EyeCatchingImageID != nil && s.drive != nil {
		if file, err := s.drive.Get(ctx, *p.EyeCatchingImageID); err == nil && file != nil {
			packed["eyeCatchingImage"] = packDriveFile(file)
		} else {
			packed["eyeCatchingImage"] = nil
		}
	} else {
		packed["eyeCatchingImage"] = nil
	}
	// The image blocks reference the author's drive files; upstream walks
	// the content recursively for type === 'image' and loads those files.
	fileIDs := collectPageImageIDs(p.Content)
	files, err := s.orderedDriveFiles(ctx, p.UserID, fileIDs)
	if err != nil {
		return nil, err
	}
	fileList := make([]any, 0, len(files))
	for _, f := range files {
		fileList = append(fileList, packDriveFile(f))
	}
	packed["attachedFiles"] = fileList
	if viewerID != "" {
		liked, err := s.accounts.HasPageLike(ctx, viewerID, p.ID)
		if err != nil {
			return nil, err
		}
		packed["isLiked"] = liked
	}
	return packed, nil
}

// jsonVal decodes the stored JSON of a page column into a generic value
// for packing; null on a decode failure never blocks a page read.
func jsonVal(raw json.RawMessage) any {
	if len(raw) == 0 {
		return []any{}
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return []any{}
	}
	return v
}

// collectPageImageIDs walks a page content tree collecting the fileId of
// every image block, recursing through section children.
func collectPageImageIDs(content json.RawMessage) []string {
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return []string{}
	}
	ids := []string{}
	seen := map[string]bool{}
	var walk func(blocks []map[string]any)
	walk = func(blocks []map[string]any) {
		for _, block := range blocks {
			typ, _ := block["type"].(string)
			if typ == "image" {
				if id, ok := block["fileId"].(string); ok && id != "" && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
			if children, ok := block["children"].([]any); ok {
				childBlocks := make([]map[string]any, 0, len(children))
				for _, c := range children {
					if m, ok := c.(map[string]any); ok {
						childBlocks = append(childBlocks, m)
					}
				}
				walk(childBlocks)
			}
		}
	}
	walk(blocks)
	if ids == nil {
		ids = []string{}
	}
	return ids
}

// writePageError maps the page service sentinels onto Misskey's envelope
// with the verbatim per-endpoint ids.
func writePageError(w http.ResponseWriter, err error, ids pageErrorIDs) {
	switch {
	case errors.Is(err, account.ErrPageNotFound):
		writeAPIError(w, &APIError{
			Message: "No such page.", Code: "NO_SUCH_PAGE",
			ID: ids.noSuchPage, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrPageNameTaken):
		writeAPIError(w, &APIError{
			Message: "Name already exists.", Code: "NAME_ALREADY_EXISTS",
			ID: ids.nameTaken, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageFileNotFound):
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: ids.noSuchFile, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageYourPage):
		writeAPIError(w, &APIError{
			Message: "You cannot like your own page.", Code: "YOUR_PAGE",
			ID: errIDPageYourPage, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageAlreadyLiked):
		writeAPIError(w, &APIError{
			Message: "The page has already been liked.", Code: "ALREADY_LIKED",
			ID: errIDPageAlreadyLiked, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageNotLiked):
		writeAPIError(w, &APIError{
			Message: "You have not liked that page.", Code: "NOT_LIKED",
			ID: errIDPageNotLiked, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageOtherProfile):
		// i/update's pinnedPageId: a foreign page answers NO_SUCH_PAGE with
		// that endpoint's id.
		writeAPIError(w, &APIError{
			Message: "No such page.", Code: "NO_SUCH_PAGE",
			ID: errIDPageOtherProfile, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrPageNameInvalid):
		writeAPIError(w, invalidParamError("invalid page title or name"))
	case errors.Is(err, account.ErrPageFontInvalid):
		writeAPIError(w, invalidParamError("font must be serif or sans-serif"))
	case errors.Is(err, account.ErrAccountMoved):
		writeAPIError(w, &APIError{
			Message: "Your account has moved.", Code: "YOUR_ACCOUNT_MOVED",
			ID: errIDYourAccountMoved, Kind: "permission", Status: http.StatusForbidden,
		})
	default:
		writeInternalServerError(w)
	}
}

// requirePageCaller authenticates a page mutation.
func (s *Server) requirePageCaller(w http.ResponseWriter, r *http.Request, prohibitMoved bool) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	if prohibitMoved && info.User.MovedToURI != "" {
		writePageError(w, account.ErrAccountMoved, pageErrorIDs{})
		return nil
	}
	return info
}

// handlePagesCreate implements POST /api/pages/create.
func (s *Server) handlePagesCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title               string          `json:"title"`
		Name                string          `json:"name"`
		Summary             *string         `json:"summary"`
		AlignCenter         bool            `json:"alignCenter"`
		HideTitleWhenPinned bool            `json:"hideTitleWhenPinned"`
		Font                string          `json:"font"`
		EyeCatchingImageID  *string         `json:"eyeCatchingImageId"`
		Content             json.RawMessage `json:"content"`
		Variables           json.RawMessage `json:"variables"`
		Script              *string         `json:"script"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requirePageCaller(w, r, true)
	if info == nil {
		return
	}
	if !s.checkRateLimit(w, info.User.ID, limitPagesCreate) {
		return
	}
	script := ""
	if body.Script != nil {
		script = *body.Script
	}
	p, err := s.accounts.CreatePage(r.Context(), info.User.ID, account.PageInput{
		Title:               body.Title,
		Name:                body.Name,
		Summary:             body.Summary,
		AlignCenter:         body.AlignCenter,
		HideTitleWhenPinned: body.HideTitleWhenPinned,
		Font:                body.Font,
		EyeCatchingImageID:  body.EyeCatchingImageID,
		Content:             body.Content,
		Variables:           body.Variables,
		Script:              script,
	})
	if err != nil {
		writePageError(w, err, pageErrorIDs{
			noSuchFile: errIDPageNoSuchFileCreate,
			nameTaken:  errIDPageNameTakenCreate,
		})
		return
	}
	packed, err := s.packPage(r.Context(), p, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handlePagesUpdate implements POST /api/pages/update. The response is
// empty, as upstream's res declares.
func (s *Server) handlePagesUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PageID              string          `json:"pageId"`
		Title               *string         `json:"title"`
		Name                *string         `json:"name"`
		Summary             *string         `json:"summary"`
		AlignCenter         *bool           `json:"alignCenter"`
		HideTitleWhenPinned *bool           `json:"hideTitleWhenPinned"`
		Font                *string         `json:"font"`
		EyeCatchingImageID  *string         `json:"eyeCatchingImageId"`
		Content             json.RawMessage `json:"content"`
		Variables           json.RawMessage `json:"variables"`
		Script              *string         `json:"script"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requirePageCaller(w, r, true)
	if info == nil {
		return
	}
	raw := bodyAsMap(r)
	_, summaryPresent := raw["summary"]
	_, eyePresent := raw["eyeCatchingImageId"]
	_, contentPresent := raw["content"]
	_, variablesPresent := raw["variables"]
	_, scriptPresent := raw["script"]
	patch := &account.PagePatch{
		Title:               body.Title,
		Name:                body.Name,
		Summary:             body.Summary,
		SummaryPresent:      summaryPresent,
		AlignCenter:         body.AlignCenter,
		HideTitleWhenPinned: body.HideTitleWhenPinned,
		Font:                body.Font,
		EyeCatchingImageID:  body.EyeCatchingImageID,
		EyeCatchingPresent:  eyePresent,
		Content:             body.Content,
		ContentPresent:      contentPresent,
		Variables:           body.Variables,
		VariablesPresent:    variablesPresent,
		Script:              body.Script,
		ScriptPresent:       scriptPresent,
	}
	if len(patch.Content) == 0 && patch.ContentPresent {
		patch.Content = json.RawMessage("[]")
	}
	if len(patch.Variables) == 0 && patch.VariablesPresent {
		patch.Variables = json.RawMessage("[]")
	}
	if err := s.accounts.UpdatePage(r.Context(), body.PageID, info.User.ID, info.User.IsAdmin, patch); err != nil {
		writePageError(w, err, pageErrorIDs{
			noSuchPage: errIDNoSuchPageUpdate,
			noSuchFile: errIDPageNoSuchFileUpdate,
			nameTaken:  errIDPageNameTakenUpdate,
			access:     errIDPageAccessDenied,
		})
		return
	}
	// Upstream's handler returns nothing, so ApiCallService answers 204.
	w.WriteHeader(http.StatusNoContent)
}

// handlePagesDelete implements POST /api/pages/delete.
func (s *Server) handlePagesDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PageID string `json:"pageId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requirePageCaller(w, r, false)
	if info == nil {
		return
	}
	if body.PageID == "" {
		writeAPIError(w, invalidParamError("pageId is required"))
		return
	}
	if err := s.accounts.DeletePage(r.Context(), body.PageID, info.User.ID, info.User.IsAdmin); err != nil {
		writePageError(w, err, pageErrorIDs{noSuchPage: errIDNoSuchPageDelete, access: errIDPageAccessDeniedDel})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePagesShow implements POST /api/pages/show, with both lookup
// forms: {pageId} or {name, username}.
func (s *Server) handlePagesShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PageID   string `json:"pageId"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	var page *account.Page
	var err error
	if body.PageID != "" {
		page, err = s.accounts.GetPage(r.Context(), body.PageID)
	} else if body.Name != "" && body.Username != "" {
		page, err = s.accounts.GetPageByName(r.Context(), body.Username, body.Name)
	} else {
		writeAPIError(w, invalidParamError("pageId, or name and username, are required"))
		return
	}
	if err != nil {
		writePageError(w, err, pageErrorIDs{noSuchPage: errIDNoSuchPageShow})
		return
	}
	packed, err := s.packPage(r.Context(), page, viewerID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handlePagesFeatured implements POST /api/pages/featured. No params —
// any limit the client sends is ignored, as upstream does.
func (s *Server) handlePagesFeatured(w http.ResponseWriter, r *http.Request) {
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	pages, err := s.accounts.FeaturedPages(r.Context())
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(pages))
	for _, p := range pages {
		packed, err := s.packPage(r.Context(), p, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handlePagesLike implements POST /api/pages/like.
func (s *Server) handlePagesLike(w http.ResponseWriter, r *http.Request) {
	s.servePageLike(w, r, true)
}

// handlePagesUnlike implements POST /api/pages/unlike.
func (s *Server) handlePagesUnlike(w http.ResponseWriter, r *http.Request) {
	s.servePageLike(w, r, false)
}

func (s *Server) servePageLike(w http.ResponseWriter, r *http.Request, like bool) {
	var body struct {
		PageID string `json:"pageId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requirePageCaller(w, r, true)
	if info == nil {
		return
	}
	if body.PageID == "" {
		writeAPIError(w, invalidParamError("pageId is required"))
		return
	}
	var err error
	ids := pageErrorIDs{noSuchPage: errIDNoSuchPageUnlike}
	if like {
		ids.noSuchPage = errIDNoSuchPageLike
		err = s.accounts.LikePage(r.Context(), info.User.ID, body.PageID)
	} else {
		err = s.accounts.UnlikePage(r.Context(), info.User.ID, body.PageID)
	}
	if err != nil {
		writePageError(w, err, ids)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleIPages implements POST /api/i/pages: the caller's pages, private
// ones included, since pages have no visibility surface upstream either.
func (s *Server) handleIPages(w http.ResponseWriter, r *http.Request) {
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
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	pages, err := s.accounts.ListPages(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedPages(w, r, pages, info.User.ID)
}

// handleIPageLikes implements POST /api/i/page-likes.
func (s *Server) handleIPageLikes(w http.ResponseWriter, r *http.Request) {
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
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	likes, err := s.accounts.ListPageLikes(r.Context(), info.User.ID, body.Limit, body.UntilID, body.SinceID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(likes))
	for _, e := range likes {
		packed, err := s.packPage(r.Context(), e.Page, info.User.ID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, map[string]any{"id": e.ID, "page": packed})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUsersPages implements POST /api/users/pages: the public pages of
// one account.
func (s *Server) handleUsersPages(w http.ResponseWriter, r *http.Request) {
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
	if body.UserID == "" {
		writeAPIError(w, invalidParamError("userId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	pages, err := s.accounts.ListUserPages(r.Context(), body.UserID, body.Limit, body.SinceID, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedPages(w, r, pages, viewerID)
}

func (s *Server) writePackedPages(w http.ResponseWriter, r *http.Request, pages []*account.Page, viewerID string) {
	out := make([]any, 0, len(pages))
	for _, p := range pages {
		packed, err := s.packPage(r.Context(), p, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// applyPinnedPage overrides the packed pinnedPageId/pinnedPage of a user
// detail with the profile's pin, nil when the profile pins nothing. The
// packed page omits isLiked when no viewer is in scope.
func (s *Server) applyPinnedPage(ctx context.Context, userID string, packed map[string]any) {
	if s.accounts == nil {
		return
	}
	pageID, err := s.accounts.PinnedPageID(ctx, userID)
	if err != nil || pageID == nil {
		return
	}
	packed["pinnedPageId"] = *pageID
	if p, err := s.accounts.GetPage(ctx, *pageID); err == nil {
		pagePack, err := s.packPage(ctx, p, userID)
		if err == nil {
			packed["pinnedPage"] = pagePack
		}
	}
}
