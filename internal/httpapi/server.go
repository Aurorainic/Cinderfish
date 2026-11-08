// Package httpapi assembles CinderFish's HTTP surface: the Misskey-style
// RPC endpoints under /api, ActivityPub endpoints, and static frontend
// serving. M1 covers /api/meta and /api/ping.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/announcement"
	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/mail"
	"github.com/Aurorainic/cinderfish/internal/media"
	"github.com/Aurorainic/cinderfish/internal/note"
	"github.com/Aurorainic/cinderfish/internal/report"
	"github.com/Aurorainic/cinderfish/internal/streaming"
	"github.com/Aurorainic/cinderfish/internal/webhook"
)

// Server holds dependencies shared by all handlers.
type Server struct {
	cfg       *config.Config
	startedAt time.Time
	mux       *http.ServeMux

	pool     *pgxpool.Pool
	accounts *account.Service
	tokens   *account.TokenService

	notes         *note.Service
	timeline      *note.TimelineService
	noteReactions *note.ReactionService
	notifications *note.NotificationService
	favorites     *note.FavoriteService
	hashtags      *note.HashtagService
	folders       *media.FolderService
	emojis        *media.EmojiService
	registry      *account.RegistryService
	announcements *announcement.Service
	reports       *report.Service
	webhooks      *webhook.Service

	storage media.Storage
	drive   *media.DriveService

	// mail delivers verification and password reset links. It is never nil:
	// an instance without SMTP configured logs messages instead.
	mail mail.Sender

	// captchaHTTP performs the outbound verification calls. The endpoints
	// are fixed (internal/captcha); tests inject a stub transport.
	captchaHTTP *http.Client

	// web serves the built Vue frontend. It is nil when no build is
	// present, so API-only deployments and tests still work.
	web *staticSite

	// streamBus carries Streaming API events. It is always present so
	// handlers can publish unconditionally; stream is nil until the
	// server is attached to a running listener.
	streamBus *streaming.Bus
	stream    *streaming.Server

	// activity records the last authenticated request of each user, which is
	// what get-online-users-count reports.
	activity *activityTracker

	// rateLimiter enforces Misskey's per-endpoint write caps.
	rateLimiter *rateLimiter

	// ActivityPub. federationPolicy, fedRenderer and federationStore are
	// populated by AttachFederation; the greenlet handlers check for nil.
	federationPolicy     *federation.Policy
	fedRenderer          *federation.Renderer
	federationStore      *federation.Store
	federationClient     *federation.Client
	federationHTTPClient *http.Client
	federationInbox      *federation.Inbox
	federationDeliver    *federation.Deliverer
}

// NewServer returns a Server with no database backing. It serves the
// endpoints that do not need persistence.
func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:         cfg,
		startedAt:   time.Now(),
		streamBus:   streaming.NewBus(),
		activity:    newActivityTracker(),
		rateLimiter: newRateLimiter(),
		captchaHTTP: &http.Client{Timeout: 10 * time.Second},
	}
	if cfg != nil {
		s.mail = mail.NewSwitchableSender(mail.New(mail.Settings{
			Host:     cfg.Mail.Host,
			Port:     cfg.Mail.Port,
			Username: cfg.Mail.Username,
			Password: cfg.Mail.Password,
			From:     cfg.Mail.From,
			TLS:      cfg.Mail.TLS,
		}))
	}
	if site, err := newStaticSite(cfg.WebDir); err == nil {
		s.web = site
	}
	s.mux = http.NewServeMux()
	s.registerRoutes()
	return s
}

// NewServerWithDB returns a Server wired to the given database pool.
func NewServerWithDB(cfg *config.Config, pool *pgxpool.Pool) *Server {
	s := NewServer(cfg)
	s.pool = pool
	s.accounts = account.NewService(pool)
	s.tokens = account.NewTokenService(pool)
	s.notes = note.NewService(pool)
	// Runtime instance settings are read per request; the providers keep the
	// account and note services in step with the stored values without
	// importing the httpapi package into them.
	s.accounts.SetReservedUsernames(func() []string {
		return s.livePreservedUsernames(context.Background())
	})
	s.notes.SetMaxTextLength(func() int {
		return s.liveMaxNoteTextLength(context.Background())
	})
	s.timeline = note.NewTimelineService(pool)
	s.noteReactions = note.NewReactionService(pool)
	s.notifications = note.NewNotificationService(pool)
	s.favorites = note.NewFavoriteService(pool)
	s.hashtags = note.NewHashtagService(pool)
	s.folders = media.NewFolderService(pool)
	s.emojis = media.NewEmojiService(pool)
	s.registry = account.NewRegistryService(pool)
	s.announcements = announcement.NewService(pool)
	s.reports = report.NewService(pool)
	s.webhooks = webhook.NewService(pool)
	// A persisted SMTP override from the admin console wins over config.toml.
	s.applyStoredMail()
	return s
}

// SetStorage attaches the media backend and the drive service. The backend
// is wrapped in a switchable facade so the admin console can replace it at
// runtime without rebuilding the drive service; a persisted override from
// instance_meta (object storage page) wins over the passed-in backend.
func (s *Server) SetStorage(storage media.Storage) {
	s.storage = media.NewSwitchableStorage(storage)
	if s.pool != nil {
		s.drive = media.NewDriveService(s.pool, s.storage)
		if s.accounts != nil {
			s.accounts.AttachDrive(s.drive)
		}
	}
	s.applyStoredStorage()
}

// AllowPrivateWebhookTargets disables the public-address validation of
// webhook URLs. The HTTP tests deliver against loopback servers; a real
// deployment must keep the default.
func (s *Server) AllowPrivateWebhookTargets() {
	if s.webhooks != nil {
		s.webhooks.AllowPrivateTargets = true
	}
}

// SetCaptchaHTTPClient replaces the client used for captcha verification.
// Tests inject a stub transport; the verification endpoints themselves stay
// fixed in internal/captcha.
func (s *Server) SetCaptchaHTTPClient(client *http.Client) {
	if client != nil {
		s.captchaHTTP = client
	}
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /api/meta", s.handleMeta)
	s.mux.HandleFunc("POST /api/stats", s.handleStats)
	s.mux.HandleFunc("POST /api/get-online-users-count", s.handleGetOnlineUsersCount)
	s.mux.HandleFunc("GET /api/get-online-users-count", s.handleGetOnlineUsersCount)
	s.mux.HandleFunc("POST /api/charts/user/notes", s.handleChartsUserNotes)
	s.mux.HandleFunc("GET /api/charts/user/notes", s.handleChartsUserNotes)
	s.mux.HandleFunc("POST /api/server-info", s.handleServerInfo)
	s.mux.HandleFunc("GET /api/server-info", s.handleServerInfo)
	s.mux.HandleFunc("POST /api/fetch-rss", s.handleFetchRSS)
	s.mux.HandleFunc("GET /api/fetch-rss", s.handleFetchRSS)
	s.mux.HandleFunc("POST /api/get-sounds", s.handleGetSounds)
	s.mux.HandleFunc("GET /api/get-sounds", s.handleGetSounds)
	s.mux.HandleFunc("POST /api/ping", s.handlePing)
	s.mux.HandleFunc("GET /api/ping", s.handlePing)

	s.mux.HandleFunc("POST /api/signup", s.handleSignup)
	s.mux.HandleFunc("POST /api/signin", s.handleSignin)
	s.mux.HandleFunc("POST /api/username/available", s.handleUsernameAvailable)
	s.mux.HandleFunc("POST /api/i", s.handleI)
	s.mux.HandleFunc("POST /api/email-address/available", s.handleEmailAddressAvailable)
	s.mux.HandleFunc("POST /api/request-reset-password", s.handleRequestResetPassword)
	s.mux.HandleFunc("POST /api/reset-password", s.handleResetPassword)
	s.mux.HandleFunc("POST /api/verify-email", s.handleVerifyEmail)
	s.mux.HandleFunc("POST /api/i/email-verification", s.handleIEmailVerification)
	s.mux.HandleFunc("POST /api/i/update-email", s.handleIUpdateEmail)
	s.mux.HandleFunc("POST /api/i/change-password", s.handleIChangePassword)
	s.mux.HandleFunc("POST /api/i/regenerate-token", s.handleIRegenerateToken)
	s.mux.HandleFunc("POST /api/i/signin-history", s.handleISigninHistory)
	s.mux.HandleFunc("POST /api/i/update", s.handleIUpdate)
	// Misskey keeps users/update as the canonical spelling and i/update as
	// its alias; both write the same fields.
	s.mux.HandleFunc("POST /api/users/update", s.handleIUpdate)
	s.mux.HandleFunc("POST /api/i/2fa/register", s.handleI2FARegister)
	s.mux.HandleFunc("POST /api/i/2fa/done", s.handleI2FADone)
	s.mux.HandleFunc("POST /api/i/2fa/unregister", s.handleI2FAUnregister)
	s.mux.HandleFunc("POST /api/i/2fa/password-less", s.handleI2FAPasswordLess)
	s.mux.HandleFunc("POST /api/i/pin", s.handleIPin)
	s.mux.HandleFunc("POST /api/i/unpin", s.handleIUnpin)
	// Account migration: the alias list a move is validated against, and the
	// move itself.
	s.mux.HandleFunc("POST /api/i/known-as", s.handleIKnownAs)
	s.mux.HandleFunc("POST /api/i/move", s.handleIMove)
	// Federation lookup: resolves an ActivityPub URL to a note or a user.
	s.mux.HandleFunc("POST /api/ap/show", s.handleAPShow)
	s.mux.HandleFunc("POST /api/users", s.handleUsers)
	s.mux.HandleFunc("POST /api/users/show", s.handleUsersShow)
	s.mux.HandleFunc("POST /api/users/search", s.handleUsersSearch)
	s.mux.HandleFunc("POST /api/users/report-abuse", s.handleReportAbuse)
	s.mux.HandleFunc("POST /api/users/notes", s.handleUsersNotes)
	s.mux.HandleFunc("POST /api/users/relation", s.handleUsersRelation)
	s.mux.HandleFunc("POST /api/users/get-frequently-replied-users", s.handleUsersFrequentlyReplied)
	s.mux.HandleFunc("POST /api/users/stats", s.handleUsersStats)
	s.mux.HandleFunc("POST /api/users/reactions", s.handleUsersReactions)
	s.mux.HandleFunc("POST /api/users/followers", s.handleUsersFollowers)
	s.mux.HandleFunc("POST /api/users/following", s.handleUsersFollowing)
	s.mux.HandleFunc("POST /api/miauth/gen-token", s.handleMiAuthGenToken)
	s.mux.HandleFunc("POST /api/miauth/{session}/check", s.handleMiAuthCheck)
	s.mux.HandleFunc("POST /api/i/apps", s.handleApps)
	// Application registration and the legacy authorization session flow:
	// an app exchanges its secret for an approval link, the user approves it,
	// and the app polls for the token.
	s.mux.HandleFunc("POST /api/app/create", s.handleAppCreate)
	s.mux.HandleFunc("POST /api/app/show", s.handleAppShow)
	s.mux.HandleFunc("POST /api/auth/session/generate", s.handleAuthSessionGenerate)
	s.mux.HandleFunc("POST /api/auth/session/show", s.handleAuthSessionShow)
	s.mux.HandleFunc("POST /api/auth/session/userkey", s.handleAuthSessionUserKey)
	s.mux.HandleFunc("POST /api/auth/accept", s.handleAuthAccept)
	s.mux.HandleFunc("POST /api/i/revoke-token", s.handleRevokeToken)
	s.mux.HandleFunc("POST /api/i/export", s.handleIExport)
	s.mux.HandleFunc("POST /api/i/import", s.handleIImport)
	s.mux.HandleFunc("POST /api/i/delete-account", s.handleIDeleteAccount)
	s.mux.HandleFunc("POST /api/i/webhooks/create", s.handleWebhookCreate)
	s.mux.HandleFunc("POST /api/i/webhooks/list", s.handleWebhookList)
	s.mux.HandleFunc("POST /api/i/webhooks/show", s.handleWebhookShow)
	s.mux.HandleFunc("POST /api/i/webhooks/update", s.handleWebhookUpdate)
	s.mux.HandleFunc("POST /api/i/webhooks/delete", s.handleWebhookDelete)

	// Following. These give the Streaming API's follow/followed events a
	// source, which M4 left as a known gap.
	s.mux.HandleFunc("POST /api/following/create", s.handleFollowingCreate)
	s.mux.HandleFunc("POST /api/following/delete", s.handleFollowingDelete)
	s.mux.HandleFunc("POST /api/following/requests/list", s.handleFollowingRequestsList)
	s.mux.HandleFunc("POST /api/following/requests/sent", s.handleFollowingRequestsSent)
	s.mux.HandleFunc("POST /api/following/requests/accept", s.handleFollowingRequestsAccept)
	s.mux.HandleFunc("POST /api/following/requests/reject", s.handleFollowingRequestsReject)
	s.mux.HandleFunc("POST /api/following/requests/cancel", s.handleFollowingRequestsCancel)
	// Per-follow settings and the forced unfollow of a follower.
	s.mux.HandleFunc("POST /api/following/list", s.handleFollowingList)
	s.mux.HandleFunc("POST /api/following/update", s.handleFollowingUpdate)
	s.mux.HandleFunc("POST /api/following/update-all", s.handleFollowingUpdateAll)
	s.mux.HandleFunc("POST /api/following/invalidate", s.handleFollowingInvalidate)

	// User lists (schema v30), the users/lists/* family upstream defines
	// in packages/backend/src/server/api/endpoints/users/lists/.
	s.mux.HandleFunc("POST /api/users/lists/create", s.handleUserListsCreate)
	s.mux.HandleFunc("POST /api/users/lists/create-from-public", s.handleUserListsCreateFromPublic)
	s.mux.HandleFunc("POST /api/users/lists/delete", s.handleUserListsDelete)
	s.mux.HandleFunc("POST /api/users/lists/list", s.handleUserListsList)
	s.mux.HandleFunc("POST /api/users/lists/show", s.handleUserListsShow)
	s.mux.HandleFunc("POST /api/users/lists/update", s.handleUserListsUpdate)
	s.mux.HandleFunc("POST /api/users/lists/push", s.handleUserListsPush)
	s.mux.HandleFunc("POST /api/users/lists/pull", s.handleUserListsPull)
	s.mux.HandleFunc("POST /api/users/lists/update-membership", s.handleUserListsUpdateMembership)
	s.mux.HandleFunc("POST /api/users/lists/get-memberships", s.handleUserListsGetMemberships)
	s.mux.HandleFunc("POST /api/users/lists/favorite", s.handleUserListsFavorite)
	s.mux.HandleFunc("POST /api/users/lists/unfavorite", s.handleUserListsUnfavorite)
	// The timeline that joins a list's members.
	s.mux.HandleFunc("POST /api/notes/user-list-timeline", s.handleUserListTimeline)

	// Clips (schema v31), the clips/* family upstream defines in
	// packages/backend/src/server/api/endpoints/clips/.
	s.mux.HandleFunc("POST /api/clips/create", s.handleClipsCreate)
	s.mux.HandleFunc("POST /api/clips/update", s.handleClipsUpdate)
	s.mux.HandleFunc("POST /api/clips/delete", s.handleClipsDelete)
	s.mux.HandleFunc("POST /api/clips/list", s.handleClipsList)
	s.mux.HandleFunc("POST /api/clips/show", s.handleClipsShow)
	s.mux.HandleFunc("POST /api/clips/my-favorites", s.handleClipsMyFavorites)
	s.mux.HandleFunc("POST /api/clips/favorite", s.handleClipsFavorite)
	s.mux.HandleFunc("POST /api/clips/unfavorite", s.handleClipsUnfavorite)
	s.mux.HandleFunc("POST /api/clips/add-note", s.handleClipsAddNote)
	s.mux.HandleFunc("POST /api/clips/remove-note", s.handleClipsRemoveNote)
	s.mux.HandleFunc("POST /api/clips/notes", s.handleClipsNotes)
	s.mux.HandleFunc("POST /api/notes/clips", s.handleNotesClips)
	s.mux.HandleFunc("POST /api/users/clips", s.handleUsersClips)

	// Antennas (schema v32), the antennas/* family upstream defines in
	// packages/backend/src/server/api/endpoints/antennas/. The timeline
	// endpoint is antennas/notes, not notes/antennas-timeline.
	s.mux.HandleFunc("POST /api/antennas/create", s.handleAntennasCreate)
	s.mux.HandleFunc("POST /api/antennas/update", s.handleAntennasUpdate)
	s.mux.HandleFunc("POST /api/antennas/delete", s.handleAntennasDelete)
	s.mux.HandleFunc("POST /api/antennas/list", s.handleAntennasList)
	s.mux.HandleFunc("POST /api/antennas/show", s.handleAntennasShow)
	s.mux.HandleFunc("POST /api/antennas/notes", s.handleAntennasNotes)
	s.mux.HandleFunc("POST /api/antennas/remove-note", s.handleAntennasRemoveNote)

	// Channels (schema v33), the channels/* family upstream defines in
	// packages/backend/src/server/api/endpoints/channels/. There is no
	// channels/delete: archiving through update is the deletion.
	s.mux.HandleFunc("POST /api/channels/create", s.handleChannelsCreate)
	s.mux.HandleFunc("POST /api/channels/update", s.handleChannelsUpdate)
	s.mux.HandleFunc("POST /api/channels/show", s.handleChannelsShow)
	s.mux.HandleFunc("POST /api/channels/timeline", s.handleChannelsTimeline)
	s.mux.HandleFunc("POST /api/channels/favorite", s.handleChannelsFavorite)
	s.mux.HandleFunc("POST /api/channels/unfavorite", s.handleChannelsUnfavorite)
	s.mux.HandleFunc("POST /api/channels/follow", s.handleChannelsFollow)
	s.mux.HandleFunc("POST /api/channels/unfollow", s.handleChannelsUnfollow)
	s.mux.HandleFunc("POST /api/channels/followed", s.handleChannelsFollowed)
	s.mux.HandleFunc("POST /api/channels/owned", s.handleChannelsOwned)
	s.mux.HandleFunc("POST /api/channels/my-favorites", s.handleChannelsMyFavorites)
	s.mux.HandleFunc("POST /api/channels/featured", s.handleChannelsFeatured)
	s.mux.HandleFunc("POST /api/channels/search", s.handleChannelsSearch)
	s.mux.HandleFunc("POST /api/channels/mute/create", s.handleChannelsMuteCreate)
	s.mux.HandleFunc("POST /api/channels/mute/delete", s.handleChannelsMuteDelete)
	s.mux.HandleFunc("POST /api/channels/mute/list", s.handleChannelsMuted)

	// Mutes and blocks. Both tables exist since schema v1; these endpoints
	// are what the settings page and the timeline filter read.
	s.mux.HandleFunc("POST /api/mute/create", s.handleMuteCreate)
	s.mux.HandleFunc("POST /api/mute/delete", s.handleMuteDelete)
	s.mux.HandleFunc("POST /api/mute/list", s.handleMuteList)
	s.mux.HandleFunc("POST /api/blocking/create", s.handleBlockingCreate)
	s.mux.HandleFunc("POST /api/blocking/delete", s.handleBlockingDelete)
	s.mux.HandleFunc("POST /api/blocking/list", s.handleBlockingList)

	// Federation administration. The mode and allowlist live in the
	// database so they change at runtime without a restart.
	s.mux.HandleFunc("POST /api/i/registry/set", s.handleRegistrySet)
	s.mux.HandleFunc("POST /api/i/registry/get", s.handleRegistryGet)
	s.mux.HandleFunc("POST /api/i/registry/get-detail", s.handleRegistryGetDetail)
	s.mux.HandleFunc("POST /api/i/registry/keys", s.handleRegistryKeys)
	s.mux.HandleFunc("POST /api/i/registry/keys-with-type", s.handleRegistryKeysWithType)
	s.mux.HandleFunc("POST /api/i/registry/get-all", s.handleRegistryGetAll)
	s.mux.HandleFunc("POST /api/i/registry/remove", s.handleRegistryRemove)
	s.mux.HandleFunc("POST /api/i/registry/scopes", s.handleRegistryScopes)
	s.mux.HandleFunc("POST /api/i/registry/clear", s.handleRegistryClear)
	s.mux.HandleFunc("POST /api/emojis", s.handleEmojis)
	s.mux.HandleFunc("POST /api/admin/emoji/list", s.handleAdminEmojiList)
	s.mux.HandleFunc("POST /api/admin/emoji/add", s.handleAdminEmojiAdd)
	s.mux.HandleFunc("POST /api/admin/emoji/delete", s.handleAdminEmojiDelete)
	s.mux.HandleFunc("POST /api/admin/accounts/create", s.handleAdminAccountsCreate)
	s.mux.HandleFunc("POST /api/admin/show-users", s.handleAdminShowUsers)
	s.mux.HandleFunc("POST /api/admin/drive/files", s.handleAdminDriveFiles)
	s.mux.HandleFunc("POST /api/admin/drive/show-file", s.handleAdminDriveShowFile)
	s.mux.HandleFunc("POST /api/admin/get-table-stats", s.handleAdminTableStats)
	s.mux.HandleFunc("POST /api/admin/meta", s.handleAdminMeta)
	s.mux.HandleFunc("POST /api/admin/object-storage", s.handleAdminObjectStorage)
	s.mux.HandleFunc("POST /api/admin/object-storage/update", s.handleAdminObjectStorageUpdate)
	s.mux.HandleFunc("POST /api/admin/email-settings", s.handleAdminEmailSettings)
	s.mux.HandleFunc("POST /api/admin/email-settings/update", s.handleAdminEmailSettingsUpdate)
	s.mux.HandleFunc("POST /api/admin/deliver-queue", s.handleAdminDeliverQueue)
	s.mux.HandleFunc("POST /api/admin/update-meta", s.handleAdminUpdateMeta)
	s.mux.HandleFunc("POST /api/admin/get-index-stats", s.handleAdminIndexStats)
	s.mux.HandleFunc("POST /api/admin/suspend-user", s.handleAdminSuspendUser)
	s.mux.HandleFunc("POST /api/admin/unsuspend-user", s.handleAdminUnsuspendUser)
	// IP bans and the cross-account signin audit (Cinderfish extension,
	// HANDOVER §12GV).
	s.mux.HandleFunc("POST /api/admin/ip-bans/list", s.handleAdminIPBans)
	s.mux.HandleFunc("POST /api/admin/ip-bans/create", s.handleAdminIPBansCreate)
	s.mux.HandleFunc("POST /api/admin/ip-bans/delete", s.handleAdminIPBansDelete)
	s.mux.HandleFunc("POST /api/admin/signin-history", s.handleAdminSigninHistory)
	// Invitation code management: the create/list paths are Misskey's admin
	// namespace; delete is registered under both the upstream user path
	// (invite/delete) and the admin path, since every issuer here is an
	// administrator.
	s.mux.HandleFunc("POST /api/admin/invite/create", s.handleAdminInviteCreate)
	s.mux.HandleFunc("POST /api/admin/invite/list", s.handleAdminInviteList)
	s.mux.HandleFunc("POST /api/admin/invite/delete", s.handleAdminInviteDelete)
	s.mux.HandleFunc("POST /api/invite/delete", s.handleAdminInviteDelete)
	s.mux.HandleFunc("GET /api/admin/federation/mode", s.handleAdminFederationMode)
	s.mux.HandleFunc("POST /api/federation/show-instance", s.handleFederationShowInstance)
	s.mux.HandleFunc("POST /api/federation/instances", s.handleFederationInstances)
	s.mux.HandleFunc("POST /api/admin/federation/mode", s.handleAdminFederationMode)
	s.mux.HandleFunc("GET /api/admin/federation/allowlist", s.handleAdminAllowlist)
	s.mux.HandleFunc("POST /api/admin/federation/allowlist", s.handleAdminAllowlist)

	s.mux.HandleFunc("GET /.well-known/oauth-authorization-server", s.handleOAuthMetadata)
	s.mux.HandleFunc("GET /oauth/authorize", s.handleOAuthAuthorize)
	s.mux.HandleFunc("POST /api/oauth/authorize", s.handleOAuthAuthorizeAPI)
	s.mux.HandleFunc("POST /oauth/token", s.handleOAuthToken)

	s.mux.HandleFunc("POST /api/notes/create", s.handleNotesCreate)
	s.mux.HandleFunc("POST /api/notes/show", s.handleNotesShow)
	s.mux.HandleFunc("POST /api/notes/delete", s.handleNotesDelete)
	s.mux.HandleFunc("POST /api/notes/unrenote", s.handleNotesUnrenote)
	s.mux.HandleFunc("POST /api/notes/featured", s.handleNotesFeatured)
	s.mux.HandleFunc("POST /api/notes/edit", s.handleNotesEdit)
	s.mux.HandleFunc("POST /api/notes/history", s.handleNotesHistory)
	s.mux.HandleFunc("POST /api/notes/state", s.handleNotesState)
	s.mux.HandleFunc("POST /api/notes/replies", s.handleNotesReplies)
	s.mux.HandleFunc("POST /api/notes/conversation", s.handleNotesConversation)
	s.mux.HandleFunc("POST /api/notes/children", s.handleNotesChildren)
	s.mux.HandleFunc("POST /api/notes/mentions", s.handleNotesMentions)
	s.mux.HandleFunc("POST /api/notes/search", s.handleNotesSearch)
	s.mux.HandleFunc("POST /api/notes/search-by-tag", s.handleNotesSearchByTag)
	s.mux.HandleFunc("POST /api/notes/renotes", s.handleNotesRenotes)
	s.mux.HandleFunc("POST /api/notes/reactions", s.handleNotesReactions)
	s.mux.HandleFunc("POST /api/notes/polls/vote", s.handleNotesPollsVote)
	s.mux.HandleFunc("POST /api/hashtags/trend", s.handleHashtagsTrend)
	s.mux.HandleFunc("POST /api/hashtags/search", s.handleHashtagsSearch)
	s.mux.HandleFunc("POST /api/hashtags/show", s.handleHashtagsShow)
	s.mux.HandleFunc("POST /api/hashtags/list", s.handleHashtagsList)
	s.mux.HandleFunc("POST /api/hashtags/users", s.handleHashtagsUsers)
	s.mux.HandleFunc("POST /api/notes/favorites/create", s.handleNotesFavoritesCreate)
	s.mux.HandleFunc("POST /api/notes/favorites/delete", s.handleNotesFavoritesDelete)
	s.mux.HandleFunc("POST /api/i/favorites", s.handleIFavorites)
	s.mux.HandleFunc("POST /api/notes/reactions/create", s.handleNotesReactionsCreate)
	s.mux.HandleFunc("POST /api/notes/reactions/delete", s.handleNotesReactionsDelete)
	s.mux.HandleFunc("POST /api/notes/local-timeline", s.handleLocalTimeline)
	s.mux.HandleFunc("POST /api/notes/hybrid-timeline", s.handleHybridTimeline)
	s.mux.HandleFunc("POST /api/notes/global-timeline", s.handleGlobalTimeline)
	s.mux.HandleFunc("POST /api/notes/timeline", s.handleHomeTimeline)

	s.mux.HandleFunc("POST /api/notifications", s.handleNotifications)
	s.mux.HandleFunc("POST /api/i/notifications", s.handleNotifications)
	s.mux.HandleFunc("POST /api/notifications/mark-all-as-read", s.handleNotificationsMarkAllAsRead)
	s.mux.HandleFunc("POST /api/notifications/mark-as-read", s.handleNotificationsMarkAsRead)

	// Instance announcements: public list plus the admin console.
	s.mux.HandleFunc("POST /api/announcements", s.handleAnnouncements)
	s.mux.HandleFunc("POST /api/i/read-announcement", s.handleIReadAnnouncement)
	s.mux.HandleFunc("POST /api/admin/announcements/create", s.handleAdminAnnouncementsCreate)
	s.mux.HandleFunc("POST /api/admin/announcements/list", s.handleAdminAnnouncementsList)
	s.mux.HandleFunc("POST /api/admin/announcements/update", s.handleAdminAnnouncementsUpdate)
	s.mux.HandleFunc("POST /api/admin/announcements/delete", s.handleAdminAnnouncementsDelete)

	// Abuse reports: users file them, administrators resolve them.
	s.mux.HandleFunc("POST /api/admin/abuse-user-reports", s.handleAdminAbuseReports)
	s.mux.HandleFunc("POST /api/admin/resolve-abuse-user-report", s.handleAdminResolveAbuseReport)
	s.mux.HandleFunc("POST /api/admin/forward-abuse-user-report", s.handleAdminForwardAbuseReport)
	s.mux.HandleFunc("POST /api/admin/update-abuse-user-report", s.handleAdminUpdateAbuseReport)

	s.mux.HandleFunc("POST /api/drive/files/create", s.handleDriveFilesCreate)
	s.mux.HandleFunc("POST /api/drive/files/upload-from-url", s.handleDriveFilesUploadFromURL)
	s.mux.HandleFunc("POST /api/drive/folders", s.handleDriveFolders)
	s.mux.HandleFunc("POST /api/drive/folders/create", s.handleDriveFoldersCreate)
	s.mux.HandleFunc("POST /api/drive/folders/show", s.handleDriveFoldersShow)
	s.mux.HandleFunc("POST /api/drive/folders/update", s.handleDriveFoldersUpdate)
	s.mux.HandleFunc("POST /api/drive/folders/delete", s.handleDriveFoldersDelete)
	s.mux.HandleFunc("POST /api/drive", s.handleDrive)
	s.mux.HandleFunc("POST /api/drive/files", s.handleDriveFilesList)
	s.mux.HandleFunc("POST /api/drive/files/attached-notes", s.handleDriveAttachedNotes)
	s.mux.HandleFunc("POST /api/drive/files/show", s.handleDriveFilesShow)
	s.mux.HandleFunc("POST /api/drive/files/update", s.handleDriveFilesUpdate)
	s.mux.HandleFunc("POST /api/drive/files/delete", s.handleDriveFilesDelete)
	s.mux.HandleFunc("GET /media/{path...}", s.handleMediaServe)

	s.mux.HandleFunc("GET /.well-known/webfinger", s.handleWebFinger)
	s.mux.HandleFunc("GET /.well-known/host-meta", s.handleHostMetaXML)
	s.mux.HandleFunc("GET /.well-known/host-meta.json", s.handleHostMeta)
	s.mux.HandleFunc("GET /.well-known/nodeinfo", s.handleNodeInfo)
	s.mux.HandleFunc("GET /nodeinfo/2.1", s.handleNodeInfo21)
	s.mux.HandleFunc("GET /nodeinfo/2.0", s.handleNodeInfo20)
	s.mux.HandleFunc("OPTIONS /.well-known/", s.handleWellKnownOptions)

	s.mux.HandleFunc("POST /inbox", s.handleInbox)
	s.mux.HandleFunc("GET /users/{id}", s.handleActor)
	s.mux.HandleFunc("GET /users/{id}/inbox", s.handleUserInbox)
	s.mux.HandleFunc("POST /users/{id}/inbox", s.handleInbox)
	s.mux.HandleFunc("GET /users/{id}/publickey", s.handlePublicKey)
	s.mux.HandleFunc("GET /users/{id}/outbox", s.handleUserOutbox)
	s.mux.HandleFunc("GET /users/{id}/followers", s.handleUserFollowers)
	s.mux.HandleFunc("GET /users/{id}/following", s.handleUserFollowing)
	s.mux.HandleFunc("GET /users/{id}/collections/featured", s.handleUserFeatured)
	// Go patterns require a wildcard to start a path segment, so the
	// @handle route is matched as /{acct} and filtered in the handler.
	s.mux.HandleFunc("GET /{acct}", s.handleActorByAcct)
	s.mux.HandleFunc("GET /notes/{id}", s.handleNote)
	s.mux.HandleFunc("GET /notes/{id}/activity", s.handleNoteActivity)

	s.mux.HandleFunc("GET /streaming", s.handleStreaming)

	s.mux.HandleFunc("/", notFoundHandler)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		setAPICORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	if s.web != nil && s.web.handles(r) {
		s.web.serve(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// setAPICORS lets browser clients call the API from another origin.
// Misskey registers @fastify/cors with origin "*" on its API server
// (server/api/ApiServerService.ts), and web clients rely on it: without the
// preflight answer the browser blocks the call and the client can only
// report a generic load failure (misskey-hub's follow flow is one).
// A star origin is sent without Access-Control-Allow-Credentials, so
// cross-origin callers cannot reuse a session cookie; tokens still travel
// in the request body or an Authorization header.
func setAPICORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodOptions {
		return
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
		w.Header().Set("Access-Control-Allow-Headers", h)
	} else {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	}
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// writeJSON emits a Misskey-shaped success response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// APIError mirrors Misskey's ApiError: the response body is
// {"error":{"message","code","id","kind"[, "info"]}} and kind is always
// present (client, server or permission).
type APIError struct {
	Message string
	Code    string
	ID      string
	Kind    string // client | server | permission
	Info    any
	HasInfo bool
	Status  int
}

func (e *APIError) Error() string { return e.Message }

// Well-known error ids, copied from Misskey so clients and operators can
// recognise them (packages/backend/src/server/api/error.ts).
const (
	errIDInvalidParam    = "3d81ceae-475f-4600-b2a8-2bc116157532"
	errIDAuthentication  = "b0a7f5f8-dc2f-4171-b91f-de88ad238e14"
	errIDInternal        = "5d37dbcb-891e-41ca-a3d6-e690c97775ac"
	errIDNoSuchEndpoint  = "fc9c4a0b-f3a1-4b0e-9a8c-6f7f3d0f6a12"
	errIDRegistrationOff = "4d371c9a-3a0f-4c8b-9b8e-2f9c0a1d3e45"
	// Cinderfish extension ids, outside Misskey's table.
	errIDIPBanned      = "7a1e9c34-5b2d-4f6a-8c70-1d9b3e5f2a41"
	errIDNotFound      = "c1f6a9d2-3e4b-4c5d-9a78-2b3e4f5a6c7d"
	errIDCaptchaFailed = "2b4d6f80-1a3c-4e5f-9d72-8c6e4a2b0f13"
)

// writeAPIError emits the Misskey error envelope.
func writeAPIError(w http.ResponseWriter, e *APIError) {
	body := map[string]any{
		"message": e.Message,
		"code":    e.Code,
		"id":      e.ID,
		"kind":    e.Kind,
	}
	if e.HasInfo {
		body["info"] = e.Info
	}
	status := e.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Misskey"`)
	}
	writeJSON(w, status, map[string]any{"error": body})
}

// invalidParamError reports a request that failed parameter validation.
func invalidParamError(info any) *APIError {
	return &APIError{
		Message: "Invalid param.",
		Code:    "INVALID_PARAM",
		ID:      errIDInvalidParam,
		Kind:    "client",
		Info:    info,
		HasInfo: true,
		Status:  http.StatusBadRequest,
	}
}

// writeAuthenticationError emits Misskey's authentication failure
// response: HTTP 401 with a WWW-Authenticate header.
func writeAuthenticationError(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="Misskey", error="invalid_token", error_description="Authentication failed. Please ensure your token is correct."`)
	writeAPIError(w, &APIError{
		Message: "Authentication failed. Please ensure your token is correct.",
		Code:    "AUTHENTICATION_FAILED",
		ID:      errIDAuthentication,
		Kind:    "client",
		Status:  http.StatusUnauthorized,
	})
}

// clientIP returns the remote address of the request.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// notFoundHandler answers everything that did not match a route with a
// Misskey-style JSON error instead of Go's default plain-text 404, so
// API clients always see the documented error envelope.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, &APIError{
		Message: fmt.Sprintf("No such endpoint: %s %s", r.Method, r.URL.Path),
		Code:    "NO_SUCH_ENDPOINT",
		ID:      errIDNoSuchEndpoint,
		Kind:    "client",
		Status:  http.StatusNotFound,
	})
}
