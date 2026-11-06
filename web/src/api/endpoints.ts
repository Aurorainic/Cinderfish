// Endpoint constants and request/response types for every route registered by
// `internal/httpapi/server.go`.
//
// Paths are relative to `/api` unless noted. They are kept as a const object
// (not string literals at call sites) so a route rename is a single edit and
// so `EndpointName` can drive `api()`'s typing.

import type {
	AbuseReport,
	AdminDriveFile,
	Announcement,
	AdminEmoji,
	AuthorizedApp,
	BlockingRecord,
	AllowlistResponse,
	DriveFile,
	DriveFolder,
	EmojiSimple,
	FederationModeResponse,
	FollowRequest,
	Following,
	HashtagTrend,
	IndexStat,
	MeDetailed,
	MetaDetailed,
	MetaLite,
	MiAuthTokenResponse,
	MuteRecord,
	Note,
	NoteCreateResponse,
	NoteDetailed,
	NoteFavorite,
	NoteReaction,
	Notification,
	NoteVisibility,
	PingResponse,
	RegistryDetail,
	SigninResponse,
	SignupResponse,
	TableStat,
	UserNameAvailability,
	UserDetailed,
	UserLite,
	Webhook,
} from "./types";

export const ENDPOINTS = {
	// meta / ping
	meta: "meta",
	ping: "ping",

	// auth
	signup: "signup",
	signin: "signin",
	usernameAvailable: "username/available",
	miAuthGenToken: "miauth/gen-token",
	iApps: "i/apps",
	iExport: "i/export",
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
	iFavorites: "i/favorites",
	iNotifications: "i/notifications",

	// users
	usersShow: "users/show",
	usersSearch: "users/search",
	usersReportAbuse: "users/report-abuse",
	usersNotes: "users/notes",
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
	registryScopes: "i/registry/scopes",
	registryGetAll: "i/registry/get-all",
	registryRemove: "i/registry/remove",
	registryClear: "i/registry/clear",

	// notes
	notesCreate: "notes/create",
	notesShow: "notes/show",
	notesDelete: "notes/delete",
	notesReplies: "notes/replies",
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

	// timelines
	notesTimeline: "notes/timeline",
	notesLocalTimeline: "notes/local-timeline",
	notesHybridTimeline: "notes/hybrid-timeline",
	notesGlobalTimeline: "notes/global-timeline",

	// notifications
	announcements: "announcements",
	adminAbuseReports: "admin/abuse-user-reports",
	adminTableStats: "admin/get-table-stats",
	adminIndexStats: "admin/get-index-stats",
	adminResolveAbuseReport: "admin/resolve-abuse-user-report",
	adminAnnouncementsCreate: "admin/announcements/create",
	adminAnnouncementsList: "admin/announcements/list",
	adminAnnouncementsDelete: "admin/announcements/delete",

	notifications: "notifications",
	notificationsMarkAllAsRead: "notifications/mark-all-as-read",

	// drive
	driveFiles: "drive/files",
	driveFilesCreate: "drive/files/create",
	driveFilesShow: "drive/files/show",
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
	adminSuspendUser: "admin/suspend-user",
	adminUnsuspendUser: "admin/unsuspend-user",
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
	[ENDPOINTS.signin]: { req: SigninParams; res: SigninResponse };
	[ENDPOINTS.usernameAvailable]: {
		req: { username: string };
		res: UserNameAvailability;
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
	[ENDPOINTS.iFavorites]: {
		req: PaginationParams;
		res: NoteFavorite[];
	};
	[ENDPOINTS.iNotifications]: {
		req: NotificationsParams;
		res: Notification[];
	};

	[ENDPOINTS.usersShow]: { req: UsersShowParams; res: UserDetailed };
	[ENDPOINTS.usersNotes]: { req: UsersNotesParams; res: NoteDetailed[] };
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

	[ENDPOINTS.registrySet]: { req: RegistryParams; res: EmptyResponse };
	[ENDPOINTS.registryGet]: { req: RegistryGetParams; res: unknown };
	[ENDPOINTS.registryGetDetail]: {
		req: RegistryGetParams;
		res: RegistryDetail;
	};
	[ENDPOINTS.registryKeys]: { req: RegistryGetParams; res: string[] };
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
	[ENDPOINTS.notesDelete]: { req: NoteIdParams; res: EmptyResponse };
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
	[ENDPOINTS.hashtagsTrend]: { req: Record<string, never>; res: HashtagTrend[] };

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
	[ENDPOINTS.adminAnnouncementsDelete]: {
		req: { id: string };
		res: EmptyResponse;
	};

	[ENDPOINTS.notificationsMarkAllAsRead]: {
		req: Record<string, never>;
		res: EmptyResponse;
	};

	[ENDPOINTS.driveFiles]: { req: DriveFilesParams; res: DriveFile[] };
	[ENDPOINTS.driveFilesCreate]: {
		req: { folderId?: string | null };
		res: DriveFile;
	};
	[ENDPOINTS.driveFilesShow]: { req: { fileId: string }; res: DriveFile };
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

	[ENDPOINTS.adminFederationMode]: {
		req: { mode: "public" | "allowlist" | "none" };
		res: FederationModeResponse;
	};
	[ENDPOINTS.adminFederationAllowlist]: {
		req: { host?: string; remove?: boolean };
		res: AllowlistResponse;
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
	| typeof ENDPOINTS.adminEmojiList;
