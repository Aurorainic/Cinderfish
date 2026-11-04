// Light / dark / auto theme controller.
//
// The theme engine (parsing, compiling, applying CSS variables) lives in
// web/src/themes; this file owns only the reactive choice: which mode the user
// picked, whether `auto` currently resolves to dark, and which concrete theme
// id is active. The preference is stored under `cinderfish.theme`
// (light | dark | auto), the key index.html's no-flash bootstrap reads.
import { computed, ref, type ComputedRef, type Ref } from "vue";
import {
  applyTheme,
  defaultThemeIds,
  findTheme,
  getDefaultTheme,
  prefersDarkMode,
  type ColorSchema,
  type Theme,
} from "../themes";

export type ThemeMode = "light" | "dark" | "auto";

const MODE_KEY = "cinderfish.theme";
const THEME_KEY = "cinderfish.themeId";
const SYNC_CHANNEL = "cinderfish:theme";

function readMode(): ThemeMode {
  try {
    const raw = localStorage.getItem(MODE_KEY);
    if (raw === "light" || raw === "dark" || raw === "auto") return raw;
  } catch {
    // ignore
  }
  return "auto";
}

function readThemeId(): string | null {
  try {
    const raw = localStorage.getItem(THEME_KEY);
    return raw != null && raw.length > 0 ? raw : null;
  } catch {
    return null;
  }
}

export const modeRef: Ref<ThemeMode> = ref<ThemeMode>(readMode());

/** Concrete theme id chosen by the user (null = follow the mode default). */
export const selectedThemeId: Ref<string | null> = ref<string | null>(readThemeId());

const systemPrefersDark: Ref<boolean> = ref<boolean>(prefersDarkMode());

export const resolvedMode: ComputedRef<ColorSchema> = computed<ColorSchema>(() =>
  modeRef.value === "auto"
    ? systemPrefersDark.value
      ? "dark"
      : "light"
    : modeRef.value,
);

/** The active theme object. `findTheme("light" | "dark")` maps to the default. */
export const activeTheme: ComputedRef<Theme> = computed<Theme>(() => {
  const id = selectedThemeId.value;
  if (id != null) {
    const found = findTheme(id);
    if (found != null) return found;
  }
  return getDefaultTheme(resolvedMode.value);
});

/** Theme id, e.g. "l-rosepinedawn". */
export const themeName: ComputedRef<string> = computed(() => activeTheme.value.id);

let initialized = false;
let mediaQuery: MediaQueryList | null = null;
let syncChannel: BroadcastChannel | null = null;

function persist(): void {
  try {
    localStorage.setItem(MODE_KEY, modeRef.value);
    if (selectedThemeId.value == null) localStorage.removeItem(THEME_KEY);
    else localStorage.setItem(THEME_KEY, selectedThemeId.value);
  } catch {
    // ignore
  }
}

export function applyCurrentTheme(): void {
  applyTheme(activeTheme.value);
}

export function setMode(next: ThemeMode): void {
  modeRef.value = next;
  persist();
  applyCurrentTheme();
  syncChannel?.postMessage({ mode: next });
}

/**
 * Accepts "light" | "dark" | "auto" or a theme id. Mode names switch mode and
 * clear any custom theme; ids select that palette.
 */
export function setTheme(name: string): void {
  if (name === "light" || name === "dark" || name === "auto") {
    selectedThemeId.value = null;
    setMode(name);
    return;
  }

  if (findTheme(name) != null) {
    selectedThemeId.value = name;
    persist();
    applyCurrentTheme();
    syncChannel?.postMessage({ themeId: name });
  }
}

/** Switching between light and dark keeps the palette family default. */
export function toggleMode(): void {
  setMode(resolvedMode.value === "dark" ? "light" : "dark");
}

export function initTheme(): void {
  if (initialized) return;
  initialized = true;

  if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
    mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = (event: MediaQueryListEvent): void => {
      systemPrefersDark.value = event.matches;
      if (modeRef.value === "auto") applyCurrentTheme();
    };
    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", onChange);
    } else {
      // Safari < 14
      mediaQuery.addListener(onChange);
    }
  }

  if (typeof BroadcastChannel !== "undefined") {
    syncChannel = new BroadcastChannel(SYNC_CHANNEL);
    syncChannel.addEventListener(
      "message",
      (event: MessageEvent<{ mode?: ThemeMode; themeId?: string }>) => {
        const nextMode = event.data?.mode;
        if (nextMode === "light" || nextMode === "dark" || nextMode === "auto") {
          modeRef.value = nextMode;
          applyCurrentTheme();
        }
        const nextTheme = event.data?.themeId;
        if (typeof nextTheme === "string") {
          selectedThemeId.value = nextTheme;
          applyCurrentTheme();
        }
      },
    );
  }

  applyCurrentTheme();
}

export interface UseThemeReturn {
  /** Id of the active theme (e.g. "l-rosepinedawn"). */
  themeName: ComputedRef<string>;
  /** "light" or "dark" after resolving `auto`. */
  resolvedMode: ComputedRef<ColorSchema>;
  /** The user's raw preference, including `auto`. */
  mode: Ref<ThemeMode>;
  theme: ComputedRef<Theme>;
  setTheme: (name: string) => void;
  setMode: (mode: ThemeMode) => void;
  toggleMode: () => void;
}

export function useTheme(): UseThemeReturn {
  initTheme();

  return {
    themeName,
    resolvedMode,
    mode: modeRef,
    theme: activeTheme,
    setTheme,
    setMode,
    toggleMode,
  };
}

export { defaultThemeIds };
