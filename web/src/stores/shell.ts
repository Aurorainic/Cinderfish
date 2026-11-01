import { ref, watch } from "vue";

// The shell mode is a client-side display preference, not a route: "universal"
// is the normal three-zone layout, "deck" the column container (task F5).
// Both live under one `cinderfish.ui` record so later UI preferences can be
// added without a second storage key.
export type ShellMode = "universal" | "deck";

const STORAGE_KEY = "cinderfish.ui";
const DEFAULT_MODE: ShellMode = "universal";

type UiPrefs = Record<string, unknown> & { shellMode?: unknown };

function isShellMode(value: unknown): value is ShellMode {
  return value === "universal" || value === "deck";
}

function readPrefs(): UiPrefs {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as UiPrefs;
    }
  } catch {
    // Unreadable storage (private mode, corrupted value) falls back to default.
  }
  return {};
}

function readShellMode(): ShellMode {
  const { shellMode } = readPrefs();
  return isShellMode(shellMode) ? shellMode : DEFAULT_MODE;
}

export const shellMode = ref<ShellMode>(readShellMode());

// Watching the ref (instead of persisting inside setShellMode) keeps any
// direct assignment persisted too, so the deck switch in F5 cannot drift.
watch(shellMode, (mode) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ ...readPrefs(), shellMode: mode }));
  } catch {
    // Storage may be unavailable; the preference simply does not persist.
  }
});

export function setShellMode(mode: ShellMode): void {
  shellMode.value = mode;
}

// The preference and the layout are separate values. Deck columns can only
// render the three surfaces they own (timeline, notifications, explore); every
// other route keeps the scrolling universal layout even while the preference is
// "deck". Folding the two together is what clipped /notes/:id, /@handle and
// /settings/* behind `height: 100dvh; overflow: hidden` before.
export function shellLayout(preferredDeck: boolean, routeOwnsDeck: boolean): ShellMode {
  return preferredDeck && routeOwnsDeck ? "deck" : "universal";
}
