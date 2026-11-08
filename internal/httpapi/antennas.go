package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// Misskey error ids for the antennas endpoints, copied from
// packages/backend/src/server/api/endpoints/antennas/*.ts. NO_SUCH_ANTENNA
// carries a different id per endpoint and antennas/remove-note shares the
// notes id, as upstream does.
const (
	errIDNoSuchAntennaUpdate  = "10c673ac-8852-48eb-aa1f-f5b67f069290"
	errIDNoSuchUserListUpdate = "1c6b35c9-943e-48c2-81e4-2844989407f7"
	errIDEmptyKeywordUpdate   = "721aaff6-4e1b-4d88-8de6-877fae9f68c4"
	errIDNoSuchAntennaDelete  = "b34dcf9d-348f-44bb-99d0-6c9314cfe2df"
	errIDNoSuchAntennaShow    = "c06569fb-b025-4f23-b22d-1fcd20d2816b"
	errIDNoSuchAntennaNotes   = "850926e0-fd3b-49b6-b69a-b28a5dbd82fe"
	errIDNoSuchUserListCreate = "95063e93-a283-4b8b-9aa5-bcdb8df69a7f"
	errIDTooManyAntennas      = "faf47050-e8b5-438c-913c-db2b1576fde4"
	errIDEmptyKeywordCreate   = "53ee222e-1ddd-4f9a-92e5-9fb82ddb463a"
)

// antennaErrorIDs carries the per-endpoint error ids of the antenna family.
type antennaErrorIDs struct {
	noSuchAntenna  string
	noSuchUserList string
	emptyKeyword   string
}

// packAntenna renders Misskey's Antenna schema. hasUnreadNote and notify
// are hardcoded false, as upstream's entity service does.
func (s *Server) packAntenna(a *account.Antenna) map[string]any {
	keywords := make([][]string, len(a.Keywords))
	for i, group := range a.Keywords {
		keywords[i] = append([]string{}, group...)
	}
	exclude := make([][]string, len(a.ExcludeKeywords))
	for i, group := range a.ExcludeKeywords {
		exclude[i] = append([]string{}, group...)
	}
	var userListID any
	if a.UserListID != nil {
		userListID = *a.UserListID
	}
	users := a.Users
	if users == nil {
		users = []string{}
	}
	return map[string]any{
		"id":                             a.ID,
		"createdAt":                      a.CreatedAt.UTC().Format(time.RFC3339Nano),
		"name":                           a.Name,
		"keywords":                       keywords,
		"excludeKeywords":                exclude,
		"src":                            a.Src,
		"userListId":                     userListID,
		"users":                          users,
		"caseSensitive":                  a.CaseSensitive,
		"localOnly":                      a.LocalOnly,
		"excludeBots":                    a.ExcludeBots,
		"withReplies":                    a.WithReplies,
		"withFile":                       a.WithFile,
		"isActive":                       a.IsActive,
		"hasUnreadNote":                  false,
		"notify":                         false,
		"excludeNotesInSensitiveChannel": a.ExcludeSensitiveChannel,
	}
}

// writeAntennaError maps the antenna service sentinel errors onto
// Misskey's envelope, following the user-lists/clips precedent of 404 for
// lookups.
func writeAntennaError(w http.ResponseWriter, err error, ids antennaErrorIDs) {
	switch {
	case errors.Is(err, account.ErrAntennaNotFound):
		writeAPIError(w, &APIError{
			Message: "No such antenna.", Code: "NO_SUCH_ANTENNA",
			ID: ids.noSuchAntenna, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrUserListNotFound):
		writeAPIError(w, &APIError{
			Message: "No such user list.", Code: "NO_SUCH_USER_LIST",
			ID: ids.noSuchUserList, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrTooManyAntennas):
		writeAPIError(w, &APIError{
			Message: "You have too many antennas.", Code: "TOO_MANY_ANTENNAS",
			ID: errIDTooManyAntennas, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrEmptyKeyword):
		writeAPIError(w, &APIError{
			Message: "Empty keyword.", Code: "EMPTY_KEYWORD",
			ID: ids.emptyKeyword, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrAntennaSrcInvalid):
		writeAPIError(w, invalidParamError("src must be one of home/all/users/list/users_blacklist"))
	case errors.Is(err, account.ErrAntennaNameInvalid):
		writeAPIError(w, invalidParamError("name must be 1..100 characters"))
	case errors.Is(err, account.ErrAccountMoved):
		writeAPIError(w, &APIError{
			Message: "Your account has moved.", Code: "YOUR_ACCOUNT_MOVED",
			ID: errIDYourAccountMoved, Kind: "permission", Status: http.StatusForbidden,
		})
	default:
		writeInternalServerError(w)
	}
}

// requireAntennaCaller authenticates a mutating antenna request and
// rejects accounts that have moved.
func (s *Server) requireAntennaCaller(w http.ResponseWriter, r *http.Request) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	if info.User.MovedToURI != "" {
		writeAntennaError(w, account.ErrAccountMoved, antennaErrorIDs{})
		return nil
	}
	return info
}

// decodeKeywordGroups reads the keyword group arrays of a request body.
func decodeKeywordGroupsFromRequest(raw map[string]any, key string) ([][]string, bool) {
	value, present := raw[key]
	if !present {
		return nil, false
	}
	list, ok := value.([]any)
	if !ok {
		return nil, true
	}
	groups := make([][]string, 0, len(list))
	for _, groupValue := range list {
		group := []string{}
		if words, ok := groupValue.([]any); ok {
			for _, wordValue := range words {
				if word, ok := wordValue.(string); ok {
					group = append(group, word)
				}
			}
		}
		groups = append(groups, group)
	}
	return groups, true
}

// handleAntennasCreate implements POST /api/antennas/create.
func (s *Server) handleAntennasCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name                    string     `json:"name"`
		Src                     string     `json:"src"`
		UserListID              *string    `json:"userListId"`
		Users                   []string   `json:"users"`
		Keywords                [][]string `json:"keywords"`
		ExcludeKeywords         [][]string `json:"excludeKeywords"`
		CaseSensitive           bool       `json:"caseSensitive"`
		ExcludeBots             bool       `json:"excludeBots"`
		WithReplies             bool       `json:"withReplies"`
		WithFile                bool       `json:"withFile"`
		LocalOnly               bool       `json:"localOnly"`
		ExcludeSensitiveChannel bool       `json:"excludeNotesInSensitiveChannel"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireAntennaCaller(w, r)
	if info == nil {
		return
	}
	a, err := s.accounts.CreateAntenna(r.Context(), info.User.ID, account.AntennaInput{
		Name:                    body.Name,
		Src:                     body.Src,
		UserListID:              body.UserListID,
		Users:                   body.Users,
		Keywords:                body.Keywords,
		ExcludeKeywords:         body.ExcludeKeywords,
		CaseSensitive:           body.CaseSensitive,
		ExcludeBots:             body.ExcludeBots,
		WithReplies:             body.WithReplies,
		WithFile:                body.WithFile,
		LocalOnly:               body.LocalOnly,
		ExcludeSensitiveChannel: body.ExcludeSensitiveChannel,
	})
	if err != nil {
		writeAntennaError(w, err, antennaErrorIDs{
			noSuchUserList: errIDNoSuchUserListCreate,
			emptyKeyword:   errIDEmptyKeywordCreate,
		})
		return
	}
	writeJSON(w, http.StatusOK, s.packAntenna(a))
}

// handleAntennasUpdate implements POST /api/antennas/update. Keyword
// presence is read from the raw body because the merge distinguishes
// "omitted" from "explicitly empty".
func (s *Server) handleAntennasUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AntennaID               string     `json:"antennaId"`
		Name                    *string    `json:"name"`
		Src                     *string    `json:"src"`
		UserListID              *string    `json:"userListId"`
		Users                   []string   `json:"users"`
		Keywords                [][]string `json:"keywords"`
		ExcludeKeywords         [][]string `json:"excludeKeywords"`
		CaseSensitive           *bool      `json:"caseSensitive"`
		ExcludeBots             *bool      `json:"excludeBots"`
		WithReplies             *bool      `json:"withReplies"`
		WithFile                *bool      `json:"withFile"`
		LocalOnly               *bool      `json:"localOnly"`
		ExcludeSensitiveChannel *bool      `json:"excludeNotesInSensitiveChannel"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireAntennaCaller(w, r)
	if info == nil {
		return
	}
	if body.AntennaID == "" {
		writeAPIError(w, invalidParamError("antennaId is required"))
		return
	}
	raw := bodyAsMap(r)
	keywords, keywordsPresent := decodeKeywordGroupsFromRequest(raw, "keywords")
	excludeKeywords, excludePresent := decodeKeywordGroupsFromRequest(raw, "excludeKeywords")
	// An omitted users array leaves it unchanged; an explicit null does too,
	// matching upstream's skip-undefined merge.
	if _, present := raw["users"]; !present {
		body.Users = nil
	}

	a, err := s.accounts.UpdateAntenna(r.Context(), body.AntennaID, info.User.ID, &account.AntennaPatch{
		Name:                    body.Name,
		Src:                     body.Src,
		UserListID:              body.UserListID,
		Users:                   body.Users,
		Keywords:                keywords,
		KeywordsPresent:         keywordsPresent,
		ExcludeKeywords:         excludeKeywords,
		ExcludeKeywordsPresent:  excludePresent,
		CaseSensitive:           body.CaseSensitive,
		ExcludeBots:             body.ExcludeBots,
		WithReplies:             body.WithReplies,
		WithFile:                body.WithFile,
		LocalOnly:               body.LocalOnly,
		ExcludeSensitiveChannel: body.ExcludeSensitiveChannel,
	})
	if err != nil {
		writeAntennaError(w, err, antennaErrorIDs{
			noSuchAntenna:  errIDNoSuchAntennaUpdate,
			noSuchUserList: errIDNoSuchUserListUpdate,
			emptyKeyword:   errIDEmptyKeywordUpdate,
		})
		return
	}
	writeJSON(w, http.StatusOK, s.packAntenna(a))
}

// handleAntennasDelete implements POST /api/antennas/delete.
func (s *Server) handleAntennasDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AntennaID string `json:"antennaId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireAntennaCaller(w, r)
	if info == nil {
		return
	}
	if body.AntennaID == "" {
		writeAPIError(w, invalidParamError("antennaId is required"))
		return
	}
	if err := s.accounts.DeleteAntenna(r.Context(), body.AntennaID, info.User.ID); err != nil {
		writeAntennaError(w, err, antennaErrorIDs{noSuchAntenna: errIDNoSuchAntennaDelete})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAntennasList implements POST /api/antennas/list.
func (s *Server) handleAntennasList(w http.ResponseWriter, r *http.Request) {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	antennas, err := s.accounts.ListAntennas(r.Context(), info.User.ID, 0)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(antennas))
	for _, a := range antennas {
		out = append(out, s.packAntenna(a))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAntennasShow implements POST /api/antennas/show.
func (s *Server) handleAntennasShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AntennaID string `json:"antennaId"`
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
	if body.AntennaID == "" {
		writeAPIError(w, invalidParamError("antennaId is required"))
		return
	}
	a, err := s.accounts.GetAntenna(r.Context(), body.AntennaID, info.User.ID)
	if err != nil {
		writeAntennaError(w, err, antennaErrorIDs{noSuchAntenna: errIDNoSuchAntennaShow})
		return
	}
	writeJSON(w, http.StatusOK, s.packAntenna(a))
}

// handleAntennasNotes implements POST /api/antennas/notes.
func (s *Server) handleAntennasNotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AntennaID string `json:"antennaId"`
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
	if body.AntennaID == "" {
		writeAPIError(w, invalidParamError("antennaId is required"))
		return
	}
	// Ownership check first, as upstream loads the antenna by id+userId.
	if _, err := s.accounts.GetAntenna(r.Context(), body.AntennaID, info.User.ID); err != nil {
		writeAntennaError(w, err, antennaErrorIDs{noSuchAntenna: errIDNoSuchAntennaNotes})
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
	notes, err := s.timeline.AntennaNotes(r.Context(), body.AntennaID, info.User.ID, body.Limit, body.SinceID, body.UntilID)
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

// handleAntennasRemoveNote implements POST /api/antennas/remove-note,
// which drops one note from an antenna's collection.
func (s *Server) handleAntennasRemoveNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AntennaID string `json:"antennaId"`
		NoteID    string `json:"noteId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireAntennaCaller(w, r)
	if info == nil {
		return
	}
	if body.AntennaID == "" || body.NoteID == "" {
		writeAPIError(w, invalidParamError("antennaId and noteId are required"))
		return
	}
	if _, err := s.accounts.GetAntenna(r.Context(), body.AntennaID, info.User.ID); err != nil {
		writeAntennaError(w, err, antennaErrorIDs{noSuchAntenna: errIDNoSuchAntennaNotes})
		return
	}
	if err := s.accounts.RemoveAntennaNote(r.Context(), body.AntennaID, body.NoteID); err != nil {
		writeInternalServerError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ingestAntennas runs a created note past every antenna and publishes the
// streaming event for each hit. It is called for local creates and, through
// the inbox hook, for inbound remote notes, matching upstream's
// postNoteCreated.
func (s *Server) ingestAntennas(ctx context.Context, n *note.Note, authorID string) {
	if s.accounts == nil || n == nil {
		return
	}
	author, err := s.accounts.GetByID(ctx, n.UserID)
	if err != nil || author == nil {
		return
	}
	var isBot bool
	if err := s.pool.QueryRow(ctx,
		`SELECT is_bot FROM users WHERE id = $1`, n.UserID).Scan(&isBot); err != nil {
		return
	}
	an := account.AntennaNote{
		ID:          n.ID,
		AuthorID:    n.UserID,
		AuthorAcct:  authorAcct(author),
		AuthorIsBot: isBot,
		Visibility:  n.Visibility,
		ReplyID:     n.ReplyID,
		Text:        n.Text,
		CW:          n.CW,
		FileIDs:     n.FileIDs,
	}
	if an.Visibility == "specified" {
		an.VisibleUsers = s.notes.VisibleUserIDs(ctx, n.ID)
	}
	// The channel-sensitivity exclusion is real now that channels exist:
	// a sensitive channel's notes are dropped by antennas that ask for it.
	if n.ChannelID != nil {
		var sensitive bool
		if err := s.pool.QueryRow(ctx,
			`SELECT is_sensitive FROM channels WHERE id = $1`, *n.ChannelID).Scan(&sensitive); err == nil {
			an.InSensitiveChannel = sensitive
		}
	}
	matched, err := s.accounts.IngestNote(ctx, an)
	if err != nil || len(matched) == 0 {
		return
	}
	packed := s.packNoteFor(n, "")
	for _, antennaID := range matched {
		s.publishAntennaEvent(antennaID, packed)
	}
}

// authorAcct renders @user (local) or @user@host (remote).
func authorAcct(u *account.User) string {
	if u.Host == nil || *u.Host == "" {
		return "@" + u.UsernameLower
	}
	return "@" + u.UsernameLower + "@" + *u.Host
}
