import { markRaw } from "vue";
import type { Note, NoteDetailed, NoteVisibility, UserLite } from "../api/types";
import type { NotesCreateParams } from "../api/endpoints";
import { createPizzax } from "./pizzax";

/**
 * Port of Firefish `packages/client/src/store.ts`.
 *
 * Bucket name ("base") and every key are preserved so future page ports can
 * bind directly: `defaultStore.state.showFixedPostForm`, etc. Account-scoped
 * keys continue to go through `i/registry/*`; device keys stay local.
 */

/** Hook arrays Firefish exposes for plugins/experiments. */
const signedIn = (() => {
	try {
		return localStorage.getItem("account") != null;
	} catch {
		return false;
	}
})();

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

const menuOptions = [
	"notifications",
	"followRequests",
	"messaging",
	"explore",
	"favorites",
	"channels",
	"search",
];

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
	"🔥",
	"🐟",
];

export interface StatusbarItem {
	name: string;
	id: string;
	type: string;
	size: "verySmall" | "small" | "medium" | "large" | "veryLarge";
	black: boolean;
	props: Record<string, unknown>;
}

export interface WidgetItem {
	name: string;
	id: string;
	place: string | null;
	data: Record<string, unknown>;
}

export const defaultStore = markRaw(
	createPizzax("base", {
		tutorial: {
			where: "account" as const,
			default: 0,
		},
		tlHomeHintClosed: {
			where: "account" as const,
			default: false,
		},
		tlLocalHintClosed: {
			where: "account" as const,
			default: false,
		},
		tlRecommendedHintClosed: {
			where: "account" as const,
			default: false,
		},
		tlSocialHintClosed: {
			where: "account" as const,
			default: false,
		},
		tlGlobalHintClosed: {
			where: "account" as const,
			default: false,
		},
		keepCw: {
			where: "account" as const,
			default: true,
		},
		showFullAcct: {
			where: "account" as const,
			default: false,
		},
		rememberNoteVisibility: {
			where: "account" as const,
			default: false,
		},
		defaultNoteVisibility: {
			where: "account" as const,
			default: "public" as NoteVisibility,
		},
		defaultNoteLocalOnly: {
			where: "account" as const,
			default: false,
		},
		uploadFolder: {
			where: "account" as const,
			default: null as string | null,
		},
		pastedFileName: {
			where: "account" as const,
			default: "yyyy-MM-dd HH-mm-ss [{{number}}]",
		},
		keepOriginalUploading: {
			where: "account" as const,
			default: false,
		},
		memo: {
			where: "account" as const,
			default: null as string | null,
		},
		reactions: {
			where: "account" as const,
			default: defaultReactions,
		},
		mutedWords: {
			where: "account" as const,
			default: [] as string[][],
		},
		mutedLangs: {
			where: "account" as const,
			default: [] as string[],
		},
		mutedAds: {
			where: "account" as const,
			default: [] as string[],
		},
		showAds: {
			where: "account" as const,
			default: true,
		},
		menu: {
			where: "deviceAccount" as const,
			default: menuOptions as string[],
		},
		visibility: {
			where: "deviceAccount" as const,
			default: "public" as NoteVisibility,
		},
		localOnly: {
			where: "deviceAccount" as const,
			default: false,
		},
		statusbars: {
			where: "deviceAccount" as const,
			default: [] as StatusbarItem[],
		},
		widgets: {
			where: "deviceAccount" as const,
			default: [] as WidgetItem[],
		},
		tl: {
			where: "deviceAccount" as const,
			default: {
				src: (signedIn ? "home" : "local") as
					| "home"
					| "local"
					| "social"
					| "global"
					| "recommended",
				arg: null as string | null,
			},
		},

		overridedDeviceKind: {
			where: "device" as const,
			default: null as null | "smartphone" | "tablet" | "desktop",
		},
		serverDisconnectedBehavior: {
			where: "device" as const,
			default: "nothing" as "nothing" | "quiet" | "reload" | "dialog",
		},
		seperateRenoteQuote: {
			where: "device" as const,
			default: true,
		},
		expandOnNoteClick: {
			where: "device" as const,
			default: true,
		},
		nsfw: {
			where: "deviceAccount" as const,
			default: "respect" as "respect" | "force" | "ignore",
		},
		animation: {
			where: "device" as const,
			default: true,
		},
		advancedMfm: {
			where: "device" as const,
			default: true,
		},
		animatedMfm: {
			where: "device" as const,
			default: true,
		},
		animatedMfmWarnShown: {
			where: "device" as const,
			default: false,
		},
		loadRawImages: {
			where: "device" as const,
			default: false,
		},
		imageNewTab: {
			where: "device" as const,
			default: false,
		},
		disableShowingAnimatedImages: {
			where: "device" as const,
			default: false,
		},
		disablePagesScript: {
			where: "device" as const,
			default: false,
		},
		useOsNativeEmojis: {
			where: "device" as const,
			default: false,
		},
		disableDrawer: {
			where: "device" as const,
			default: false,
		},
		useBlurEffectForModal: {
			where: "device" as const,
			default: true,
		},
		useBlurEffect: {
			where: "device" as const,
			default: true,
		},
		showFixedPostForm: {
			where: "device" as const,
			default: false,
		},
		enableInfiniteScroll: {
			where: "device" as const,
			default: true,
		},
		useReactionPickerForContextMenu: {
			where: "device" as const,
			default: false,
		},
		showGapBetweenNotesInTimeline: {
			where: "device" as const,
			default: false,
		},
		darkMode: {
			where: "device" as const,
			default: false,
		},
		instanceTicker: {
			where: "device" as const,
			default: "always" as "none" | "remote" | "always",
		},
		reactionPickerSkinTone: {
			where: "account" as const,
			default: 1,
		},
		reactionPickerSize: {
			where: "device" as const,
			default: 3,
		},
		reactionPickerWidth: {
			where: "device" as const,
			default: 3,
		},
		reactionPickerHeight: {
			where: "device" as const,
			default: 3,
		},
		reactionPickerUseDrawerForMobile: {
			where: "device" as const,
			default: true,
		},
		recentlyUsedEmojis: {
			where: "device" as const,
			default: [] as string[],
		},
		recentlyUsedUsers: {
			where: "device" as const,
			default: [] as string[],
		},
		defaultSideView: {
			where: "device" as const,
			default: false,
		},
		menuDisplay: {
			where: "device" as const,
			default: "sideFull" as "sideFull" | "sideIcon" | "top",
		},
		reportError: {
			where: "device" as const,
			default: false,
		},
		squareAvatars: {
			where: "device" as const,
			default: true,
		},
		squareCatAvatars: {
			where: "device" as const,
			default: false,
		},
		postFormWithHashtags: {
			where: "device" as const,
			default: false,
		},
		postFormHashtags: {
			where: "device" as const,
			default: "",
		},
		themeInitial: {
			where: "device" as const,
			default: true,
		},
		numberOfPageCache: {
			where: "device" as const,
			default: 5,
		},
		enterSendsMessage: {
			where: "device" as const,
			default: true,
		},
		showUpdates: {
			where: "device" as const,
			default: true,
		},
		swipeOnDesktop: {
			where: "device" as const,
			default: false,
		},
		swipeOnMobile: {
			where: "device" as const,
			default: true,
		},
		showAdminUpdates: {
			where: "account" as const,
			default: true,
		},
		woozyMode: {
			where: "device" as const,
			default: false,
		},
		enableCustomKaTeXMacro: {
			where: "device" as const,
			default: false,
		},
		enableEmojiReactions: {
			where: "account" as const,
			default: true,
		},
		showEmojisInReactionNotifications: {
			where: "account" as const,
			default: true,
		},
		showTimelineReplies: {
			where: "deviceAccount" as const,
			default: false,
		},
		addRe: {
			where: "account" as const,
			default: true,
		},
		detectPostLanguage: {
			where: "deviceAccount" as const,
			default: true,
		},
		openServerInfo: {
			where: "device" as const,
			default: true,
		},
		iconSet: {
			where: "device" as const,
			default: "ph-bold" as "ph-bold" | "ph-duotone" | "ph-light" | "ph" | "ph-fill",
		},
		recentlyUsedPostLanguages: {
			where: "account" as const,
			default: [] as string[],
		},
		useEmojiCdn: {
			where: "device" as const,
			default: true,
		},
		showPreviewByDefault: {
			where: "deviceAccount" as const,
			default: false,
		},
		hideFollowButtons: {
			where: "device" as const,
			default: true,
		},
		replaceChatButtonWithAccountButton: {
			where: "device" as const,
			default: true,
		},
		replaceWidgetsButtonWithReloadButton: {
			where: "device" as const,
			default: false,
		},
		searchURL: {
			where: "device" as const,
			default: "https://duckduckgo.com/?q=",
		},
		showBigPostButton: {
			where: "device" as const,
			default: false,
		},
		enableTimelineStreaming: {
			where: "deviceAccount" as const,
			default: true,
		},
		enablePullToRefresh: {
			where: "deviceAccount" as const,
			default: true,
		},
		pullToRefreshThreshold: {
			where: "device" as const,
			default: 150,
		},
		showNoAltTextWarning: {
			where: "account" as const,
			default: true,
		},
		showAddFileDescriptionAtFirstPost: {
			where: "account" as const,
			default: false,
		},
		addAlt4MeTag: {
			where: "account" as const,
			default: false,
		},
		autocorrectNoteLanguage: {
			where: "account" as const,
			default: false,
		},
		foldNotification: {
			where: "deviceAccount" as const,
			default: true,
		},
		mergeThreadInTimeline: {
			where: "deviceAccount" as const,
			default: true,
		},
		mergeRenotesInTimeline: {
			where: "deviceAccount" as const,
			default: true,
		},
		writingMode: {
			where: "deviceAccount" as const,
			default: "horizontal-tb" as
				| "horizontal-tb"
				| "vertical-rl"
				| "vertical-lr"
				| "vertical-rl-upright"
				| "vertical-lr-upright",
		},
	}),
);
