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
		endpointHost: string;
		bucket: string;
		region: string;
		useSSL: boolean;
		publicBaseUrl: string | null;
		hasCredentials: boolean;
	} | null;
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
