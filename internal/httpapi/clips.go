package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// Misskey error ids for the clips endpoints, copied from
// packages/backend/src/server/api/endpoints/clips/*.ts. The same logical
// error often carries a different id per endpoint upstream, so ids travel
// with the handler through clipErrorIDs.
const (
	errIDTooManyClips         = "920f7c2d-6208-4b76-8082-e632020f5883"
	errIDNoSuchClipUpdate     = "b4d92d70-b216-46fa-9a3f-a8c811699257"
	errIDNoSuchClipDelete     = "70ca08ba-6865-4630-b6fb-8494759aa754"
	errIDNoSuchClipShow       = "c3c5fe33-d62c-44d2-9ea5-d997703f5c20"
	errIDNoSuchClipFavorite   = "4c2aaeae-80d8-4250-9606-26cb1fdb77a5"
	errIDClipAlreadyFavorited = "92658936-c625-4273-8326-2d790129256e"
	errIDNoSuchClipUnfavorite = "2603966e-b865-426c-94a7-af4a01241dc1"
	errIDClipNotFavorited     = "90c3a9e8-b321-4dae-bf57-2bf79bbcc187"
	errIDNoSuchClipAddNote    = "d6e76cc0-a1b5-4c7c-a287-73fa9c716dcf"
	errIDNoSuchClipNoteAdd    = "fc8c0b49-c7a3-4664-a0a6-b418d386bb8b"
	// Load-bearing id: Firefish's note menu matches it verbatim to offer
	// the "already clipped, unclip instead" confirmation.
	errIDAlreadyClipped       = "734806c4-542c-463a-9311-15c512803965"
	errIDTooManyClipNotes     = "f0dba960-ff73-4615-8df4-d6ac5d9dc118"
	errIDNoSuchClipRemoveNote = "b80525c6-97f7-49d7-a42d-ebccd49cfd52"
	errIDNoSuchClipNoteRemove = "aff017de-190e-434b-893e-33a9ff5049d8"
	errIDNoSuchClipNotes      = "1d7645e6-2b6d-4635-b0fe-fe22b0e72e00"
	errIDNoSuchNoteNotesClips = "47db1a1c-b0af-458d-8fb4-986e4efafe1e"
)

// limitClipsAddNote is Misskey's add-note rate cap (20 per hour).
var limitClipsAddNote = rateLimit{bucket: "clips/add-note", max: 20, duration: time.Hour}

// clipErrorIDs carries the per-endpoint error ids of the clip family.
type clipErrorIDs struct {
	noSuchClip string
	noSuchNote string
}

// packClip renders Misskey's Clip schema. favoritedCount always travels;
// isFavorited only when a viewer is known; notesCount only for the owner,
// as upstream's pack does.
func (s *Server) packClip(ctx context.Context, c *account.Clip, viewerID string) (map[string]any, error) {
	packed := map[string]any{
		"id":            c.ID,
		"createdAt":     c.CreatedAt.UTC().Format(time.RFC3339Nano),
		"lastClippedAt": nil,
		"userId":        c.UserID,
		"name":          c.Name,
		"description":   nil,
		"isPublic":      c.IsPublic,
	}
	if c.LastClippedAt != nil {
		packed["lastClippedAt"] = c.LastClippedAt.UTC().Format(time.RFC3339Nano)
	}
	if c.Description != nil {
		packed["description"] = *c.Description
	}
	if u, err := s.accounts.GetByID(ctx, c.UserID); err == nil && u != nil {
		packed["user"] = packUserLite(u, "", loadProfile(ctx, s, c.UserID))
	}
	favorites, err := s.accounts.CountClipFavorites(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	packed["favoritedCount"] = favorites
	if viewerID != "" {
		favorited, err := s.accounts.HasClipFavorite(ctx, viewerID, c.ID)
		if err != nil {
			return nil, err
		}
		packed["isFavorited"] = favorited
	}
	if viewerID == c.UserID {
		notesCount, err := s.accounts.CountClipNotes(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		packed["notesCount"] = notesCount
	}
	return packed, nil
}

// writeClipError maps the clip service sentinel errors onto Misskey's
// envelope. Statuses follow the user-lists precedent (404 for lookups).
func writeClipError(w http.ResponseWriter, err error, ids clipErrorIDs) {
	switch {
	case errors.Is(err, account.ErrClipNotFound):
		writeAPIError(w, &APIError{
			Message: "No such clip.", Code: "NO_SUCH_CLIP",
			ID: ids.noSuchClip, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrTooManyClips):
		writeAPIError(w, &APIError{
			Message: "You have too many clips.", Code: "TOO_MANY_CLIPS",
			ID: errIDTooManyClips, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrTooManyClipNotes):
		writeAPIError(w, &APIError{
			Message: "You have too many notes in that clip.", Code: "TOO_MANY_CLIP_NOTES",
			ID: errIDTooManyClipNotes, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrAlreadyClipped):
		writeAPIError(w, &APIError{
			Message: "That note has already been clipped.", Code: "ALREADY_CLIPPED",
			ID: errIDAlreadyClipped, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrClipNameInvalid):
		writeAPIError(w, invalidParamError("name must be 1..100 characters"))
	case errors.Is(err, account.ErrClipDescInvalid):
		writeAPIError(w, invalidParamError("description must be at most 2048 characters"))
	case errors.Is(err, account.ErrClipAlreadyFavorited):
		writeAPIError(w, &APIError{
			Message: "The clip has already been favorited.", Code: "ALREADY_FAVORITED",
			ID: errIDClipAlreadyFavorited, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrClipNotFavorited):
		writeAPIError(w, &APIError{
			Message: "You have not favorited that clip.", Code: "NOT_FAVORITED",
			ID: errIDClipNotFavorited, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrClipNoteMissing):
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: ids.noSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrClipNoteNotFound):
		writeAPIError(w, &APIError{
			Message: "No such note.", Code: "NO_SUCH_NOTE",
			ID: ids.noSuchNote, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrAccountMoved):
		writeAPIError(w, &APIError{
			Message: "Your account has moved.", Code: "YOUR_ACCOUNT_MOVED",
			ID: errIDYourAccountMoved, Kind: "permission", Status: http.StatusForbidden,
		})
	default:
		writeInternalServerError(w)
	}
}

// requireClipCaller authenticates a mutating clip request and rejects
// accounts that have moved, the prohibitMoved guard.
func (s *Server) requireClipCaller(w http.ResponseWriter, r *http.Request) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	if info.User.MovedToURI != "" {
		writeClipError(w, account.ErrAccountMoved, clipErrorIDs{})
		return nil
	}
	return info
}

// handleClipsCreate implements POST /api/clips/create.
func (s *Server) handleClipsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		IsPublic    bool    `json:"isPublic"`
		Description *string `json:"description"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	c, err := s.accounts.CreateClip(r.Context(), info.User.ID, body.Name, body.IsPublic, body.Description)
	if err != nil {
		writeClipError(w, err, clipErrorIDs{})
		return
	}
	packed, err := s.packClip(r.Context(), c, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleClipsUpdate implements POST /api/clips/update. description must
// distinguish "omitted" from explicit null, which a plain pointer cannot,
// so key presence is read from the raw body map.
func (s *Server) handleClipsUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID      string  `json:"clipId"`
		Name        *string `json:"name"`
		IsPublic    *bool   `json:"isPublic"`
		Description *string `json:"description"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}

	var description *string
	if _, present := bodyAsMap(r)["description"]; present {
		// An explicit null decodes to a nil pointer; keep it nil but marked
		// present so the service clears the column.
		if body.Description != nil {
			description = body.Description
		} else {
			empty := ""
			description = &empty
		}
	}

	c, err := s.accounts.UpdateClip(r.Context(), body.ClipID, info.User.ID, body.Name, body.IsPublic, description)
	if err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipUpdate})
		return
	}
	s.publishClipEvent(c.ID, "updated")
	packed, err := s.packClip(r.Context(), c, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleClipsDelete implements POST /api/clips/delete.
func (s *Server) handleClipsDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}
	// Upstream publishes the deletion before the row is gone; clients
	// close on the event, not on a follow-up read.
	s.publishClipEvent(body.ClipID, "deleted")
	if err := s.accounts.DeleteClip(r.Context(), body.ClipID, info.User.ID); err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipDelete})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleClipsList implements POST /api/clips/list.
func (s *Server) handleClipsList(w http.ResponseWriter, r *http.Request) {
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
	query := account.ClipQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	clips, err := s.accounts.ListClips(r.Context(), info.User.ID, query)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedClips(w, r, clips, info.User.ID)
}

func (s *Server) writePackedClips(w http.ResponseWriter, r *http.Request, clips []*account.Clip, viewerID string) {
	out := make([]any, 0, len(clips))
	for _, c := range clips {
		packed, err := s.packClip(r.Context(), c, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleClipsMyFavorites implements POST /api/clips/my-favorites. Upstream
// carries no pagination and no explicit order; private favourites are
// included, as upstream joins on the caller's favourites alone.
func (s *Server) handleClipsMyFavorites(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	clips, err := s.accounts.ListFavoriteClips(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedClips(w, r, clips, info.User.ID)
}

// handleClipsShow implements POST /api/clips/show.
func (s *Server) handleClipsShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	c, err := s.accounts.GetClip(r.Context(), body.ClipID, viewerID)
	if err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipShow})
		return
	}
	packed, err := s.packClip(r.Context(), c, viewerID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleClipsFavorite implements POST /api/clips/favorite.
func (s *Server) handleClipsFavorite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}
	if err := s.accounts.FavoriteClip(r.Context(), info.User.ID, body.ClipID); err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipFavorite})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleClipsUnfavorite implements POST /api/clips/unfavorite.
func (s *Server) handleClipsUnfavorite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}
	if err := s.accounts.UnfavoriteClip(r.Context(), info.User.ID, body.ClipID); err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipUnfavorite})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleClipsAddNote implements POST /api/clips/add-note.
func (s *Server) handleClipsAddNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
		NoteID string `json:"noteId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if !s.checkRateLimit(w, info.User.ID, limitClipsAddNote) {
		return
	}
	if body.ClipID == "" || body.NoteID == "" {
		writeAPIError(w, invalidParamError("clipId and noteId are required"))
		return
	}
	ids := clipErrorIDs{noSuchClip: errIDNoSuchClipAddNote, noSuchNote: errIDNoSuchClipNoteAdd}
	if err := s.accounts.AddClipNote(r.Context(), body.ClipID, info.User.ID, body.NoteID); err != nil {
		writeClipError(w, err, ids)
		return
	}
	s.publishClipEvent(body.ClipID, "updated")
	w.WriteHeader(http.StatusNoContent)
}

// handleClipsRemoveNote implements POST /api/clips/remove-note.
func (s *Server) handleClipsRemoveNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID string `json:"clipId"`
		NoteID string `json:"noteId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireClipCaller(w, r)
	if info == nil {
		return
	}
	if body.ClipID == "" || body.NoteID == "" {
		writeAPIError(w, invalidParamError("clipId and noteId are required"))
		return
	}
	if err := s.accounts.RemoveClipNote(r.Context(), body.ClipID, info.User.ID, body.NoteID); err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipRemoveNote, noSuchNote: errIDNoSuchClipNoteRemove})
		return
	}
	s.publishClipEvent(body.ClipID, "updated")
	w.WriteHeader(http.StatusNoContent)
}

// handleClipsNotes implements POST /api/clips/notes.
func (s *Server) handleClipsNotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClipID    string `json:"clipId"`
		Limit     int    `json:"limit"`
		SinceID   string `json:"sinceId"`
		UntilID   string `json:"untilId"`
		SinceDate *int64 `json:"sinceDate"`
		UntilDate *int64 `json:"untilDate"`
		Search    string `json:"search"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ClipID == "" {
		writeAPIError(w, invalidParamError("clipId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	if _, err := s.accounts.GetClip(r.Context(), body.ClipID, viewerID); err != nil {
		writeClipError(w, err, clipErrorIDs{noSuchClip: errIDNoSuchClipNotes})
		return
	}
	if body.SinceID == "" && body.SinceDate != nil {
		id, err := s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.SinceDate))
		if err != nil {
			writeInternalServerError(w)
			return
		}
		body.SinceID = id
	}
	if body.UntilID == "" && body.UntilDate != nil {
		id, err := s.timeline.SinceIDForDate(r.Context(), unixMilli(*body.UntilDate))
		if err != nil {
			writeInternalServerError(w)
			return
		}
		body.UntilID = id
	}
	// The search words must each ILIKE-match the text or the cw.
	var words []string
	if trimmed := strings.TrimSpace(body.Search); trimmed != "" {
		words = strings.Fields(trimmed)
	}
	notes, err := s.timeline.ClipNotes(r.Context(), body.ClipID, viewerID, body.Limit, body.SinceID, body.UntilID, words)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	// Upstream packs base Notes here, not NoteDetailed.
	out := make([]any, 0, len(notes))
	for _, n := range notes {
		pc := s.notePackContext(r.Context(), n, viewerID)
		out = append(out, packNote(n, pc))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUsersClips implements POST /api/users/clips: only the public
// clips of the target account, with no user-existence check, as upstream.
func (s *Server) handleUsersClips(w http.ResponseWriter, r *http.Request) {
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

	// Upstream pages users/clips with makePaginationQuery on public clips
	// of the target; the cap and cursors mirror clips/list.
	query := account.ClipQuery{Limit: body.Limit, SinceID: body.SinceID, UntilID: body.UntilID}
	if body.SinceDate != nil {
		t := time.UnixMilli(*body.SinceDate)
		query.SinceDate = &t
	}
	if body.UntilDate != nil {
		t := time.UnixMilli(*body.UntilDate)
		query.UntilDate = &t
	}
	clips, err := s.accounts.ListClipsOf(r.Context(), body.UserID, true, query)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedClips(w, r, clips, viewerID)
}

// handleNotesClips implements POST /api/notes/clips: the public clips
// containing a note, never private ones, with only an existence check on
// the note, as upstream.
func (s *Server) handleNotesClips(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NoteID string `json:"noteId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.NoteID == "" {
		writeAPIError(w, invalidParamError("noteId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	if _, err := s.notes.Get(r.Context(), body.NoteID); err != nil {
		if errors.Is(err, note.ErrNotFound) {
			writeAPIError(w, &APIError{
				Message: "No such note.", Code: "NO_SUCH_NOTE",
				ID: errIDNoSuchNoteNotesClips, Kind: "client", Status: http.StatusNotFound,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	clips, err := s.accounts.ClipsOfNote(r.Context(), body.NoteID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	s.writePackedClips(w, r, clips, viewerID)
}
