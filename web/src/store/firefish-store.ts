// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/store.ts
//
// Slices the ported UI needs, with Firefish-compatible default values. Dropped
// relative to upstream: the cold store (cloud backup), the plugin store, and
// slices tied to capabilities Cinderfish lacks (ads, channels, chat, statusbars,
// deck, reaction picker tuning beyond the basics). Where the Firefish default
// depended on signed-in state at module load (tl.src), the signed-in default
// is used because the store is constructed after session restore.
import { markRaw } from "vue";
import { Storage } from "./pizzax";

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

export type ThemeMode = "auto" | "light" | "dark";

export const defaultStore = markRaw(
  new Storage("base", {
    // --- menu / navigation -------------------------------------------------
    menu: {
      where: "deviceAccount",
      default: menuOptions,
    },
    menuDisplay: {
      where: "device",
      default: "sideFull" as "sideFull" | "sideIcon" | "top",
    },
    sidebarCollapsed: {
      // Cinderfish-specific slice shaping the ported deck navigation.
      where: "device",
      default: false,
    },

    // --- widgets ------------------------------------------------------------
    widgets: {
      where: "deviceAccount",
      default: [] as {
        name: string;
        id: string;
        place: string | null;
        data: Record<string, unknown>;
      }[],
    },

    // --- post form ----------------------------------------------------------
    keepCw: {
      where: "account",
      default: true,
    },
    rememberNoteVisibility: {
      where: "account",
      default: false,
    },
    defaultNoteVisibility: {
      where: "account",
      default: "public" as "public" | "home" | "followers" | "specified",
    },
    defaultNoteLocalOnly: {
      where: "account",
      default: false,
    },
    visibility: {
      where: "deviceAccount",
      default: "public" as "public" | "home" | "followers" | "specified",
    },
    localOnly: {
      where: "deviceAccount",
      default: false,
    },
    postFormWithHashtags: {
      where: "device",
      default: false,
    },
    postFormHashtags: {
      where: "device",
      default: "",
    },
    showFixedPostForm: {
      where: "device",
      default: false,
    },
    showBigPostButton: {
      where: "device",
      default: false,
    },

    // --- theme ---------------------------------------------------------------
    themeMode: {
      // Cinderfish slice replacing Firefish's boolean darkMode: the ported
      // settings UI offers auto/light/dark.
      where: "device",
      default: "auto" as ThemeMode,
    },
    darkMode: {
      where: "device",
      default: false,
    },
    themeInitial: {
      where: "device",
      default: true,
    },

    // --- timeline -------------------------------------------------------------
    tl: {
      where: "deviceAccount",
      default: {
        src: "home" as "home" | "local" | "social" | "global",
        arg: null,
      },
    },
    tlHomeHintClosed: { where: "account", default: false },
    tlLocalHintClosed: { where: "account", default: false },
    tlSocialHintClosed: { where: "account", default: false },
    tlGlobalHintClosed: { where: "account", default: false },
    numberOfPageCache: {
      where: "device",
      default: 5,
    },
    enableInfiniteScroll: {
      where: "device",
      default: true,
    },
    enableTimelineStreaming: {
      where: "deviceAccount",
      default: true,
    },
    enablePullToRefresh: {
      where: "deviceAccount",
      default: true,
    },
    showGapBetweenNotesInTimeline: {
      where: "device",
      default: false,
    },
    mergeRenotesInTimeline: {
      where: "deviceAccount",
      default: true,
    },
    showTimelineReplies: {
      where: "deviceAccount",
      default: false,
    },

    // --- misc display ---------------------------------------------------------
    tutorial: { where: "account", default: 0 },
    showFullAcct: { where: "account", default: false },
    squareAvatars: { where: "device", default: true },
    animation: { where: "device", default: true },
    advancedMfm: { where: "device", default: true },
    animatedMfm: { where: "device", default: true },
    loadRawImages: { where: "device", default: false },
    imageNewTab: { where: "device", default: false },
    disableShowingAnimatedImages: { where: "device", default: false },
    useBlurEffectForModal: { where: "device", default: true },
    useBlurEffect: { where: "device", default: true },
    instanceTicker: {
      where: "device",
      default: "always" as "none" | "remote" | "always",
    },
    nsfw: {
      where: "deviceAccount",
      default: "respect" as "respect" | "force" | "ignore",
    },
    expandOnNoteClick: { where: "device", default: true },
    seperateRenoteQuote: { where: "device", default: true },
    reactions: {
      where: "account",
      default: defaultReactions,
    },
    recentlyUsedEmojis: { where: "device", default: [] as string[] },
    recentlyUsedUsers: { where: "device", default: [] as string[] },
    overwriteNoteBody: {
      // Firefish's overridedDeviceKind, renamed to what the ported layout uses.
      where: "device",
      default: null as null | "smartphone" | "tablet" | "desktop",
    },
    serverDisconnectedBehavior: {
      where: "device",
      default: "nothing" as "nothing" | "quiet" | "reload" | "dialog",
    },
  }),
);

export type DefaultStoreState = typeof defaultStore.state;
export type DefaultStoreReactiveState = typeof defaultStore.reactiveState;

export const postFormActions: {
  title: string;
  handler: (...args: unknown[]) => void | Promise<void>;
}[] = [];
export const userActions: {
  title: string;
  handler: (user: unknown) => void | Promise<void>;
}[] = [];
export const noteActions: {
  title: string;
  handler: (note: unknown) => void | Promise<void>;
}[] = [];
export const noteViewInterruptors: {
  handler: (note: unknown) => Promise<unknown>;
}[] = [];
export const notePostInterruptors: {
  handler: (note: unknown) => Promise<unknown>;
}[] = [];
