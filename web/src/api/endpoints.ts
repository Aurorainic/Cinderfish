// Endpoint constants and request/response types for every route registered by
// `internal/httpapi/server.go`.
//
// Paths are relative to `/api` unless noted. They are kept as a const object
// (not string literals at call sites) so a route rename is a single edit and
// so `EndpointName` can drive `api()`'s typing.
//
// None of these strings is a credential: every value is a public API route
// path copied from the Misskey endpoint catalogue (the compatibility
// contract), not a token, key or secret.

import type {
	AbuseReport,
	AdminDriveFile,
	AdminInstanceMeta,
	DeliverQueueInfo,
	Announcement,
	AdminEmoji,
	AuthorizedApp,
	BlockingRecord,
	AllowlistResponse,
	Antenna,
	AntennaSrc,
	Channel,
	ChannelShow,
	Clip,
	DriveFile,
	DriveUsage,
	DriveFolder,
	EmojiSimple,
	FederationInstance,
	FederationModeResponse,
	FollowRequest,
	Following,
	GalleryLike,
	GalleryPost,
	Hashtag,
	HashtagTrend,
	ImportSummary,
	IndexStat,
	InstanceStats,
	InviteCode,
	IPBan,
	AdminSigninEntry,
	MailSettingsInfo,
	MeDetailed,
	MessagingMessage,
	MetaDetailed,
	MetaLite,
	MiAuthTokenResponse,
	MuteRecord,
	Note,
	NoteCreateResponse,
	NoteDetailed,
	UserNotesChart,
	ServerInfo,
	FetchRssFeed,
	NoteReactionWithNote,
	NoteEditEntry,
	NoteEditResponse,
	NoteFavorite,
	NoteReaction,
	NoteState,
	Notification,
	NoteVisibility,
	ObjectStorageInfo,
	Page,
	PageBlock,
	PageFont,
	PageLike,
	PingResponse,
	RegistryDetail,
	SigninHistoryEntry,
	SigninStep,
	SignupResponse,
	TableStat,
	TwoFactorRegistration,
	UserNameAvailability,
	EmailAddressAvailability,
	UserStats,
	RegistryValueType,
	UserDetailed,
	UserLite,
	UserList,
	UserListMembership,
	Webhook,
} from "./types";

export const ENDPOINTS = {
	// meta / ping
	meta: "meta",
	stats: "stats",
	ping: "ping",

	// auth
	signup: "signup",
	signin: "signin",
	usernameAvailable: "username/available",
	emailAddressAvailable: "email-address/available",
	iUpdateEmail: "i/update-email",
	iEmailVerification: "i/email-verification",
	verifyEmail: "verify-email",
	requestResetPassword: "request-reset-password",
	resetPassword: "reset-password",
	miAuthGenToken: "miauth/gen-token",
	iTwoFactorRegister: "i/2fa/register",
	iTwoFactorDone: "i/2fa/done",
	iTwoFactorUnregister: "i/2fa/unregister",
	iTwoFactorPasswordLess: "i/2fa/password-less",
	iChangePassword: "i/change-password",
	iRegenerateToken: "i/regenerate-token",
	iSigninHistory: "i/signin-history",
	iReadAnnouncement: "i/read-announcement",
	iApps: "i/apps",
	iExport: "i/export",
	iImport: "i/import",
	iWebhooksCreate: "i/webhooks/create",
	iWebhooksList: "i/webhooks/list",
	iWebhooksShow: "i/webhooks/show",
	iWebhooksUpdate: "i/webhooks/update",
	iWebhooksDelete: "i/webhooks/delete",
	iDeleteAccount: "i/delete-account",
	iRevokeToken: "i/revoke-token",

	// current user
	i: "i",
	iUpdate: "i/update",
	usersUpdate: "users/update",
	iPin: "i/pin",
	iUnpin: "i/unpin",
	iKnownAs: "i/known-as",
	iMove: "i/move",
	oauthAuthorize: "oauth/authorize",
	apShow: "ap/show",
	appCreate: "app/create",
	appShow: "app/show",
	authSessionGenerate: "auth/session/generate",
	authSessionShow: "auth/session/show",
	authSessionUserkey: "auth/session/userkey",
	authAccept: "auth/accept",
	iFavorites: "i/favorites",
	iNotifications: "i/notifications",

	// users
	usersList: "users",
	usersShow: "users/show",
	usersSearch: "users/search",
	usersReportAbuse: "users/report-abuse",
	usersNotes: "users/notes",
	usersStats: "users/stats",
	usersFollowers: "users/followers",
	usersFollowing: "users/following",

	// following
	followingCreate: "following/create",
	followingDelete: "following/delete",
	followingRequestsList: "following/requests/list",
	followingRequestsSent: "following/requests/sent",
	followingRequestsAccept: "following/requests/accept",
	followingRequestsReject: "following/requests/reject",
	followingRequestsCancel: "following/requests/cancel",
	followingList: "following/list",
	followingUpdate: "following/update",
	followingUpdateAll: "following/update-all",
	followingInvalidate: "following/invalidate",

	// mutes and blocks
	muteCreate: "mute/create",
	muteDelete: "mute/delete",
	muteList: "mute/list",
	blockingCreate: "blocking/create",
	blockingDelete: "blocking/delete",
	blockingList: "blocking/list",

	// registry
	registrySet: "i/registry/set",
	registryGet: "i/registry/get",
	registryGetDetail: "i/registry/get-detail",
	registryKeys: "i/registry/keys",
	registryKeysWithType: "i/registry/keys-with-type",
	registryScopes: "i/registry/scopes",
	registryGetAll: "i/registry/get-all",
	registryRemove: "i/registry/remove",
	registryClear: "i/registry/clear",

	// notes
	notesCreate: "notes/create",
	notesShow: "notes/show",
	notesEdit: "notes/edit",
	notesHistory: "notes/history",
	notesState: "notes/state",
	notesDelete: "notes/delete",
	notesUnrenote: "notes/unrenote",
	notesFeatured: "notes/featured",
	notesReplies: "notes/replies",
	notesMentions: "notes/mentions",
	notesSearch: "notes/search",
	notesSearchByTag: "notes/search-by-tag",
	notesConversation: "notes/conversation",
	notesChildren: "notes/children",
	notesRenotes: "notes/renotes",
	notesReactions: "notes/reactions",
	notesReactionsCreate: "notes/reactions/create",
	notesReactionsDelete: "notes/reactions/delete",
	notesFavoritesCreate: "notes/favorites/create",
	notesFavoritesDelete: "notes/favorites/delete",
	notesPollsVote: "notes/polls/vote",
	hashtagsTrend: "hashtags/trend",
	hashtagsSearch: "hashtags/search",
	hashtagsShow: "hashtags/show",
	hashtagsList: "hashtags/list",
	hashtagsUsers: "hashtags/users",

	// user lists
	usersListsCreate: "users/lists/create",
	usersListsCreateFromPublic: "users/lists/create-from-public",
	usersListsDelete: "users/lists/delete",
	usersListsList: "users/lists/list",
	usersListsShow: "users/lists/show",
	usersListsUpdate: "users/lists/update",
	usersListsPush: "users/lists/push",
	usersListsPull: "users/lists/pull",
	usersListsUpdateMembership: "users/lists/update-membership",
	usersListsGetMemberships: "users/lists/get-memberships",
	usersListsFavorite: "users/lists/favorite",
	usersListsUnfavorite: "users/lists/unfavorite",

	// messaging (fork-preserved 1:1 contract; groups are deferred)
	messagingHistory: "messaging/history",
	messagingMessages: "messaging/messages",
	messagingMessagesCreate: "messaging/messages/create",
	messagingMessagesDelete: "messaging/messages/delete",
	messagingMessagesRead: "messaging/messages/read",
	iReadAllMessagingMessages: "i/read-all-messaging-messages",

	// clips
	clipsCreate: "clips/create",
	clipsUpdate: "clips/update",
	clipsDelete: "clips/delete",
	clipsList: "clips/list",
	clipsShow: "clips/show",
	clipsMyFavorites: "clips/my-favorites",
	clipsFavorite: "clips/favorite",
	clipsUnfavorite: "clips/unfavorite",
	clipsAddNote: "clips/add-note",
	clipsRemoveNote: "clips/remove-note",
	clipsNotes: "clips/notes",
	notesClips: "notes/clips",
	usersClips: "users/clips",

	// antennas
	antennasCreate: "antennas/create",
	antennasUpdate: "antennas/update",
	antennasDelete: "antennas/delete",
	antennasList: "antennas/list",
	antennasShow: "antennas/show",
	antennasNotes: "antennas/notes",
	antennasRemoveNote: "antennas/remove-note",

	// channels (schema v33; there is no channels/delete — archiving is deletion)
	channelsCreate: "channels/create",
	channelsUpdate: "channels/update",
	channelsShow: "channels/show",
	channelsTimeline: "channels/timeline",
	channelsFavorite: "channels/favorite",
	channelsUnfavorite: "channels/unfavorite",
	channelsFollow: "channels/follow",
	channelsUnfollow: "channels/unfollow",
	channelsFollowed: "channels/followed",
	channelsOwned: "channels/owned",
	channelsMyFavorites: "channels/my-favorites",
	channelsFeatured: "channels/featured",
	channelsSearch: "channels/search",
	channelsMuteCreate: "channels/mute/create",
	channelsMuteDelete: "channels/mute/delete",
	channelsMuteList: "channels/mute/list",

	// gallery (schema v35)
	galleryPosts: "gallery/posts",
	galleryFeatured: "gallery/featured",
	galleryPopular: "gallery/popular",
	galleryPostsCreate: "gallery/posts/create",
	galleryPostsUpdate: "gallery/posts/update",
	galleryPostsDelete: "gallery/posts/delete",
	galleryPostsShow: "gallery/posts/show",
	galleryPostsLike: "gallery/posts/like",
	galleryPostsUnlike: "gallery/posts/unlike",
	iGalleryPosts: "i/gallery/posts",
	iGalleryLikes: "i/gallery/likes",
	usersGalleryPosts: "users/gallery/posts",

	// pages (schema v36)
	pagesCreate: "pages/create",
	pagesUpdate: "pages/update",
	pagesDelete: "pages/delete",
	pagesShow: "pages/show",
	pagesFeatured: "pages/featured",
	pagesLike: "pages/like",
	pagesUnlike: "pages/unlike",
	iPages: "i/pages",
	iPageLikes: "i/page-likes",
	usersPages: "users/pages",

	// timelines
	notesTimeline: "notes/timeline",
	notesLocalTimeline: "notes/local-timeline",
	notesHybridTimeline: "notes/hybrid-timeline",
	notesGlobalTimeline: "notes/global-timeline",
	notesUserListTimeline: "notes/user-list-timeline",

	// notifications
	announcements: "announcements",
	adminAbuseReports: "admin/abuse-user-reports",
	adminMeta: "admin/meta",
	adminUpdateMeta: "admin/update-meta",
	adminObjectStorage: "admin/object-storage",
	adminObjectStorageUpdate: "admin/object-storage/update",
	adminEmailSettings: "admin/email-settings",
	adminEmailSettingsUpdate: "admin/email-settings/update",
	adminDeliverQueue: "admin/deliver-queue",
	adminTableStats: "admin/get-table-stats",
	adminIndexStats: "admin/get-index-stats",
	adminResolveAbuseReport: "admin/resolve-abuse-user-report",
	adminForwardAbuseReport: "admin/forward-abuse-user-report",
	adminUpdateAbuseReport: "admin/update-abuse-user-report",
	adminAnnouncementsCreate: "admin/announcements/create",
	adminAnnouncementsList: "admin/announcements/list",
	adminAnnouncementsUpdate: "admin/announcements/update",
	adminAnnouncementsDelete: "admin/announcements/delete",
	adminInviteCreate: "admin/invite/create",
	adminInviteList: "admin/invite/list",
	adminInviteDelete: "admin/invite/delete",
	adminIPBansList: "admin/ip-bans/list",
	adminIPBansCreate: "admin/ip-bans/create",
	adminIPBansDelete: "admin/ip-bans/delete",
	adminSigninHistory: "admin/signin-history",

	notifications: "notifications",
	notificationsMarkAllAsRead: "notifications/mark-all-as-read",

	// drive
	drive: "drive",
	driveFiles: "drive/files",
	driveFilesCreate: "drive/files/create",
	driveFilesUploadFromUrl: "drive/files/upload-from-url",
	driveFilesShow: "drive/files/show",
	driveFilesUpdate: "drive/files/update",
	driveFilesDelete: "drive/files/delete",
	driveFolders: "drive/folders",
	driveFoldersCreate: "drive/folders/create",
	driveFoldersShow: "drive/folders/show",
	driveFoldersUpdate: "drive/folders/update",
	driveFoldersDelete: "drive/folders/delete",

	// emoji
	emojis: "emojis",
	adminEmojiList: "admin/emoji/list",
	adminEmojiAdd: "admin/emoji/add",
	adminEmojiDelete: "admin/emoji/delete",

	// admin
	adminAccountsCreate: "admin/accounts/create",
	adminShowUsers: "admin/show-users",
	adminDriveFiles: "admin/drive/files",
	adminDriveShowFile: "admin/drive/show-file",
	adminSuspendUser: "admin/suspend-user",
	adminUnsuspendUser: "admin/unsuspend-user",
	federationShowInstance: "federation/show-instance",
	federationInstances: "federation/instances",
	chartsUserNotes: "charts/user/notes",
	serverInfo: "server-info",
	usersReactions: "users/reactions",
	getSounds: "get-sounds",
	driveAttachedNotes: "drive/files/attached-notes",
	fetchRss: "fetch-rss",
	adminFederationMode: "admin/federation/mode",
	adminFederationAllowlist: "admin/federation/allowlist",
} as const;

export type EndpointName = (typeof ENDPOINTS)[keyof typeof ENDPOINTS];

/** Body for routes whose successful response carries no data (`{}`). */
export type EmptyResponse = Record<string, never>;

export interface PaginationParams {
	limit?: number;
	sinceId?: string;
	untilId?: string;
	sinceDate?: number;
	untilDate?: number;
}

/**
 * Writable antenna fields. `antennas/create` requires every non-optional field;
 * `antennas/update` accepts any subset, and an omitted field stays unchanged.
 * `userListId` is required when `src` is `"list"`.
 */
export interface AntennaInput {
	name: string;
	src: AntennaSrc;
	keywords: string[][];
	excludeKeywords: string[][];
	users: string[];
	caseSensitive: boolean;
	withReplies: boolean;
	withFile: boolean;
	userListId?: string | null;
	localOnly?: boolean;
	excludeBots?: boolean;
	excludeNotesInSensitiveChannel?: boolean;
}

/**
 * Writable channel fields. `channels/create` requires `name`; the rest is
 * optional. `channels/update` accepts any subset, and an omitted field keeps
 * its stored value. `bannerId: ""` (or `null`) clears the banner — Cinderfish
 * fixes upstream's silent no-clear.
 */
export interface ChannelInput {
	name: string;
	description?: string | null;
	bannerId?: string | null;
	color?: string;
	isSensitive?: boolean;
	allowRenoteToExternal?: boolean;
}

/** `channels/update` also carries the archive switch and the pin list. */
export interface ChannelPatch extends Partial<ChannelInput> {
	channelId: string;
	isArchived?: boolean;
	pinnedNoteIds?: string[];
}

/** `channels/search` type selector. */
export type ChannelSearchType = "nameAndDescription" | "nameOnly";

export interface NotesSearchParams {
	query: string;
	limit?: number;
	offset?: number;
	sinceId?: string;
	untilId?: string;
	/** null = local only, a host = that instance, absent = every instance. */
	host?: string | null;
	userId?: string;
	withFiles?: boolean;
}

export interface NotesSearchByTagParams {
	tag: string;
	limit?: number;
	sinceId?: string;
	untilId?: string;
	withFiles?: boolean;
}

export interface UsersSearchParams {
	query: string;
	origin?: "combined" | "local" | "remote";
	host?: string | null;
	limit?: number;
	offset?: number;
	sinceId?: string;
	untilId?: string;
}

export interface RelationListParams {
	limit?: number;
	sinceId?: string;
	untilId?: string;
}

export interface TimelineParams extends PaginationParams {
	withFiles?: boolean;
	withRenotes?: boolean;
	withReplies?: boolean;
	includeMyRenotes?: boolean;
	includeRenotedMyNotes?: boolean;
	includeLocalRenotes?: boolean;
}

export interface MetaRequest {
	detail?: boolean;
}

export interface SignupParams {
	username: string;
	password: string;
	invitationCode?: string;
}

export interface SigninParams {
	username: string;
	password: string;
	/** TOTP code, supplied only on the second step of a 2FA signin. */
	token?: string;
}

export interface MiAuthGenTokenParams {
	session: string;
	name?: string;
	description?: string;
	iconUrl?: string;
	permission?: string[];
}

export interface IUpdateParams {
	name?: string | null;
	description?: string | null;
	avatarId?: string | null;
	followersVisibility?: "public" | "followers" | "private";
	followingVisibility?: "public" | "followers" | "private";
	mutedInstances?: string[] | null;
	/** Pins a page to the profile; null unpins. Foreign pages answer NO_SUCH_PAGE. */
	pinnedPageId?: string | null;
}

export interface AppEntity {
	id: string;
	name: string;
	callbackUrl: string | null;
	permission: string[];
	isAuthorized: boolean;
	secret?: string;
}

export interface UsersShowParams {
	userId?: string;
	username?: string;
	/** Must be empty/null; the backend rejects non-empty hosts. */
	host?: string | null;
}

export interface UsersNotesParams {
	userId?: string;
	username?: string;
	limit?: number;
	sinceId?: string;
	untilId?: string;
	withFiles?: boolean;
	withReplies?: boolean;
}

export interface UsersFollowersParams {
	userId?: string;
	limit?: number;
	sinceId?: string;
	untilId?: string;
}

export interface NoteIdParams {
	noteId: string;
}

export interface NoteListParams extends PaginationParams {
	noteId: string;
}

export interface NoteReactionsListParams extends PaginationParams {
	noteId: string;
	type?: string;
}

export interface PollParam {
	choices: string[];
	multiple?: boolean;
	expiredAfter?: number;
	expiresAt?: number;
}

export interface NotesCreateParams {
	visibility?: NoteVisibility;
	visibleUserIds?: string[];
	text?: string | null;
	cw?: string | null;
	localOnly?: boolean;
	fileIds?: string[];
	mediaIds?: string[];
	replyId?: string | null;
	renoteId?: string | null;
	/** Posts into a channel; the backend forces public + localOnly. */
	channelId?: string | null;
	poll?: PollParam;
}

export interface NotificationsParams extends PaginationParams {
	markAsRead?: boolean;
	includeTypes?: string[];
	excludeTypes?: string[];
}

export interface RegistryParams {
	key?: string;
	value?: unknown;
	scope?: string[];
	domain?: string | null;
}

export interface RegistryGetParams {
	key?: string;
	scope?: string[];
	domain?: string | null;
}

export interface DriveFilesParams extends PaginationParams {
	folderId?: string | null;
}

export interface DriveFolderCreateParams {
	name?: string;
	parentId?: string | null;
}

export interface DriveFolderUpdateParams {
	folderId: string;
	name?: string | null;
	parentId?: string | null;
}

export interface AdminEmojiListParams extends PaginationParams {
	query?: string | null;
}

export interface AdminEmojiAddParams {
	name: string;
	fileId: string;
	category?: string | null;
	aliases?: string[];
}

export interface AdminAccountsCreateParams {
	username: string;
	password: string;
}

export interface WebhookParams {
	name: string;
	url: string;
	secret: string;
	on: string[];
}

export interface WebhookUpdateParams extends WebhookParams {
	webhookId: string;
	active: boolean;
}

export interface AbuseReportsParams {
	limit?: number;
	sinceId?: string;
	untilId?: string;
	state?: "resolved" | "unresolved" | null;
	reporterOrigin?: "combined" | "local" | "remote";
	targetUserOrigin?: "combined" | "local" | "remote";
}

export interface AnnouncementsParams {
	limit?: number;
	untilId?: string;
}

export interface AdminAnnouncementsCreateParams {
	title: string;
	text: string;
	imageUrl?: string | null;
	/** Unix milliseconds. */
	expiresAt?: number | null;
}

export interface AdminShowUsersParams {
	limit?: number;
	offset?: number;
	sort?: "+follower" | "-follower" | "+createdAt" | "-createdAt" | "+updatedAt" | "-updatedAt";
	state?: "all" | "alive" | "available" | "admin" | "moderator" | "adminOrModerator" | "silenced" | "suspended";
	origin?: "combined" | "local" | "remote";
	username?: string | null;
	hostname?: string | null;
}

export interface AdminDriveFilesParams {
	limit?: number;
	sinceId?: string;
	untilId?: string;
	userId?: string | null;
	type?: string | null;
	origin?: "combined" | "local" | "remote";
	hostname?: string | null;
}

export interface FollowingRequestsParams extends PaginationParams {
	userId?: string;
}

/**
 * Endpoint -> request/response contract. This is the typed surface used by
 * `api()`, `CfPagination` and future page ports.
 */
export interface EndpointMap {
	[ENDPOINTS.meta]: { req: MetaRequest; res: MetaDetailed | MetaLite };
	[ENDPOINTS.ping]: { req: Record<string, never>; res: PingResponse };

	[ENDPOINTS.signup]: { req: SignupParams; res: SignupResponse };
	[ENDPOINTS.signin]: { req: SigninParams; res: SigninStep };
	[ENDPOINTS.usernameAvailable]: {
		req: { username: string };
		res: UserNameAvailability;
	};
	[ENDPOINTS.emailAddressAvailable]: {
		req: { email: string };
		res: EmailAddressAvailability;
	};
	[ENDPOINTS.iUpdateEmail]: {
		req: { password: string; email: string };
		res: MeDetailed;
	};
	[ENDPOINTS.iEmailVerification]: {
		req: { email?: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.verifyEmail]: { req: { code: string }; res: EmptyResponse };
	[ENDPOINTS.requestResetPassword]: {
		req: { username: string; email: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.resetPassword]: {
		req: { token: string; password: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.iTwoFactorRegister]: {
		req: { password: string };
		res: TwoFactorRegistration;
	};
	[ENDPOINTS.iTwoFactorDone]: {
		req: { token: string };
		res: Record<string, never>;
	};
	[ENDPOINTS.iTwoFactorUnregister]: {
		req: { password: string };
		res: Record<string, never>;
	};
	[ENDPOINTS.iTwoFactorPasswordLess]: {
		req: { value: boolean };
		res: Record<string, never>;
	};
	[ENDPOINTS.iChangePassword]: {
		req: { currentPassword: string; newPassword: string };
		res: Record<string, never>;
	};
	[ENDPOINTS.iRegenerateToken]: {
		req: { password: string };
		res: { token: string };
	};
	[ENDPOINTS.iReadAnnouncement]: {
		req: { announcementId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.iSigninHistory]: {
		req: { limit?: number; sinceId?: string; untilId?: string };
		res: SigninHistoryEntry[];
	};
	[ENDPOINTS.miAuthGenToken]: {
		req: MiAuthGenTokenParams;
		res: MiAuthTokenResponse;
	};

	[ENDPOINTS.i]: { req: Record<string, never>; res: MeDetailed };
	[ENDPOINTS.iUpdate]: { req: IUpdateParams; res: MeDetailed };
	[ENDPOINTS.usersUpdate]: { req: IUpdateParams; res: MeDetailed };
	[ENDPOINTS.iPin]: { req: NoteIdParams; res: MeDetailed };
	[ENDPOINTS.iUnpin]: { req: NoteIdParams; res: MeDetailed };
	[ENDPOINTS.iKnownAs]: { req: { alsoKnownAs: string[] }; res: MeDetailed };
	[ENDPOINTS.iMove]: { req: { moveToAccount: string }; res: MeDetailed };
	[ENDPOINTS.appCreate]: {
		req: {
			name: string;
			description: string;
			permission: string[];
			callbackUrl?: string | null;
		};
		res: AppEntity;
	};
	[ENDPOINTS.appShow]: { req: { appId: string }; res: AppEntity };
	[ENDPOINTS.authSessionGenerate]: {
		req: { appSecret: string };
		res: { token: string; url: string };
	};
	[ENDPOINTS.authSessionShow]: {
		req: { token: string };
		res: { id: string; app: AppEntity; token: string };
	};
	[ENDPOINTS.authSessionUserkey]: {
		req: { appSecret: string; token: string };
		res: { accessToken: string; user: UserDetailed };
	};
	[ENDPOINTS.authAccept]: { req: { token: string }; res: EmptyResponse };
	[ENDPOINTS.apShow]: {
		req: { uri: string };
		res: {
			type: "Note" | "User";
			object: NoteDetailed | UserDetailed;
		};
	};
	[ENDPOINTS.oauthAuthorize]: {
		req: {
			client_id: string;
			redirect_uri: string;
			response_type: "code";
			code_challenge: string;
			code_challenge_method: "S256";
			state?: string;
			scope?: string[];
		};
		res: { redirect: string };
	};
	[ENDPOINTS.iFavorites]: {
		req: PaginationParams;
		res: NoteFavorite[];
	};
	[ENDPOINTS.iNotifications]: {
		req: NotificationsParams;
		res: Notification[];
	};

	[ENDPOINTS.usersList]: {
		req: {
			limit?: number;
			offset?: number;
			sort?: "+follower" | "-follower" | "+createdAt" | "-createdAt" | "+updatedAt" | "-updatedAt";
			state?: "all" | "alive" | "admin" | "moderator" | "adminOrModerator";
			origin?: "combined" | "local" | "remote";
			hostname?: string | null;
		};
		res: UserDetailed[];
	};
	[ENDPOINTS.usersShow]: { req: UsersShowParams; res: UserDetailed };
	[ENDPOINTS.usersNotes]: { req: UsersNotesParams; res: NoteDetailed[] };
	[ENDPOINTS.usersStats]: { req: { userId?: string }; res: UserStats };
	[ENDPOINTS.usersFollowers]: {
		req: UsersFollowersParams;
		res: Following[];
	};
	[ENDPOINTS.usersFollowing]: {
		req: UsersFollowersParams;
		res: Following[];
	};

	[ENDPOINTS.followingCreate]: { req: { userId: string }; res: UserLite };
	[ENDPOINTS.followingDelete]: { req: { userId: string }; res: UserLite };
	[ENDPOINTS.followingList]: {
		req: { notification?: boolean; limit?: number; sinceId?: string; untilId?: string };
		res: Following[];
	};
	[ENDPOINTS.followingUpdate]: {
		req: { userId: string; notify?: "normal" | "none"; withReplies?: boolean };
		res: UserLite;
	};
	[ENDPOINTS.followingUpdateAll]: {
		req: { notify?: "normal" | "none"; withReplies?: boolean };
		res: EmptyResponse;
	};
	[ENDPOINTS.followingInvalidate]: { req: { userId: string }; res: UserLite };
	[ENDPOINTS.followingRequestsList]: {
		req: FollowingRequestsParams;
		res: FollowRequest[];
	};
	[ENDPOINTS.followingRequestsSent]: {
		req: FollowingRequestsParams;
		res: FollowRequest[];
	};
	[ENDPOINTS.followingRequestsAccept]: {
		req: { userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.followingRequestsReject]: {
		req: { userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.followingRequestsCancel]: {
		req: { userId: string };
		res: UserLite;
	};

	// user lists
	[ENDPOINTS.usersListsCreate]: {
		req: { name: string };
		res: UserList;
	};
	[ENDPOINTS.usersListsCreateFromPublic]: {
		req: { name: string; listId: string };
		res: UserList;
	};
	[ENDPOINTS.usersListsDelete]: {
		req: { listId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.usersListsList]: {
		req: { userId?: string };
		res: UserList[];
	};
	[ENDPOINTS.usersListsShow]: {
		req: { listId: string; forPublic?: boolean };
		res: UserList;
	};
	[ENDPOINTS.usersListsUpdate]: {
		req: { listId: string; name?: string; isPublic?: boolean };
		res: UserList;
	};
	[ENDPOINTS.usersListsPush]: {
		req: { listId: string; userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.usersListsPull]: {
		req: { listId: string; userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.usersListsUpdateMembership]: {
		req: { listId: string; userId: string; withReplies: boolean };
		res: EmptyResponse;
	};
	[ENDPOINTS.usersListsGetMemberships]: {
		req: {
			listId: string;
			forPublic?: boolean;
			limit?: number;
			sinceId?: string;
			untilId?: string;
		};
		res: UserListMembership[];
	};
	[ENDPOINTS.usersListsFavorite]: {
		req: { listId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.usersListsUnfavorite]: {
		req: { listId: string };
		res: EmptyResponse;
	};

	// clips
	[ENDPOINTS.clipsCreate]: {
		req: { name: string; isPublic?: boolean; description?: string | null };
		res: Clip;
	};
	[ENDPOINTS.clipsUpdate]: {
		req: {
			clipId: string;
			name?: string;
			isPublic?: boolean;
			/** Omitted keeps the stored description; null or "" clears it. */
			description?: string | null;
		};
		res: Clip;
	};
	[ENDPOINTS.clipsDelete]: {
		req: { clipId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.clipsList]: {
		req: PaginationParams;
		res: Clip[];
	};
	[ENDPOINTS.clipsShow]: {
		req: { clipId: string };
		res: Clip;
	};
	[ENDPOINTS.clipsMyFavorites]: {
		req: Record<string, never>;
		res: Clip[];
	};
	[ENDPOINTS.clipsFavorite]: {
		req: { clipId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.clipsUnfavorite]: {
		req: { clipId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.clipsAddNote]: {
		req: { clipId: string; noteId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.clipsRemoveNote]: {
		req: { clipId: string; noteId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.clipsNotes]: {
		req: PaginationParams & { clipId: string; search?: string };
		res: Note[];
	};
	[ENDPOINTS.notesClips]: {
		req: { noteId: string };
		res: Clip[];
	};
	[ENDPOINTS.usersClips]: {
		req: PaginationParams & { userId: string };
		res: Clip[];
	};

	// messaging
	[ENDPOINTS.messagingHistory]: {
		req: { limit?: number; group?: boolean };
		res: MessagingMessage[];
	};
	[ENDPOINTS.messagingMessages]: {
		req: {
			userId: string;
			limit?: number;
			sinceId?: string;
			untilId?: string;
			/** Defaults to true on the backend; the room passes false and reads explicitly. */
			markAsRead?: boolean;
		};
		res: MessagingMessage[];
	};
	[ENDPOINTS.messagingMessagesCreate]: {
		req: { userId: string; text?: string; fileId?: string };
		res: MessagingMessage;
	};
	[ENDPOINTS.messagingMessagesDelete]: {
		req: { messageId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.messagingMessagesRead]: {
		req: { messageId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.iReadAllMessagingMessages]: {
		req: Record<string, never>;
		res: EmptyResponse;
	};

	// antennas
	[ENDPOINTS.antennasCreate]: { req: AntennaInput; res: Antenna };
	[ENDPOINTS.antennasUpdate]: {
		req: Partial<AntennaInput> & { antennaId: string };
		res: Antenna;
	};
	[ENDPOINTS.antennasDelete]: {
		req: { antennaId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.antennasList]: { req: Record<string, never>; res: Antenna[] };
	[ENDPOINTS.antennasShow]: {
		req: { antennaId: string };
		res: Antenna;
	};
	[ENDPOINTS.antennasNotes]: {
		req: PaginationParams & { antennaId: string };
		res: NoteDetailed[];
	};
	[ENDPOINTS.antennasRemoveNote]: {
		req: { antennaId: string; noteId: string };
		res: EmptyResponse;
	};

	// channels
	[ENDPOINTS.channelsCreate]: { req: ChannelInput; res: Channel };
	[ENDPOINTS.channelsUpdate]: { req: ChannelPatch; res: Channel };
	[ENDPOINTS.channelsShow]: {
		req: { channelId: string };
		res: ChannelShow;
	};
	[ENDPOINTS.channelsTimeline]: {
		req: PaginationParams & { channelId: string };
		res: NoteDetailed[];
	};
	[ENDPOINTS.channelsFavorite]: {
		req: { channelId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsUnfavorite]: {
		req: { channelId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsFollow]: {
		req: { channelId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsUnfollow]: {
		req: { channelId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsFollowed]: {
		req: PaginationParams;
		res: Channel[];
	};
	[ENDPOINTS.channelsOwned]: {
		req: PaginationParams;
		res: Channel[];
	};
	[ENDPOINTS.channelsMyFavorites]: {
		req: Record<string, never>;
		res: Channel[];
	};
	[ENDPOINTS.channelsFeatured]: {
		req: Record<string, never>;
		res: Channel[];
	};
	[ENDPOINTS.channelsSearch]: {
		req: {
			query: string;
			type?: ChannelSearchType;
			limit?: number;
			sinceId?: string;
			untilId?: string;
		};
		res: Channel[];
	};
	[ENDPOINTS.channelsMuteCreate]: {
		req: { channelId: string; expiresAt?: number };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsMuteDelete]: {
		req: { channelId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.channelsMuteList]: {
		req: Record<string, never>;
		res: Channel[];
	};

	// gallery
	[ENDPOINTS.galleryPosts]: {
		req: PaginationParams;
		res: GalleryPost[];
	};
	[ENDPOINTS.galleryFeatured]: {
		req: { limit?: number; untilId?: string; untilDate?: number };
		res: GalleryPost[];
	};
	[ENDPOINTS.galleryPopular]: {
		req: Record<string, never>;
		res: GalleryPost[];
	};
	[ENDPOINTS.galleryPostsCreate]: {
		req: {
			title: string;
			description?: string | null;
			/** 1–32 ids of the caller's own drive files, in display order. */
			fileIds: string[];
			isSensitive?: boolean;
		};
		res: GalleryPost;
	};
	[ENDPOINTS.galleryPostsUpdate]: {
		req: {
			postId: string;
			title?: string;
			/** Omitted keeps the stored description; null clears it. */
			description?: string | null;
			fileIds?: string[];
			isSensitive?: boolean;
		};
		res: GalleryPost;
	};
	[ENDPOINTS.galleryPostsDelete]: {
		req: { postId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.galleryPostsShow]: {
		req: { postId: string };
		res: GalleryPost;
	};
	[ENDPOINTS.galleryPostsLike]: {
		req: { postId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.galleryPostsUnlike]: {
		req: { postId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.iGalleryPosts]: {
		req: PaginationParams;
		res: GalleryPost[];
	};
	[ENDPOINTS.iGalleryLikes]: {
		req: PaginationParams;
		res: GalleryLike[];
	};
	[ENDPOINTS.usersGalleryPosts]: {
		req: PaginationParams & { userId: string };
		res: GalleryPost[];
	};

	// pages
	[ENDPOINTS.pagesCreate]: {
		req: {
			title: string;
			/** The slug; unique per author. */
			name: string;
			summary?: string | null;
			eyeCatchingImageId?: string | null;
			font?: PageFont;
			alignCenter?: boolean;
			hideTitleWhenPinned?: boolean;
			content: PageBlock[];
			/** Opaque; upstream's editor always sends an empty array. */
			variables: unknown[];
			/** Opaque; upstream's editor always sends an empty string. */
			script: string;
		};
		res: Page;
	};
	[ENDPOINTS.pagesUpdate]: {
		req: {
			pageId: string;
			title?: string;
			name?: string;
			/** Omitted keeps the stored summary; null clears it. */
			summary?: string | null;
			/** Omitted keeps the stored banner; null clears it. */
			eyeCatchingImageId?: string | null;
			font?: PageFont;
			alignCenter?: boolean;
			hideTitleWhenPinned?: boolean;
			content?: PageBlock[];
			variables?: unknown[];
			script?: string;
		};
		/** The endpoint answers with an empty body. */
		res: EmptyResponse;
	};
	[ENDPOINTS.pagesDelete]: { req: { pageId: string }; res: EmptyResponse };
	[ENDPOINTS.pagesShow]: {
		/** Either `pageId`, or the pair `name` + `username`. */
		req: { pageId?: string; name?: string; username?: string };
		res: Page;
	};
	[ENDPOINTS.pagesFeatured]: {
		/** No parameters; the endpoint always answers the top ten. */
		req: Record<string, never>;
		res: Page[];
	};
	[ENDPOINTS.pagesLike]: { req: { pageId: string }; res: EmptyResponse };
	[ENDPOINTS.pagesUnlike]: { req: { pageId: string }; res: EmptyResponse };
	[ENDPOINTS.iPages]: { req: PaginationParams; res: Page[] };
	[ENDPOINTS.iPageLikes]: { req: PaginationParams; res: PageLike[] };
	[ENDPOINTS.usersPages]: {
		req: PaginationParams & { userId: string };
		res: Page[];
	};

	[ENDPOINTS.registrySet]: { req: RegistryParams; res: EmptyResponse };
	[ENDPOINTS.registryGet]: { req: RegistryGetParams; res: unknown };
	[ENDPOINTS.registryGetDetail]: {
		req: RegistryGetParams;
		res: RegistryDetail;
	};
	[ENDPOINTS.registryKeys]: { req: RegistryGetParams; res: string[] };
	[ENDPOINTS.registryKeysWithType]: {
		req: RegistryGetParams;
		res: Record<string, RegistryValueType>;
	};
	[ENDPOINTS.registryScopes]: { req: { domain?: string | null }; res: string[][] };
	[ENDPOINTS.registryGetAll]: {
		req: RegistryGetParams;
		res: Record<string, unknown>;
	};
	[ENDPOINTS.registryRemove]: { req: RegistryGetParams; res: EmptyResponse };
	[ENDPOINTS.registryClear]: { req: RegistryGetParams; res: EmptyResponse };

	[ENDPOINTS.notesCreate]: {
		req: NotesCreateParams;
		res: NoteCreateResponse;
	};
	[ENDPOINTS.notesShow]: { req: NoteIdParams; res: NoteDetailed };
	[ENDPOINTS.notesEdit]: {
		req: {
			editId: string;
			text?: string | null;
			cw?: string | null;
			visibility?: NoteVisibility;
			localOnly?: boolean;
		};
		res: NoteEditResponse;
	};
	[ENDPOINTS.notesState]: { req: { noteId: string }; res: NoteState };
	[ENDPOINTS.notesHistory]: {
		req: { noteId: string; limit?: number };
		res: NoteEditEntry[];
	};
	[ENDPOINTS.notesDelete]: { req: NoteIdParams; res: EmptyResponse };
	[ENDPOINTS.notesUnrenote]: { req: NoteIdParams; res: EmptyResponse };
	[ENDPOINTS.notesFeatured]: {
		req: { limit?: number; sinceId?: string; untilId?: string };
		res: Note[];
	};
	[ENDPOINTS.notesReplies]: { req: NoteListParams; res: NoteDetailed[] };
	[ENDPOINTS.notesRenotes]: { req: NoteListParams; res: NoteDetailed[] };
	[ENDPOINTS.notesConversation]: {
		req: { noteId: string; limit?: number; offset?: number };
		res: Note[];
	};
	[ENDPOINTS.notesChildren]: { req: NoteListParams; res: Note[] };
	[ENDPOINTS.notesReactions]: {
		req: NoteReactionsListParams;
		res: NoteReaction[];
	};
	[ENDPOINTS.notesReactionsCreate]: {
		req: { noteId: string; reaction: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.notesReactionsDelete]: {
		req: { noteId: string; reaction?: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.notesFavoritesCreate]: {
		req: NoteIdParams;
		res: NoteFavorite;
	};
	[ENDPOINTS.notesFavoritesDelete]: {
		req: NoteIdParams;
		res: EmptyResponse;
	};
	[ENDPOINTS.notesPollsVote]: {
		req: { noteId: string; choice: number };
		res: EmptyResponse;
	};
	[ENDPOINTS.hashtagsShow]: { req: { tag: string }; res: Hashtag };
	[ENDPOINTS.hashtagsList]: {
		req: {
			sort?: string;
			limit?: number;
			attachedToUserOnly?: boolean;
			attachedToLocalUserOnly?: boolean;
			attachedToRemoteUserOnly?: boolean;
		};
		res: Hashtag[];
	};
	[ENDPOINTS.hashtagsUsers]: {
		req: { tag: string; sort?: string; limit?: number; offset?: number };
		res: UserDetailed[];
	};
	[ENDPOINTS.hashtagsTrend]: { req: Record<string, never>; res: HashtagTrend[] };
	[ENDPOINTS.hashtagsSearch]: {
		req: { query?: string; limit?: number; offset?: number; sort?: string };
		res: string[];
	};

	[ENDPOINTS.notesMentions]: {
		req: {
			following?: boolean;
			limit?: number;
			sinceId?: string;
			untilId?: string;
			visibility?: NoteVisibility;
		};
		res: NoteDetailed[];
	};

	[ENDPOINTS.notesSearch]: {
		req: NotesSearchParams;
		res: NoteDetailed[];
	};
	[ENDPOINTS.notesSearchByTag]: {
		req: NotesSearchByTagParams;
		res: NoteDetailed[];
	};
	[ENDPOINTS.usersSearch]: {
		req: UsersSearchParams;
		res: UserDetailed[];
	};

	[ENDPOINTS.notesTimeline]: { req: TimelineParams; res: NoteDetailed[] };
	[ENDPOINTS.notesLocalTimeline]: { req: TimelineParams; res: NoteDetailed[] };
	[ENDPOINTS.notesHybridTimeline]: { req: TimelineParams; res: NoteDetailed[] };
	[ENDPOINTS.notesGlobalTimeline]: { req: TimelineParams; res: NoteDetailed[] };
	[ENDPOINTS.notesUserListTimeline]: {
		req: TimelineParams & { listId: string };
		res: NoteDetailed[];
	};

	[ENDPOINTS.notifications]: {
		req: NotificationsParams;
		res: Notification[];
	};
	[ENDPOINTS.iWebhooksCreate]: { req: WebhookParams; res: Webhook };
	[ENDPOINTS.iWebhooksList]: {
		req: Record<string, never>;
		res: Webhook[];
	};
	[ENDPOINTS.iWebhooksShow]: {
		req: { webhookId: string };
		res: Webhook;
	};
	[ENDPOINTS.iWebhooksUpdate]: {
		req: WebhookUpdateParams;
		res: Webhook;
	};
	[ENDPOINTS.iWebhooksDelete]: {
		req: { webhookId: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.usersReportAbuse]: {
		req: { userId: string; comment: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.stats]: {
		req: Record<string, never>;
		res: InstanceStats;
	};

	[ENDPOINTS.adminMeta]: {
		req: Record<string, never>;
		res: AdminInstanceMeta;
	};
	[ENDPOINTS.adminUpdateMeta]: {
		req: {
			name?: string;
			description?: string;
			maintainerName?: string;
			maintainerEmail?: string;
			registrationMode?: "open" | "invite" | "admin";
			tosUrl?: string | null;
			privacyPolicyUrl?: string;
			repositoryUrl?: string;
			feedbackUrl?: string;
			inquiryUrl?: string;
			impressumUrl?: string;
			serverRules?: string[];
			hiddenTags?: string[];
			iconUrl?: string | null;
			bannerUrl?: string | null;
			themeColor?: string | null;
			customCss?: string;
			preservedUsernames?: string[];
			maxNoteTextLength?: number;
			captcha?: {
				provider: "none" | "hcaptcha" | "recaptcha" | "turnstile";
				siteKey?: string;
				/** Empty keeps the stored secret for the same provider. */
				secret?: string;
			};
			signinAutoBan?: {
				enabled?: boolean;
				threshold?: number;
				windowMinutes?: number;
				banMinutes?: number;
			};
		};
		res: AdminInstanceMeta;
	};

	[ENDPOINTS.adminObjectStorage]: {
		req: Record<string, never>;
		res: ObjectStorageInfo;
	};
	[ENDPOINTS.adminEmailSettings]: {
		req: Record<string, never>;
		res: MailSettingsInfo;
	};
	[ENDPOINTS.adminEmailSettingsUpdate]: {
		req: {
			host?: string;
			port?: number;
			username?: string;
			/** Empty keeps the stored password. */
			password?: string;
			from?: string;
			tls?: "starttls" | "tls" | "none";
		};
		res: MailSettingsInfo;
	};
	[ENDPOINTS.adminObjectStorageUpdate]: {
		req: {
			backend: "local" | "s3";
			localPath?: string;
			s3?: {
				endpoint: string;
				bucket: string;
				region?: string;
				accessKey?: string;
				/** Empty keeps the stored secret. */
				secretKey?: string;
				useSSL?: boolean;
				publicBaseUrl?: string;
				caCert?: string;
			};
		};
		res: ObjectStorageInfo;
	};
	[ENDPOINTS.adminDeliverQueue]: {
		req: Record<string, never>;
		res: DeliverQueueInfo;
	};

	[ENDPOINTS.adminTableStats]: {
		req: Record<string, never>;
		res: Record<string, TableStat>;
	};
	[ENDPOINTS.adminIndexStats]: {
		req: Record<string, never>;
		res: IndexStat[];
	};

	[ENDPOINTS.adminAbuseReports]: {
		req: AbuseReportsParams;
		res: AbuseReport[];
	};
	[ENDPOINTS.adminResolveAbuseReport]: {
		req: { reportId: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.adminForwardAbuseReport]: {
		req: { reportId: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.adminUpdateAbuseReport]: {
		req: { reportId: string; moderationNote?: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.announcements]: {
		req: AnnouncementsParams;
		res: Announcement[];
	};
	[ENDPOINTS.adminAnnouncementsList]: {
		req: AnnouncementsParams;
		res: Announcement[];
	};
	[ENDPOINTS.adminAnnouncementsCreate]: {
		req: AdminAnnouncementsCreateParams;
		res: Announcement;
	};
	[ENDPOINTS.adminAnnouncementsUpdate]: {
		req: {
			id: string;
			title?: string;
			text?: string;
			imageUrl?: string | null;
			isActive?: boolean;
			expiresAt?: number | null;
		};
		res: EmptyResponse;
	};
	[ENDPOINTS.adminAnnouncementsDelete]: {
		req: { id: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.notificationsMarkAllAsRead]: {
		req: Record<string, never>;
		res: EmptyResponse;
	};

	[ENDPOINTS.drive]: { req: Record<string, never>; res: DriveUsage };
	[ENDPOINTS.driveFiles]: { req: DriveFilesParams; res: DriveFile[] };
	[ENDPOINTS.driveFilesCreate]: {
		req: { folderId?: string | null };
		res: DriveFile;
	};
	[ENDPOINTS.driveFilesUploadFromUrl]: {
		req: { url: string; folderId?: string | null; name?: string | null };
		res: DriveFile;
	};
	[ENDPOINTS.driveFilesShow]: { req: { fileId: string }; res: DriveFile };
	[ENDPOINTS.driveFilesUpdate]: {
		req: {
			fileId: string;
			name?: string | null;
			isSensitive?: boolean;
			comment?: string | null;
			folderId?: string | null;
		};
		res: DriveFile;
	};
	[ENDPOINTS.driveFilesDelete]: { req: { fileId: string }; res: EmptyResponse };
	[ENDPOINTS.driveFolders]: { req: DriveFilesParams; res: DriveFolder[] };
	[ENDPOINTS.driveFoldersCreate]: {
		req: DriveFolderCreateParams;
		res: DriveFolder;
	};
	[ENDPOINTS.driveFoldersShow]: {
		req: { folderId: string };
		res: DriveFolder;
	};
	[ENDPOINTS.driveFoldersUpdate]: {
		req: DriveFolderUpdateParams;
		res: DriveFolder;
	};
	[ENDPOINTS.driveFoldersDelete]: {
		req: { folderId: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.emojis]: {
		req: Record<string, never>;
		res: { emojis: EmojiSimple[] };
	};
	[ENDPOINTS.adminEmojiList]: {
		req: AdminEmojiListParams;
		res: AdminEmoji[];
	};
	[ENDPOINTS.adminEmojiAdd]: { req: AdminEmojiAddParams; res: AdminEmoji };
	[ENDPOINTS.adminEmojiDelete]: { req: { id: string }; res: EmptyResponse };

	[ENDPOINTS.adminAccountsCreate]: {
		req: AdminAccountsCreateParams;
		res: UserDetailed;
	};
	[ENDPOINTS.adminShowUsers]: {
		req: AdminShowUsersParams;
		res: UserDetailed[];
	};
	[ENDPOINTS.adminDriveFiles]: {
		req: AdminDriveFilesParams;
		res: AdminDriveFile[];
	};
	[ENDPOINTS.adminDriveShowFile]: {
		req: { fileId: string };
		res: AdminDriveFile;
	};
	[ENDPOINTS.adminSuspendUser]: {
		req: { userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.adminUnsuspendUser]: {
		req: { userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.iApps]: {
		req: { sort?: "+createdAt" | "-createdAt" | "+lastUsedAt" | "-lastUsedAt" };
		res: AuthorizedApp[];
	};
	[ENDPOINTS.iRevokeToken]: { req: { tokenId: string }; res: EmptyResponse };

	// Answered with a file body through apiDownload(); the entry keeps
	// EndpointName exhaustive so api("i/export") still type-checks.
	[ENDPOINTS.iImport]: { req: { fileId: string }; res: ImportSummary };
	[ENDPOINTS.iExport]: { req: Record<string, never>; res: EmptyResponse };
	[ENDPOINTS.iDeleteAccount]: {
		req: { password: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.muteCreate]: { req: { userId: string }; res: EmptyResponse };
	[ENDPOINTS.muteDelete]: { req: { userId: string }; res: EmptyResponse };
	[ENDPOINTS.muteList]: {
		req: RelationListParams;
		res: MuteRecord[];
	};
	[ENDPOINTS.blockingCreate]: { req: { userId: string }; res: UserDetailed };
	[ENDPOINTS.blockingDelete]: {
		req: { userId: string };
		res: EmptyResponse;
	};
	[ENDPOINTS.blockingList]: {
		req: RelationListParams;
		res: BlockingRecord[];
	};

	[ENDPOINTS.federationShowInstance]: {
		req: { host: string };
		res: FederationInstance | null;
	};
	[ENDPOINTS.federationInstances]: {
		req: { host?: string | null; limit?: number; offset?: number };
		res: FederationInstance[];
	};

	[ENDPOINTS.fetchRss]: {
		req: { url: string };
		res: FetchRssFeed;
	};

	[ENDPOINTS.driveAttachedNotes]: {
		req: {
			fileId: string;
			limit?: number;
			sinceId?: string;
			untilId?: string;
			sinceDate?: number;
			untilDate?: number;
		};
		res: Note[];
	};

	[ENDPOINTS.getSounds]: {
		req: Record<string, never>;
		res: (string | null)[];
	};

	[ENDPOINTS.usersReactions]: {
		req: {
			userId: string;
			limit?: number;
			sinceId?: string;
			untilId?: string;
			sinceDate?: number;
			untilDate?: number;
		};
		res: NoteReactionWithNote[];
	};

	[ENDPOINTS.serverInfo]: {
		req: Record<string, never>;
		res: ServerInfo;
	};

	[ENDPOINTS.chartsUserNotes]: {
		req: {
			span: "day" | "hour";
			userId: string;
			limit?: number;
			offset?: number | null;
		};
		res: UserNotesChart;
	};

	[ENDPOINTS.adminFederationMode]: {
		req: { mode: "public" | "allowlist" | "none" };
		res: FederationModeResponse;
	};
	[ENDPOINTS.adminFederationAllowlist]: {
		req: { host?: string; remove?: boolean };
		res: AllowlistResponse;
	};

	[ENDPOINTS.adminInviteCreate]: {
		req: {
			count?: number;
			expiresAt?: string | null;
		};
		res: InviteCode[];
	};
	[ENDPOINTS.adminInviteList]: {
		req: {
			limit?: number;
			offset?: number;
			type?: "all" | "unused" | "used" | "expired";
			sort?:
				| "+createdAt"
				| "-createdAt"
				| "+usedAt"
				| "-usedAt";
		};
		res: InviteCode[];
	};
	[ENDPOINTS.adminInviteDelete]: {
		req: { inviteId: string };
		res: Record<string, never>;
	};

	[ENDPOINTS.adminIPBansList]: {
		req: { limit?: number; offset?: number };
		res: IPBan[];
	};
	[ENDPOINTS.adminIPBansCreate]: {
		req: { ip: string; reason?: string; expiresAt?: string | null };
		res: IPBan;
	};
	[ENDPOINTS.adminIPBansDelete]: {
		req: { id: string };
		res: Record<string, never>;
	};
	[ENDPOINTS.adminSigninHistory]: {
		req: { limit?: number; offset?: number; userId?: string };
		res: AdminSigninEntry[];
	};
}

export type EndpointParams<E extends EndpointName> = EndpointMap[E]["req"];
export type EndpointResult<E extends EndpointName> = EndpointMap[E]["res"];

/** Endpoints that page components paginate over. */
export type PagingEndpoint =
	| typeof ENDPOINTS.notesTimeline
	| typeof ENDPOINTS.notesLocalTimeline
	| typeof ENDPOINTS.notesHybridTimeline
	| typeof ENDPOINTS.notesGlobalTimeline
	| typeof ENDPOINTS.notesUserListTimeline
	| typeof ENDPOINTS.notifications
	| typeof ENDPOINTS.iNotifications
	| typeof ENDPOINTS.iFavorites
	| typeof ENDPOINTS.usersNotes
	| typeof ENDPOINTS.usersFollowers
	| typeof ENDPOINTS.usersFollowing
	| typeof ENDPOINTS.notesReplies
	| typeof ENDPOINTS.notesRenotes
	| typeof ENDPOINTS.notesChildren
	| typeof ENDPOINTS.notesReactions
	| typeof ENDPOINTS.driveFiles
	| typeof ENDPOINTS.driveFolders
	| typeof ENDPOINTS.clipsList
	| typeof ENDPOINTS.clipsNotes
	| typeof ENDPOINTS.usersClips
	| typeof ENDPOINTS.antennasNotes
	| typeof ENDPOINTS.channelsTimeline
	| typeof ENDPOINTS.channelsFollowed
	| typeof ENDPOINTS.channelsOwned
	| typeof ENDPOINTS.channelsSearch
	| typeof ENDPOINTS.galleryPosts
	| typeof ENDPOINTS.galleryFeatured
	| typeof ENDPOINTS.iGalleryPosts
	| typeof ENDPOINTS.iGalleryLikes
	| typeof ENDPOINTS.usersGalleryPosts
	| typeof ENDPOINTS.adminEmojiList;
