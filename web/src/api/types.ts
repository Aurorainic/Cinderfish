// Entity shapes returned by the Cinderfish API.
//
// Source of truth: docs/research/api-contract/*.md. Fields the backend always
// emits with a constant value are typed narrowly (e.g. `null`) so the compiler
// stops page code from reading data that does not exist. Optional keys are
// marked with `?` — the contract docs call out exactly when each one appears.

export type NoteVisibility = "public" | "home" | "followers" | "specified";

export type NotificationType =
	| "mention"
	| "reply"
	| "renote"
	| "quote"
	| "reaction"
	| "note"
	| "follow"
	| "receiveFollowRequest";

export type FollowVisibility = "public" | "followers" | "private";

export interface ProfileField {
	name: string;
	value: string;
}

export interface UserLite {
	id: string;
	name: string | null;
	username: string;
	host: string | null;
	avatarUrl: string | null;
	avatarBlurhash: null;
	avatarDecorations: unknown[];
	isBot: boolean;
	isCat: boolean;
	requireSigninToViewContents: boolean;
	emojis: unknown[];
	onlineStatus: string;
	badgeRoles: unknown[];
	instance: null;
	/** The account this one migrated to; only detail endpoints fill it. */
	movedToUri: UserLite | null;
	/** Actor URIs this account declares as aliases of itself. */
	alsoKnownAs: string[] | null;
}

export interface UserDetailed extends UserLite {
	url: string;
	uri: string;
	createdAt: string;
	updatedAt: null;
	lastFetchedAt: null;
	bannerUrl: null;
	bannerBlurhash: null;
	isLocked: boolean;
	isModerator: boolean;
	isAdmin: boolean;
	isSilenced: boolean;
	isSuspended: boolean;
	description: string | null;
	location: string | null;
	birthday: string | null;
	lang: string | null;
	fields: ProfileField[];
	verifiedLinks: unknown[];
	followersCount: number;
	followingCount: number;
	notesCount: number;
	pinnedNoteIds: string[];
	pinnedNotes: Note[];
	pinnedPageId: null;
	pinnedPage: null;
	publicReactions: boolean;
	followersVisibility: FollowVisibility;
	followingVisibility: FollowVisibility;
	twoFactorEnabled: boolean;
	usePasswordLessLogin: boolean;
	securityKeys: boolean;
	chatScope: string;
	canChat: boolean;
	roles: unknown[];
	memo: null;
	isFollowing: boolean;
	isFollowed: boolean;
	hasPendingFollowRequestFromYou: boolean;
	hasPendingFollowRequestToYou: boolean;
}

export interface MeDetailed extends UserDetailed {
	avatarId: null;
	bannerId: null;
	followedMessage: null;
	injectFeaturedNote: boolean;
	receiveAnnouncementEmail: boolean;
	alwaysMarkNsfw: boolean;
	autoSensitive: boolean;
	carefulBot: boolean;
	autoAcceptFollowed: boolean;
	followApprovalLocalSeconds: number;
	followApprovalRemoteSeconds: number;
	noCrawle: boolean;
	preventAiLearning: boolean;
	isExplorable: boolean;
	isDeleted: boolean;
	twoFactorBackupCodesStock: string;
	hideOnlineStatus: boolean;
	hasUnreadSpecifiedNotes: boolean;
	hasUnreadMentions: boolean;
	hasUnreadChatMessages: boolean;
	hasUnreadAnnouncement: boolean;
	unreadAnnouncements: unknown[];
	hasUnreadAntenna: boolean;
	hasUnreadChannel: boolean;
	hasUnreadNotification: boolean;
	hasPendingReceivedFollowRequest: boolean;
	unreadNotificationsCount: number;
	mutedWords: unknown[];
	hardMutedWords: unknown[];
	mutedInstances: string[];
	mutingNotificationTypes: string[];
	notificationRecieveConfig: Record<string, unknown>;
	emailNotificationTypes: string[];
	achievements: unknown[];
	loggedInDays: number;
	policies: Record<string, unknown>;
	/** Only present when the request used a 16-char native token. */
	email?: string | null;
	emailVerified?: boolean;
	securityKeysList?: unknown[];
}

export interface DriveFile {
	id: string;
	createdAt: string;
	name: string;
	type: string;
	size: number;
	isSensitive: boolean;
	url: string;
	/** Preview URL; null when the file is not an image or has no thumbnail. */
	thumbnailUrl: string | null;
	webpublicUrl: null;
	comment: string | null;
	folderId: string | null;
	blurhash: null;
}

export interface DriveFolder {
	id: string;
	createdAt: string;
	name: string;
	parentId: string | null;
	foldersCount: number;
	filesCount: number;
	parent: null;
}

export interface PollChoice {
	text: string;
	votes: number;
	isVoted: boolean;
}

export interface Poll {
	multiple: boolean;
	/** Unix milliseconds, or `null` for a poll without an expiry. */
	expiresAt: number | null;
	choices: PollChoice[];
	votesCount: number;
}

export interface Note {
	id: string;
	createdAt: string;
	updatedAt: string | null;
	userId: string;
	user: UserLite;
	text: string | null;
	cw: string | null;
	visibility: NoteVisibility;
	localOnly: boolean;
	renoteCount: number;
	repliesCount: number;
	reactionCount: number;
	reactions: Record<string, number>;
	reactionEmojis: string[];
	emojis: unknown[];
	fileIds: string[];
	files: DriveFile[];
	replyId: string | null;
	renoteId: string | null;
	mentions: string[];
	tags: string[];
	uri: string | null;
	url: string | null;
	visibleUserIds?: string[];
	reactionAcceptance?: string;
	/** Set on a note posted into a channel; the note then never federates. */
	channelId?: string | null;
	/** Embedded reference to {@link ChannelRef}; present when `channelId` is. */
	channel?: ChannelRef | null;
}

export interface NoteDetailed extends Note {
	clippedCount: number;
	reply?: Note | null;
	renote?: Note | null;
	hasPoll?: true;
	poll?: Poll;
	myReaction?: string;
}

export interface Notification {
	id: string;
	createdAt: string;
	type: NotificationType;
	isRead: boolean;
	userId: string | null;
	user?: UserLite;
	noteId: string | null;
	note?: NoteDetailed;
	reaction: string | null;
}

export interface Hashtag {
	tag: string;
	mentionedUsersCount: number;
	mentionedLocalUsersCount: number;
	mentionedRemoteUsersCount: number;
	attachedUsersCount: number;
	attachedLocalUsersCount: number;
	attachedRemoteUsersCount: number;
}

export interface Following {
	id: string;
	createdAt: string;
	followerId: string;
	followeeId: string;
	follower?: UserLite;
	followee?: UserLite;
}

export interface FollowRequest {
	id: string;
	follower: UserLite;
	followee: UserLite;
}

/** One user list (`users/lists/*`). */
export interface UserList {
	id: string;
	createdAt: string;
	name: string;
	isPublic: boolean;
	/** Members are returned only on the owner's view (`users/lists/list`). */
	userIds?: string[];
	/** Present on the `forPublic` branch of `users/lists/show` only. */
	likedCount?: number;
	isLiked?: boolean;
}

/** One member row of `users/lists/get-memberships`. */
export interface UserListMembership {
	id: string;
	createdAt: string;
	userId: string;
	user: UserLite;
	withReplies: boolean;
}

/** One clip (`clips/*`). */
export interface Clip {
	id: string;
	createdAt: string;
	lastClippedAt: string | null;
	userId: string;
	user: UserLite;
	name: string;
	description: string | null;
	isPublic: boolean;
	favoritedCount: number;
	/** Present only when the request carries a credential. */
	isFavorited?: boolean;
	/** Present only for the clip owner. */
	notesCount?: number;
}

/**
 * The channel reference embedded in a packed note (`note.channel`). It carries
 * only what the timeline needs to link the note back to its channel; the
 * counters and viewer flags live on the full {@link Channel} shape.
 */
export interface ChannelRef {
	id: string;
	name: string;
	color: string;
	isSensitive: boolean;
	allowRenoteToExternal: boolean;
	/** `null` for the instance-owned channels the backend can create. */
	userId: string | null;
}

/** One channel (`channels/show`, `channels/featured`, …). */
export interface Channel {
	id: string;
	createdAt: string;
	lastNotedAt: string | null;
	name: string;
	description: string | null;
	userId: string | null;
	bannerUrl: string | null;
	bannerId: string | null;
	pinnedNoteIds: string[];
	color: string;
	isArchived: boolean;
	usersCount: number;
	notesCount: number;
	isSensitive: boolean;
	allowRenoteToExternal: boolean;
	/** Viewer flags; present only when the request carried a credential. */
	isFollowing?: boolean;
	isFavorited?: boolean;
	isMuting?: boolean;
}

/** `channels/show` additionally packs the pinned notes in pin order. */
export interface ChannelShow extends Channel {
	pinnedNotes: NoteDetailed[];
}

/**
 * The antenna source values the backend implements. Firefish also has `group`
 * and `instances`; neither exists in the Misskey contract (`antennas/create`),
 * so the editor must not offer them.
 */
export type AntennaSrc =
	| "home"
	| "all"
	| "users"
	| "list"
	| "users_blacklist";

/** One packed antenna (`antennas/show`, `antennas/list`). */
export interface Antenna {
	id: string;
	createdAt: string;
	name: string;
	/** Each inner array is an AND-group; the outer array is OR. */
	keywords: string[][];
	excludeKeywords: string[][];
	src: AntennaSrc;
	userListId: string | null;
	users: string[];
	caseSensitive: boolean;
	localOnly: boolean;
	excludeBots: boolean;
	withReplies: boolean;
	withFile: boolean;
	isActive: boolean;
	/** Packed as false until the backend tracks unread state. */
	hasUnreadNote: boolean;
	/** Packed as false; notification delivery is not implemented yet. */
	notify: boolean;
	excludeNotesInSensitiveChannel: boolean;
}

export interface NoteFavorite {
	id: string;
	createdAt: string;
	noteId: string;
	note: Note;
}

export interface NoteReaction {
	id: string;
	createdAt: string;
	type: string;
	user: UserLite;
}

/** `users/reactions`: a reaction together with the note it belongs to. */
export interface NoteReactionWithNote extends NoteReaction {
	note: Note;
}

export interface EmojiSimple {
	name: string;
	category: string | null;
	aliases: string[];
	url: string;
	localOnly: boolean;
	isSensitive: boolean;
}

export interface AdminEmoji extends EmojiSimple {
	id: string;
	host: string | null;
	createdAt: string;
}

export interface HashtagTrend {
	tag: string;
	chart: number[];
	usersCount: number;
}

export interface MetaFeatures {
	localTimeline: boolean;
	globalTimeline: boolean;
	registration: boolean;
	emailRequiredForSignup: boolean;
	hcaptcha: boolean;
	recaptcha: boolean;
	turnstile: boolean;
	objectStorage: boolean;
	serviceWorker: boolean;
	miauth: boolean;
}

export interface MetaLite {
	maintainerName: string;
	maintainerEmail: string;
	version: string;
	providesTarball: boolean;
	name: string;
	shortName: string;
	uri: string;
	description: string;
	langs: string[];
	tosUrl: string | null;
	repositoryUrl: string;
	feedbackUrl: string;
	impressumUrl: string;
	privacyPolicyUrl: string;
	inquiryUrl: string;
	disableRegistration: boolean;
	emailRequiredForSignup: boolean;
	enableHcaptcha: boolean;
	hcaptchaSiteKey: string;
	enableMcaptcha: boolean;
	mcaptchaSiteKey: string;
	mcaptchaInstanceUrl: string;
	enableRecaptcha: boolean;
	recaptchaSiteKey: string;
	enableTurnstile: boolean;
	turnstileSiteKey: string;
	enableTestcaptcha: boolean;
	googleAnalyticsMeasurementId: string;
	swPublickey: string;
	themeColor: string;
	mascotImageUrl: string;
	bannerUrl: string;
	infoImageUrl: string;
	serverErrorImageUrl: string;
	notFoundImageUrl: string;
	iconUrl: string;
	backgroundImageUrl: string;
	logoImageUrl: string;
	maxNoteTextLength: number;
	defaultLightTheme: null;
	defaultDarkTheme: null;
	clientOptions: Record<string, unknown>;
	ads: unknown[];
	notesPerOneAd: number;
	enableEmail: boolean;
	enableServiceWorker: boolean;
	translatorAvailable: boolean;
	serverRules: string[];
	policies: Record<string, unknown>;
	sentryForFrontend: null;
	mediaProxy: string;
	enableUrlPreview: boolean;
	noteSearchableScope: string;
	maxFileSize: number;
	/** Misskey vocabulary: `all` | `specified` | `none`. */
	federation: "all" | "specified" | "none";
}

export interface MetaDetailed extends MetaLite {
	/** Instance-wide stylesheet an administrator saved from the admin console. */
	customCss: string;
	/** SMTP host, or an empty string when mail delivery is not configured. */
	emailServer: string;
	cacheRemoteFiles: boolean;
	cacheRemoteSensitiveFiles: boolean;
	requireSetup: boolean;
	proxyAccountName: string;
	features: MetaFeatures;
}

export interface PingResponse {
	pong: number;
}

export interface RegistryKeyValue {
	key: string;
	value: unknown;
	scope: string[];
	domain: string | null;
}

export interface RegistryDetail {
	updatedAt: string;
	value: unknown;
}

export interface AllowlistResponse {
	hosts: string[];
}

export interface FederationModeResponse {
	mode: "public" | "allowlist" | "none";
}

export interface SignupResponse extends MeDetailed {
	token: string;
}

export interface SigninResponse {
	finished: true;
	id: string;
	i: string;
}

/**
 * Two-factor handshake: a signin with the right password but no `token`
 * answers an empty body, and the client repeats the request with the code.
 */
export type SigninStep = SigninResponse | Record<string, never>;

export interface TwoFactorRegistration {
	qr: string;
	url: string;
	secret: string;
	label: string;
	issuer: string;
}

/** Result of `notes/state`. */
export interface NoteState {
	isFavorited: boolean;
	isPinned: boolean;
	isMutedThread: boolean;
}

/** Result of `i/import`. */
export interface ImportSummary {
	notes: number;
	followings: number;
	favorites: number;
	skipped: number;
	skippedFollowings: number;
	skippedFavorites: number;
}

/** One row of `i/signin-history`. */
export interface SigninHistoryEntry {
	id: string;
	createdAt: string;
	ip: string;
	headers: Record<string, unknown>;
	success: boolean;
}

export interface NoteEditEntry {
	id: string;
	noteId: string;
	text: string | null;
	cw: string | null;
	fileIds: string[];
	createdAt: string;
}

export interface NoteEditResponse {
	createdNote: NoteDetailed;
}

export interface MiAuthTokenResponse {
	token: string;
}

export interface UserNameAvailability {
	available: boolean;
}

// users/stats: the three note buckets are disjoint, so they add up to every
// note the account has written.
export interface UserStats {
	notesCount: number;
	repliesCount: number;
	renotesCount: number;
	repliedCount: number;
	renotedCount: number;
	pollVotesCount: number;
	pollVotedCount: number;
}

// i/registry/keys-with-type: the stored value's JSON type, inferred on read.
export type RegistryValueType =
	| "string"
	| "number"
	| "boolean"
	| "object"
	| "array"
	| "null";

// email-address/available: `reason` is "format" for a malformed address and
// "used" when another account already owns it.
export interface EmailAddressAvailability {
	available: boolean;
	reason?: "format" | "used";
}

export interface NoteCreateResponse {
	createdNote: NoteDetailed;
}

export interface RefreshAccountResponse {
	token: string;
}

/** One row of `mute/list`. */
export interface MuteRecord {
	id: string;
	createdAt: string;
	mutee: UserDetailed;
}

/** One row of `blocking/list`. */
export interface BlockingRecord {
	id: string;
	createdAt: string;
	blockee: UserDetailed;
}

/** A muted instance host, stored only in the local base store. */
export interface MutedInstance {
	host: string;
}

/** One third-party application authorized by `i/apps`. */
export interface AuthorizedApp {
	id: string;
	name: string;
	description: string;
	iconUrl: string | null;
	permission: string[];
	createdAt: string;
	lastUsedAt: string | null;
}

/** A drive file as `admin/drive/files` returns it: the uploader is attached. */
export interface AdminDriveFile extends DriveFile {
	user: UserLite | null;
}

/** One instance announcement. */
export interface Announcement {
	id: string;
	title: string;
	text: string;
	imageUrl: string | null;
	isActive: boolean;
	createdAt: string;
	updatedAt: string;
	expiresAt: string | null;
	isRead: boolean;
}

/** One abuse report from `admin/abuse-user-reports`. */
export interface AbuseReport {
	id: string;
	targetUser: UserDetailed;
	reporter: UserDetailed;
	comment: string;
	resolved: boolean;
	createdAt: string;
	forwarded: boolean;
	/** Administrator-only note; empty when none was written. */
	moderationNote: string;
}

/** One outbound webhook from `i/webhooks/*`. */
export interface Webhook {
	id: string;
	userId: string;
	name: string;
	url: string;
	secret: string;
	on: string[];
	active: boolean;
	createdAt: string;
	latestSentAt: string | null;
	latestStatus: number | null;
}

/** One table's planner estimate and on-disk size. */
export interface TableStat {
	count: number;
	size: number;
}

/** One registration ticket as the admin console sees it (Packed<'InviteCode'>). */
export interface InviteCode {
	id: string;
	code: string;
	createdAt: string;
	expiresAt: string | null;
	createdBy: UserLite | null;
	usedBy: UserLite | null;
	usedAt: string | null;
	used: boolean;
}

/** One IP ban rule (Cinderfish extension; single address or CIDR). */
export interface IPBan {
	id: string;
	ip: string;
	reason: string;
	createdAt: string;
	expiresAt: string | null;
	createdBy: { id: string; username: string } | null;
}

/** One entry of the cross-account signin audit. */
export interface AdminSigninEntry {
	id: string;
	createdAt: string;
	ip: string;
	success: boolean;
	user: { id: string; username: string };
}

/** One row of `admin/get-index-stats`. */
export interface IndexStat {
	schemaname: string;
	tablename: string;
	indexname: string;
	tablespace: string | null;
	indexdef: string;
}

/** The subset of instance settings `admin/meta` exposes. */
export interface AdminInstanceMeta {
	name: string;
	description: string;
	maintainerName: string;
	maintainerEmail: string;
	registrationMode: "open" | "invite" | "admin";
	blockedHosts: string[];
	tosUrl: string | null;
	privacyPolicyUrl: string;
	repositoryUrl: string;
	feedbackUrl: string;
	inquiryUrl: string;
	impressumUrl: string;
	serverRules: string[];
	hiddenTags: string[];
	iconUrl: string | null;
	bannerUrl: string | null;
	themeColor: string | null;
	/** Instance-wide stylesheet; empty when none is set. */
	customCss: string;
	/** Usernames signup refuses; Misskey's meta.preservedUsernames. */
	preservedUsernames: string[];
	/** Runtime note-length cap in code points (default 3000). */
	maxNoteTextLength: number;
	/** Active captcha provider, "none" when disabled. */
	captchaProvider: "none" | "hcaptcha" | "recaptcha" | "turnstile";
	/** Site key of the active provider (public). */
	captchaSiteKey: string;
	/** Whether a captcha secret is stored; the value is never returned. */
	captchaSecretSet: boolean;
	/** Automatic IP ban after repeated signin failures (extension). */
	signinAutoBan: {
		enabled: boolean;
		threshold: number;
		windowMinutes: number;
		banMinutes: number;
	};
	/** SMTP host, empty when mail delivery is not configured. */
	emailServer: string;
}

export interface DriveUsage {
	capacity: number;
	usage: number;
	filesCount: number;
}

/** One row of `federation/show-instance`. */
export interface FederationInstance {
	host: string;
	name: string | null;
	description: string | null;
	softwareName: string | null;
	softwareVersion: string | null;
	openRegistrations: boolean | null;
	usersCount: number;
	notesCount: number;
	latestStatus: number | null;
	isBlocked: boolean;
	isSuspended: boolean;
	isNotResponding: boolean;
}

/** `charts/user/notes`: parallel arrays, oldest first. */
export interface UserNotesChart {
	total: number[];
	inc: number[];
	dec: number[];
	diffs: {
		normal: number[];
		reply: number[];
		renote: number[];
		withFile: number[];
	};
}

/** `fetch-rss`: the subset of a parsed feed the RSS widgets render. */
export interface FetchRssItem {
	title: string;
	link: string;
	pubDate: string;
	guid: string;
}

export interface FetchRssFeed {
	title: string;
	link: string;
	description: string;
	items: FetchRssItem[];
}

/** `server-info`: machine load for the server-metric widget. */
export interface ServerInfo {
	machine: string;
	cpu: { model: string; cores: number; usage: number; known: boolean };
	mem: { total: number; used: number; known: boolean };
	fs: { total: number; used: number; known: boolean };
	load: { one: number; five: number; fifteen: number; known: boolean };
}

/** Counters from `stats`. */
export interface InstanceStats {
	usersCount: number;
	notesCount: number;
	originalNotesCount: number;
	originalUsersCount: number;
	reactionsCount: number;
	followingCount: number;
	instancesCount: number;
	driveFiles: number;
	driveUsage: number;
}

/** Read-only media configuration from `admin/object-storage`. */
export interface ObjectStorageInfo {
	backend: string;
	localPath: string | null;
	s3: {
		endpoint: string;
		bucket: string;
		region: string;
		useSSL: boolean;
		publicBaseUrl: string | null;
		caCert: string | null;
		accessKeySet: boolean;
		secretKeySet: boolean;
		hasCredentials: boolean;
	} | null;
	/** True once the admin console saved a runtime override. */
	stored: boolean;
}

/** Effective SMTP settings from `admin/email-settings` (password masked). */
export interface MailSettingsInfo {
	host: string;
	port: number;
	username: string;
	passwordSet: boolean;
	from: string | null;
	tls: "starttls" | "tls" | "none";
	configured: boolean;
	/** True once the admin console saved a runtime override. */
	stored: boolean;
}

export interface DeliverQueueEntry {
	id: number;
	host: string;
	attempts: number;
	nextAttemptAt: string;
	lastError: string | null;
}

/** Health of the PostgreSQL delivery queue. */
export interface DeliverQueueInfo {
	pending: number;
	dueNow: number;
	failed: number;
	oldestNextAt: string | null;
	recent: DeliverQueueEntry[];
}
