// The `base` store, translated from Firefish's src/store.ts `defaultStore`.
// Bucket name, key names and defaults are kept identical so future page ports
// can read them without a translation layer.
import { api } from "../api";
import { darkTheme, lightTheme } from "../themes";
import { Storage, type PizzaxOptions, type PizzaxRemote, type StateDef } from "./pizzax";

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

export type TimelineSource = "home" | "local" | "social" | "global";

export interface StatusBarItem {
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

export interface TimelineMemory {
  src: TimelineSource;
  arg: string | null;
}

export type NoteVisibilityPreference = "public" | "home" | "followers" | "specified";

const baseDef = {
  tutorial: { where: "account", default: 0 },
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
  reactions: {
    where: "account",
    default: ["⭐", "❤️", "😆", "🤔", "😮", "🎉", "💢", "😥", "😇", "🥴", "🍮"] as string[],
  },
  mutedWords: { where: "account", default: [] as string[] },
  mutedAds: { where: "account", default: [] as string[] },
  showAds: { where: "account", default: true },
  menu: { where: "deviceAccount", default: [...menuOptions] as MenuOption[] },
  visibility: {
    where: "deviceAccount",
    default: "public" as NoteVisibilityPreference,
  },
  localOnly: { where: "deviceAccount", default: false },
  statusbars: { where: "deviceAccount", default: [] as StatusBarItem[] },
  widgets: { where: "deviceAccount", default: [] as WidgetItem[] },
  tl: {
    where: "deviceAccount",
    default: { src: "home", arg: null } as TimelineMemory,
  },
  deck: { where: "deviceAccount", default: null as string | null },
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

export const store = new Storage("base", baseDef, storeOptions);

/** Firefish name, kept for ports that still say `defaultStore`. */
export const defaultStore = store;

export type Store = typeof store;

// Re-export themes for callers that only touch the store.
export { lightTheme, darkTheme };
