// Ported from firefish@b425f61:packages/client/src/store.ts (AGPL-3.0-only)
//
// The one and only "base" bucket. Firefish keeps a single
// defaultStore = new Storage("base", {...}); the first-wave port ended up with
// two (store/base.ts and store/store.ts) sharing the same bucket name, which is
// a bug: two Storage instances would read/write the same pizzax::base keys and
// would not see each other's writes. This file is the merge point, so
// "@/store", "@/store/base" and "@/store/store" all resolve to the same object.
//
// Key names, values and persistence scopes follow Firefish. Cinderfish-only
// keys are at the end of the definition and marked as such.
import { markRaw } from "vue";
import { api } from "../api/client";
import type { Note, NoteDetailed, NoteVisibility, UserLite } from "../api/types";
import type { NotesCreateParams } from "../api/endpoints";
import { darkTheme, lightTheme } from "../themes";
import { Storage, type PizzaxOptions, type PizzaxRemote, type StateDef } from "./pizzax";

/**
 * Hook arrays Firefish exposes for plugins/experiments. Nothing in Cinderfish
 * registers into them yet; ported call sites read them so the seam survives.
 */
export const postFormActions: {
	title: string;
	handler: (from: unknown, update: (value: unknown) => void) => void | Promise<void>;
}[] = [];
export const userActions: {
	title: string;
	handler: (user: UserLite) => void | Promise<void>;
}[] = [];
export const noteActions: {
	title: string;
	handler: (note: NoteDetailed) => void | Promise<void>;
}[] = [];
export const noteViewInterruptors: {
	handler: (note: Note) => Promise<Note>;
}[] = [];
export const notePostInterruptors: {
	handler: (note: NotesCreateParams) => Promise<NotesCreateParams>;
}[] = [];

export const menuOptions = [
	"notifications",
	"followRequests",
	"messaging",
	"explore",
	"favorites",
	"channels",
	"search",
] as const;

export type MenuOption = (typeof menuOptions)[number];

/** Firefish's built-in reaction palette (store.ts "reactions" default). */
export const defaultReactions = [
	"⭐",
	"❤️",
	"😆",
	"🤔",
	"😮",
	"🎉",
	"💢",
	"😥",
	"😇",
	"🥴",
	"🍮",
];

export type TimelineSource = "home" | "local" | "social" | "global";

export type NoteVisibilityPreference = NoteVisibility;

export interface StatusbarItem {
	name: string;
	id: string;
	type: string;
	size: "verySmall" | "small" | "medium" | "large" | "veryLarge";
	black: boolean;
	props: Record<string, unknown>;
}

/** Firefish spells it StatusbarItem; the first-wave port used StatusBarItem. */
export type StatusBarItem = StatusbarItem;

export interface WidgetItem {
	name: string;
	id: string;
	place: string | null;
	data: Record<string, unknown>;
}

export interface TimelineMemory {
	src: TimelineSource;
	arg: string | null;
}

const baseDef = {
	// ----- Firefish store.ts, in source order -------------------------------
	tutorial: { where: "account", default: 0 },
	// Scope note: the reference snapshot has all five hint flags on "device"
	// (store.ts:29-48). The first-wave port had them on "account"; "device" is
	// kept here so the behaviour matches Firefish (guests can dismiss a hint).
	tlHomeHintClosed: { where: "device", default: false },
	tlLocalHintClosed: { where: "device", default: false },
	tlRecommendedHintClosed: { where: "device", default: false },
	tlSocialHintClosed: { where: "device", default: false },
	tlGlobalHintClosed: { where: "device", default: false },
	keepCw: { where: "account", default: true },
	showFullAcct: { where: "account", default: false },
	rememberNoteVisibility: { where: "account", default: false },
	defaultNoteVisibility: {
		where: "account",
		default: "public" as NoteVisibilityPreference,
	},
	defaultNoteLocalOnly: { where: "account", default: false },
	uploadFolder: { where: "account", default: null as string | null },
	pastedFileName: { where: "account", default: "yyyy-MM-dd HH-mm-ss [{{number}}]" },
	keepOriginalUploading: { where: "account", default: false },
	memo: { where: "account", default: null as string | null },
	// Firefish's default is the palette above; the first-wave port had a
	// different list in store.ts. The reference list wins.
	reactions: { where: "account", default: defaultReactions.slice() },
	// Firefish types this as a bare []; the settings page writes
	// (string | string[])[] word groups, so that is the useful type here.
	mutedWords: { where: "account", default: [] as (string | string[])[] },
	mutedAds: { where: "account", default: [] as string[] },
	showAds: { where: "account", default: true },
	menu: { where: "deviceAccount", default: [...menuOptions] as MenuOption[] },
	visibility: {
		where: "deviceAccount",
		default: "public" as NoteVisibilityPreference,
	},
	localOnly: { where: "deviceAccount", default: false },
	statusbars: { where: "deviceAccount", default: [] as StatusbarItem[] },
	widgets: { where: "deviceAccount", default: [] as WidgetItem[] },
	tl: {
		where: "deviceAccount",
		default: { src: "home", arg: null } as TimelineMemory,
	},
	overridedDeviceKind: {
		where: "device",
		default: null as null | "smartphone" | "tablet" | "desktop",
	},
	serverDisconnectedBehavior: {
		where: "device",
		default: "nothing" as "nothing" | "quiet" | "reload" | "dialog",
	},
	seperateRenoteQuote: { where: "device", default: true },
	expandOnNoteClick: { where: "device", default: true },
	nsfw: { where: "device", default: "respect" as "respect" | "force" | "ignore" },
	animation: { where: "device", default: true },
	advancedMfm: { where: "device", default: true },
	animatedMfm: { where: "device", default: true },
	animatedMfmWarnShown: { where: "device", default: false },
	loadRawImages: { where: "device", default: false },
	imageNewTab: { where: "device", default: false },
	disableShowingAnimatedImages: { where: "device", default: false },
	disablePagesScript: { where: "device", default: false },
	useOsNativeEmojis: { where: "device", default: false },
	disableDrawer: { where: "device", default: false },
	useBlurEffectForModal: { where: "device", default: true },
	useBlurEffect: { where: "device", default: true },
	showFixedPostForm: { where: "device", default: false },
	enableInfiniteScroll: { where: "device", default: true },
	useReactionPickerForContextMenu: { where: "device", default: false },
	showGapBetweenNotesInTimeline: { where: "device", default: true },
	darkMode: { where: "device", default: false },
	instanceTicker: {
		where: "device",
		default: "remote" as "none" | "remote" | "always",
	},
	reactionPickerSkinTone: { where: "account", default: 1 },
	reactionPickerSize: { where: "device", default: 3 },
	reactionPickerWidth: { where: "device", default: 3 },
	reactionPickerHeight: { where: "device", default: 3 },
	reactionPickerUseDrawerForMobile: { where: "device", default: true },
	recentlyUsedEmojis: { where: "device", default: [] as string[] },
	recentlyUsedUsers: { where: "device", default: [] as string[] },
	defaultSideView: { where: "device", default: false },
	menuDisplay: {
		where: "device",
		default: "sideFull" as "sideFull" | "sideIcon" | "top",
	},
	reportError: { where: "device", default: false },
	squareAvatars: { where: "device", default: true },
	postFormWithHashtags: { where: "device", default: false },
	postFormHashtags: { where: "device", default: "" },
	themeInitial: { where: "device", default: true },
	numberOfPageCache: { where: "device", default: 5 },
	enterSendsMessage: { where: "device", default: true },
	showUpdates: { where: "device", default: true },
	swipeOnDesktop: { where: "device", default: false },
	swipeOnMobile: { where: "device", default: true },
	showAdminUpdates: { where: "account", default: true },
	woozyMode: { where: "device", default: false },
	enableCustomKaTeXMacro: { where: "device", default: false },
	enableEmojiReactions: { where: "account", default: true },
	showEmojisInReactionNotifications: { where: "account", default: true },
	showTimelineReplies: { where: "device", default: false },

	// ----- Cinderfish-only keys ---------------------------------------------
	// deck is the shell switch read by main.ts (null = universal layout).
	deck: { where: "deviceAccount", default: null as string | null },
	// Firefish keeps custom CSS in a bare localStorage key; here it is a
	// per-account store entry so several accounts on one browser do not share
	// it. `composables/use-custom-css.ts` applies it live.
	customCss: { where: "account", default: "" },

	// Keys the first-wave port already shipped without an upstream counterpart
	// in this snapshot. Kept so existing pages/settings keep compiling.
	mutedLangs: { where: "account", default: [] as string[] },
	squareCatAvatars: { where: "device", default: false },
	addRe: { where: "account", default: true },
	detectPostLanguage: { where: "deviceAccount", default: true },
	openServerInfo: { where: "device", default: true },
	iconSet: {
		where: "device",
		default: "ph-bold" as "ph-bold" | "ph-duotone" | "ph-light" | "ph" | "ph-fill",
	},
	recentlyUsedPostLanguages: { where: "account", default: [] as string[] },
	useEmojiCdn: { where: "device", default: true },
	showPreviewByDefault: { where: "deviceAccount", default: false },
	hideFollowButtons: { where: "device", default: true },
	replaceChatButtonWithAccountButton: { where: "device", default: true },
	replaceWidgetsButtonWithReloadButton: { where: "device", default: false },
	searchURL: { where: "device", default: "https://duckduckgo.com/?q=" },
	showBigPostButton: { where: "device", default: false },
	enableTimelineStreaming: { where: "deviceAccount", default: true },
	enablePullToRefresh: { where: "deviceAccount", default: true },
	pullToRefreshThreshold: { where: "device", default: 150 },
	showNoAltTextWarning: { where: "account", default: true },
	showAddFileDescriptionAtFirstPost: { where: "account", default: false },
	addAlt4MeTag: { where: "account", default: false },
	autocorrectNoteLanguage: { where: "account", default: false },
	foldNotification: { where: "deviceAccount", default: true },
	mergeThreadInTimeline: { where: "deviceAccount", default: true },
	mergeRenotesInTimeline: { where: "deviceAccount", default: true },
	writingMode: {
		where: "deviceAccount",
		default: "horizontal-tb" as
			| "horizontal-tb"
			| "vertical-rl"
			| "vertical-lr"
			| "vertical-rl-upright"
			| "vertical-lr-upright",
	},
} satisfies StateDef;

export type BaseStoreDefinition = typeof baseDef;

let currentUserId: string | null = null;

/** Called by the account layer whenever the signed-in user changes. */
export function setStoreAccount(userId: string | null): void {
	currentUserId = userId;
}

const registryRemote: PizzaxRemote = {
	fetchAll: async (bucket) =>
		api<Record<string, unknown>>("i/registry/get-all", { scope: ["client", bucket] }),
	set: async (bucket, key, value) => {
		await api("i/registry/set", { scope: ["client", bucket], key, value });
	},
};

const storeOptions: PizzaxOptions = {
	getUserId: () => currentUserId,
	remote: registryRemote,
};

/**
 * markRaw mirrors Firefish: the Storage instance itself must never become a
 * reactive proxy (its state/reactiveState already are).
 */
export const store = markRaw(new Storage("base", baseDef, storeOptions));

/** Firefish name, kept for ports that still say defaultStore. */
export const defaultStore = store;

export type Store = typeof store;

// Re-export themes for callers that only touch the store.
export { lightTheme, darkTheme };
