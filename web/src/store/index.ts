import { reactive, shallowReactive, toRefs, type Ref } from "vue";

const STORAGE_KEY = "cinderfish.firefish.store";
const defaults = {
  detectPostLanguage: true,
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
export type FirefishStoreReactiveState = { [K in keyof FirefishStoreState]: Ref<FirefishStoreState[K]> };

export interface FirefishStore {
  readonly state: FirefishStoreState;
  readonly reactiveState: FirefishStoreReactiveState;
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
  const reactiveState = shallowReactive(toRefs(state)) as FirefishStoreReactiveState;

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

export const postFormActions: { title: string; handler: (...args: unknown[]) => unknown }[] = [];
export const userActions: { title: string; handler: (...args: unknown[]) => unknown }[] = [];
export const noteActions: { title: string; handler: (...args: unknown[]) => unknown }[] = [];
export const noteViewInterruptors: { handler: (...args: unknown[]) => Promise<unknown> }[] = [];
export const notePostInterruptors: { handler: (...args: unknown[]) => Promise<unknown> }[] = [];
