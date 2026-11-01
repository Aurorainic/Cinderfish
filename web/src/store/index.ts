import { reactive, shallowReactive, toRef } from "vue";

const STORAGE_KEY = "cinderfish.firefish.store";
const defaults = {
  animation: true,
  defaultNoteLocalOnly: false,
  defaultNoteVisibility: "public",
  enablePullToRefresh: true,
  enableTimelineStreaming: true,
  foldNotification: true,
  keepCw: true,
  localOnly: false,
  mergeRenotesInTimeline: true,
  mergeThreadInTimeline: true,
  recentlyUsedPostLanguages: [] as string[],
  rememberNoteVisibility: false,
  seperateRenoteQuote: true,
  showBigPostButton: false,
  showGapBetweenNotesInTimeline: false,
  showNoAltTextWarning: true,
  showPreviewByDefault: false,
  showTimelineReplies: false,
  tlGlobalHintClosed: false,
  tlHomeHintClosed: false,
  tlLocalHintClosed: false,
  tlRecommendedHintClosed: false,
  tlSocialHintClosed: false,
  uploadFolder: null as string | null,
  visibility: "public",
};

export type FirefishStoreState = typeof defaults;

export interface FirefishStore {
  readonly state: FirefishStoreState;
  readonly reactiveState: { foldNotification: ReturnType<typeof toRef<FirefishStoreState["foldNotification"]>> };
  set<K extends keyof FirefishStoreState>(key: K, value: FirefishStoreState[K]): void;
}

const memoryStorage = new Map<string, string>();

function defaultStorage(): Pick<Storage, "getItem" | "setItem"> {
  if (typeof localStorage !== "undefined") return localStorage;
  return {
    getItem: (key) => memoryStorage.get(key) ?? null,
    setItem: (key, value) => memoryStorage.set(key, value),
  };
}

export function createFirefishStore(
  storage: Pick<Storage, "getItem" | "setItem"> = defaultStorage(),
): FirefishStore {
  let saved: Partial<FirefishStoreState> = {};
  try {
    const parsed: unknown = JSON.parse(storage.getItem(STORAGE_KEY) ?? "{}");
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      saved = parsed as Partial<FirefishStoreState>;
    }
  } catch {
    saved = {};
  }

  const state = reactive({ ...defaults, ...saved }) as FirefishStoreState;
  const reactiveState = shallowReactive({ foldNotification: toRef(state, "foldNotification") });

  return {
    state,
    reactiveState,
    set(key, value) {
      state[key] = value;
      try {
        storage.setItem(STORAGE_KEY, JSON.stringify(state));
      } catch {
        // Storage may be unavailable; keep the value for this session.
      }
    },
  };
}

export const defaultStore = createFirefishStore();
