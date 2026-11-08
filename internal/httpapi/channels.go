package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// Misskey error ids for the channels endpoints, copied from
// packages/backend/src/server/api/endpoints/channels/*.ts. NO_SUCH_CHANNEL
// carries a different id per endpoint, as upstream does.
const (
	errIDNoSuchFileChannelCreate = "cd1e9f3e-5a12-4ab4-96f6-5d0a2cc32050"
	errIDNoSuchChannelUpdate     = "f9c5467f-d492-4c3c-9a8d-a70dacc86512"
	errIDChannelAccessDenied     = "1fb7cb09-d46a-4fdf-b8df-057788cce513"
	errIDNoSuchFileChannelUpdate = "e86c14a4-0da2-4032-8df3-e737a04c7f3b"
	errIDNoSuchChannelShow       = "6f6c314b-7486-4897-8966-c04a66a02923"
	errIDNoSuchChannelTimeline   = "4d0eeeba-a02c-4c3c-9966-ef60d38d2e7f"
	errIDNoSuchChannelFavorite   = "4938f5f3-6167-4c04-9149-6607b7542861"
	errIDNoSuchChannelUnfavorite = "353c68dd-131a-476c-aa99-88a345e83668"
	errIDNoSuchChannelFollow     = "c0031718-d573-4e85-928e-10039f1fbb68"
	errIDAlreadyFollowingChannel = "7db31665-651e-40c1-8e6e-28e9ad829a2d"
	errIDNoSuchChannelUnfollow   = "19959ee9-0153-4c51-bbd9-a98c49dc59d6"
	errIDNoSuchChannelMuteCreate = "7174361e-d58f-31d6-2e7c-6fb830786a3f"
	errIDAlreadyMutingChannel    = "5a251978-769a-da44-3e89-3931e43bb592"
	errIDMuteExpiresInPast       = "42b32236-df2c-a45f-fdbf-def67268f749"
	errIDNoSuchChannelMuteDelete = "e7998769-6e94-d9c2-6b8f-94a527314aba"
	errIDNotMutingChannel        = "14d55962-6ea8-d990-1333-d6bef78dc2ab"
	// notes/create's mapping of the two channel errors.
	errIDNoteNoSuchChannel            = "b1653923-5453-4edc-b786-7c4f39bb0bbb"
	errIDCannotRenoteOutsideOfChannel = "33510210-8452-094c-6227-4a6c05d99f00"
	errIDChannelAlreadyFavorited      = "4938f5f3-6167-4c04-9149-6607b7542861"
)

// limitChannelsCreate is Misskey's channels/create rate cap.
var limitChannelsCreate = rateLimit{bucket: "channels/create", max: 10, duration: time.Hour}

// channelErrorIDs carries the per-endpoint error ids of the channel family.
type channelErrorIDs struct {
	noSuchChannel string
	noSuchFile    string
}

// packChannelRef is the embedded channel reference of a packed note.
func packChannelRef(c *account.Channel) map[string]any {
	var userID any
	if c.UserID != nil {
		userID = *c.UserID
	}
	return map[string]any{
		"id":                    c.ID,
		"name":                  c.Name,
		"color":                 c.Color,
		"isSensitive":           c.IsSensitive,
		"allowRenoteToExternal": c.AllowRenoteToExternal,
		"userId":                userID,
	}
}

// packChannel renders Misskey's Channel schema. viewerID fills the
// isFollowing/isFavorited/isMuting flags when a viewer is known.
func (s *Server) packChannel(ctx context.Context, c *account.Channel, viewerID string) (map[string]any, error) {
	var userID, bannerID, bannerURL, lastNotedAt any
	if c.UserID != nil {
		userID = *c.UserID
	}
	if c.BannerID != nil {
		bannerID = *c.BannerID
		if s.drive != nil {
			if file, err := s.drive.Get(ctx, *c.BannerID); err == nil && file != nil {
				bannerURL = file.URL
			}
		}
	}
	if c.LastNotedAt != nil {
		lastNotedAt = c.LastNotedAt.UTC().Format(time.RFC3339Nano)
	}
	pinned := c.PinnedNoteIDs
	if pinned == nil {
		pinned = []string{}
	}
	packed := map[string]any{
		"id":                    c.ID,
		"createdAt":             c.CreatedAt.UTC().Format(time.RFC3339Nano),
		"lastNotedAt":           lastNotedAt,
		"name":                  c.Name,
		"description":           c.Description,
		"userId":                userID,
		"bannerUrl":             bannerURL,
		"bannerId":              bannerID,
		"pinnedNoteIds":         pinned,
		"color":                 c.Color,
		"isArchived":            c.IsArchived,
		"usersCount":            c.UsersCount,
		"notesCount":            c.NotesCount,
		"isSensitive":           c.IsSensitive,
		"allowRenoteToExternal": c.AllowRenoteToExternal,
	}
	if viewerID != "" {
		following, err := s.accounts.IsFollowingChannel(ctx, viewerID, c.ID)
		if err != nil {
			return nil, err
		}
		favorited, err := s.accounts.HasChannelFavorite(ctx, viewerID, c.ID)
		if err != nil {
			return nil, err
		}
		muting, err := s.accountMutesChannel(ctx, viewerID, c.ID)
		if err != nil {
			return nil, err
		}
		packed["isFollowing"] = following
		packed["isFavorited"] = favorited
		packed["isMuting"] = muting
	}
	return packed, nil
}

// accountMutesChannel reports whether the viewer currently mutes a channel.
func (s *Server) accountMutesChannel(ctx context.Context, userID, channelID string) (bool, error) {
	muted, err := s.accounts.MutedChannelIDsOf(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, id := range muted {
		if id == channelID {
			return true, nil
		}
	}
	return false, nil
}

// writeChannelError maps the channel service sentinel errors onto
// Misskey's envelope.
func writeChannelError(w http.ResponseWriter, err error, ids channelErrorIDs) {
	switch {
	case errors.Is(err, account.ErrChannelNotFound):
		writeAPIError(w, &APIError{
			Message: "No such channel.", Code: "NO_SUCH_CHANNEL",
			ID: ids.noSuchChannel, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, errChannelAccessDenied):
		writeAPIError(w, &APIError{
			Message: "Access denied.", Code: "ACCESS_DENIED",
			ID: errIDChannelAccessDenied, Kind: "permission", Status: http.StatusForbidden,
		})
	case errors.Is(err, account.ErrNoSuchBannerFile):
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: ids.noSuchFile, Kind: "client", Status: http.StatusNotFound,
		})
	case errors.Is(err, account.ErrChannelNameInvalid):
		writeAPIError(w, invalidParamError("name must be 1..128 characters"))
	case errors.Is(err, account.ErrChannelDescInvalid):
		writeAPIError(w, invalidParamError("description must be at most 2048 characters"))
	case errors.Is(err, account.ErrAlreadyFollowingChannel):
		writeAPIError(w, &APIError{
			Message: "You are already following that channel.", Code: "ALREADY_FOLLOWING",
			ID: errIDAlreadyFollowingChannel, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrChannelAlreadyFavorited):
		writeAPIError(w, &APIError{
			Message: "The channel has already been favorited.", Code: "ALREADY_FAVORITED",
			ID: errIDChannelAlreadyFavorited, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrAlreadyMutingChannel):
		writeAPIError(w, &APIError{
			Message: "You are already muting that channel.", Code: "ALREADY_MUTING_CHANNEL",
			ID: errIDAlreadyMutingChannel, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrNotMutingChannel):
		writeAPIError(w, &APIError{
			Message: "You are not muting that channel.", Code: "NOT_MUTING_CHANNEL",
			ID: errIDNotMutingChannel, Kind: "client", Status: http.StatusBadRequest,
		})
	case errors.Is(err, account.ErrMuteExpiresInPast):
		writeAPIError(w, &APIError{
			Message: "expiresAt is in the past.", Code: "EXPIRES_AT_IS_PAST",
			ID: errIDMuteExpiresInPast, Kind: "client", Status: http.StatusBadRequest,
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

var errChannelAccessDenied = account.ErrChannelAccessDenied()

// requireChannelCaller authenticates a channel mutation. prohibitMoved is
// passed separately because not every endpoint declares it.
func (s *Server) requireChannelCaller(w http.ResponseWriter, r *http.Request, prohibitMoved bool) *AuthInfo {
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return nil
	}
	if prohibitMoved && info.User.MovedToURI != "" {
		writeChannelError(w, account.ErrAccountMoved, channelErrorIDs{})
		return nil
	}
	return info
}

// handleChannelsCreate implements POST /api/channels/create.
func (s *Server) handleChannelsCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name                  string  `json:"name"`
		Description           *string `json:"description"`
		BannerID              *string `json:"bannerId"`
		Color                 string  `json:"color"`
		IsSensitive           bool    `json:"isSensitive"`
		AllowRenoteToExternal *bool   `json:"allowRenoteToExternal"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireChannelCaller(w, r, true)
	if info == nil {
		return
	}
	if !s.checkRateLimit(w, info.User.ID, limitChannelsCreate) {
		return
	}
	allowExternal := true
	if body.AllowRenoteToExternal != nil {
		allowExternal = *body.AllowRenoteToExternal
	}
	c, err := s.accounts.CreateChannel(r.Context(), info.User.ID, account.ChannelInput{
		Name:                  body.Name,
		Description:           body.Description,
		BannerID:              body.BannerID,
		Color:                 body.Color,
		IsSensitive:           body.IsSensitive,
		AllowRenoteToExternal: allowExternal,
	})
	if err != nil {
		writeChannelError(w, err, channelErrorIDs{noSuchFile: errIDNoSuchFileChannelCreate})
		return
	}
	packed, err := s.packChannel(r.Context(), c, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleChannelsUpdate implements POST /api/channels/update.
func (s *Server) handleChannelsUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID             string   `json:"channelId"`
		Name                  *string  `json:"name"`
		Description           *string  `json:"description"`
		BannerID              *string  `json:"bannerId"`
		IsArchived            *bool    `json:"isArchived"`
		PinnedNoteIDs         []string `json:"pinnedNoteIds"`
		Color                 *string  `json:"color"`
		IsSensitive           *bool    `json:"isSensitive"`
		AllowRenoteToExternal *bool    `json:"allowRenoteToExternal"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireChannelCaller(w, r, false)
	if info == nil {
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	raw := bodyAsMap(r)
	_, descriptionPresent := raw["description"]
	_, bannerPresent := raw["bannerId"]

	patch := &account.ChannelPatch{
		Name:                  body.Name,
		Description:           body.Description,
		DescriptionPresent:    descriptionPresent,
		BannerID:              body.BannerID,
		BannerIDPresent:       bannerPresent,
		IsArchived:            body.IsArchived,
		PinnedNoteIDs:         body.PinnedNoteIDs,
		Color:                 body.Color,
		IsSensitive:           body.IsSensitive,
		AllowRenoteToExternal: body.AllowRenoteToExternal,
	}
	c, err := s.accounts.UpdateChannel(r.Context(), body.ChannelID, info.User.ID, info.User.IsAdmin, patch)
	if err != nil {
		writeChannelError(w, err, channelErrorIDs{
			noSuchChannel: errIDNoSuchChannelUpdate,
			noSuchFile:    errIDNoSuchFileChannelUpdate,
		})
		return
	}
	packed, err := s.packChannel(r.Context(), c, info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packed)
}

// handleChannelsShow implements POST /api/channels/show. The response
// carries pinnedNotes, which only this endpoint packs.
func (s *Server) handleChannelsShow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID string `json:"channelId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	c, err := s.accounts.GetChannel(r.Context(), body.ChannelID)
	if err != nil {
		writeChannelError(w, err, channelErrorIDs{noSuchChannel: errIDNoSuchChannelShow})
		return
	}
	packed, err := s.packChannel(r.Context(), c, viewerID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	packed["pinnedNotes"] = s.packPinnedNotes(r.Context(), c.PinnedNoteIDs, viewerID)
	writeJSON(w, http.StatusOK, packed)
}

// packPinnedNotes renders the pinned notes of a channel in pin order,
// silently skipping ids whose notes are gone, as upstream's findBy does.
func (s *Server) packPinnedNotes(ctx context.Context, ids []string, viewerID string) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		n, err := s.notes.Get(ctx, id)
		if err != nil || n == nil {
			continue
		}
		pc := s.notePackContext(ctx, n, viewerID)
		out = append(out, packNoteDetail(ctx, s, n, pc))
	}
	return out
}

// handleChannelsTimeline implements POST /api/channels/timeline.
func (s *Server) handleChannelsTimeline(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID    string `json:"channelId"`
		Limit        int    `json:"limit"`
		SinceID      string `json:"sinceId"`
		UntilID      string `json:"untilId"`
		SinceDate    *int64 `json:"sinceDate"`
		UntilDate    *int64 `json:"untilDate"`
		AllowPartial bool   `json:"allowPartial"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	if _, err := s.accounts.GetChannel(r.Context(), body.ChannelID); err != nil {
		writeChannelError(w, err, channelErrorIDs{noSuchChannel: errIDNoSuchChannelTimeline})
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
	var muted []string
	if viewerID != "" {
		muted, _ = s.accounts.MutedChannelIDsOf(r.Context(), viewerID)
		// The subscribed channel itself always delivers, even if muted.
		filtered := muted[:0]
		for _, id := range muted {
			if id != body.ChannelID {
				filtered = append(filtered, id)
			}
		}
		muted = filtered
	}
	notes, err := s.timeline.ChannelNotes(r.Context(), body.ChannelID, viewerID, body.Limit, body.SinceID, body.UntilID, muted)
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

// handleChannelsFavorite / Unfavorite / Follow / Unfollow / Mute... share
// their shape: a channelId plus one write.
func (s *Server) handleChannelsFavorite(w http.ResponseWriter, r *http.Request) {
	s.serveChannelRelation(w, r, "favorite")
}
func (s *Server) handleChannelsUnfavorite(w http.ResponseWriter, r *http.Request) {
	s.serveChannelRelation(w, r, "unfavorite")
}
func (s *Server) handleChannelsFollow(w http.ResponseWriter, r *http.Request) {
	s.serveChannelRelation(w, r, "follow")
}
func (s *Server) handleChannelsUnfollow(w http.ResponseWriter, r *http.Request) {
	s.serveChannelRelation(w, r, "unfollow")
}

func (s *Server) serveChannelRelation(w http.ResponseWriter, r *http.Request, action string) {
	var body struct {
		ChannelID string `json:"channelId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireChannelCaller(w, r, true)
	if info == nil {
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	var err error
	ids := channelErrorIDs{}
	switch action {
	case "favorite":
		ids.noSuchChannel = errIDNoSuchChannelFavorite
		err = s.accounts.FavoriteChannel(r.Context(), info.User.ID, body.ChannelID)
	case "unfavorite":
		ids.noSuchChannel = errIDNoSuchChannelUnfavorite
		err = s.accounts.UnfavoriteChannel(r.Context(), info.User.ID, body.ChannelID)
	case "follow":
		ids.noSuchChannel = errIDNoSuchChannelFollow
		err = s.accounts.FollowChannel(r.Context(), info.User.ID, body.ChannelID)
	case "unfollow":
		ids.noSuchChannel = errIDNoSuchChannelUnfollow
		err = s.accounts.UnfollowChannel(r.Context(), info.User.ID, body.ChannelID)
	}
	if err != nil {
		writeChannelError(w, err, ids)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleChannelsFollowed implements POST /api/channels/followed.
func (s *Server) handleChannelsFollowed(w http.ResponseWriter, r *http.Request) {
	s.serveChannelList(w, r, "followed")
}

// handleChannelsOwned implements POST /api/channels/owned.
func (s *Server) handleChannelsOwned(w http.ResponseWriter, r *http.Request) {
	s.serveChannelList(w, r, "owned")
}

// handleChannelsMyFavorites implements POST /api/channels/my-favorites.
func (s *Server) handleChannelsMyFavorites(w http.ResponseWriter, r *http.Request) {
	s.serveChannelList(w, r, "my-favorites")
}

// handleChannelsMuted implements POST /api/channels/mute/list.
func (s *Server) handleChannelsMuted(w http.ResponseWriter, r *http.Request) {
	s.serveChannelList(w, r, "muted")
}

func (s *Server) serveChannelList(w http.ResponseWriter, r *http.Request, which string) {
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
	var channels []*account.Channel
	switch which {
	case "followed":
		channels, err = s.accounts.FollowedChannels(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID)
	case "owned":
		channels, err = s.accounts.OwnedChannels(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID)
	case "my-favorites":
		channels, err = s.accounts.FavoriteChannels(r.Context(), info.User.ID)
	case "muted":
		channels, err = s.accounts.MutedChannels(r.Context(), info.User.ID)
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(channels))
	for _, c := range channels {
		packed, err := s.packChannel(r.Context(), c, info.User.ID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleChannelsFeatured implements POST /api/channels/featured.
func (s *Server) handleChannelsFeatured(w http.ResponseWriter, r *http.Request) {
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	channels, err := s.accounts.FeaturedChannels(r.Context())
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(channels))
	for _, c := range channels {
		packed, err := s.packChannel(r.Context(), c, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleChannelsSearch implements POST /api/channels/search.
func (s *Server) handleChannelsSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query   string `json:"query"`
		Type    string `json:"type"`
		Limit   int    `json:"limit"`
		SinceID string `json:"sinceId"`
		UntilID string `json:"untilId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	viewerID := ""
	if info, err := s.authenticate(r.Context(), r, bodyAsMap(r)); err == nil && info != nil {
		viewerID = info.User.ID
	}
	channels, err := s.accounts.SearchChannels(r.Context(), body.Query, body.Type, body.Limit, body.SinceID, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(channels))
	for _, c := range channels {
		packed, err := s.packChannel(r.Context(), c, viewerID)
		if err != nil {
			writeInternalServerError(w)
			return
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleChannelsMuteCreate implements POST /api/channels/mute/create.
func (s *Server) handleChannelsMuteCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID string `json:"channelId"`
		ExpiresAt *int64 `json:"expiresAt"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireChannelCaller(w, r, true)
	if info == nil {
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		t := time.UnixMilli(*body.ExpiresAt)
		expiresAt = &t
	}
	if err := s.accounts.MuteChannel(r.Context(), info.User.ID, body.ChannelID, expiresAt); err != nil {
		writeChannelError(w, err, channelErrorIDs{noSuchChannel: errIDNoSuchChannelMuteCreate})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleChannelsMuteDelete implements POST /api/channels/mute/delete.
func (s *Server) handleChannelsMuteDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChannelID string `json:"channelId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info := s.requireChannelCaller(w, r, true)
	if info == nil {
		return
	}
	if body.ChannelID == "" {
		writeAPIError(w, invalidParamError("channelId is required"))
		return
	}
	if err := s.accounts.UnmuteChannel(r.Context(), info.User.ID, body.ChannelID); err != nil {
		writeChannelError(w, err, channelErrorIDs{noSuchChannel: errIDNoSuchChannelMuteDelete})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
