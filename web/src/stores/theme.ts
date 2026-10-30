import { ref } from "vue";

export type Theme = "dark" | "light";
export type ThemePreference = Theme | "auto";

export const THEME_STORAGE_KEY = "cinderfish.theme";

const DARK_QUERY = "(prefers-color-scheme: dark)";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";
const SWITCH_CLASS = "theme-switching";
const SWITCH_MS = 350;

export function isTheme(value: unknown): value is Theme {
  return value === "dark" || value === "light";
}

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === "auto" || isTheme(value);
}

function readDataset(): Theme | null {
  const current = document.documentElement.dataset.theme;
  return isTheme(current) ? current : null;
}

export function systemTheme(): Theme {
  return window.matchMedia(DARK_QUERY).matches ? "dark" : "light";
}

export function storedThemePreference(): ThemePreference {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY);
    return isThemePreference(value) ? value : "auto";
  } catch {
    return "auto";
  }
}

export function resolveTheme(preference: ThemePreference = storedThemePreference()): Theme {
  return preference === "auto" ? systemTheme() : preference;
}

export const themePreference = ref<ThemePreference>(storedThemePreference());

export const theme = ref<Theme>(readDataset() ?? resolveTheme(themePreference.value));

let switchTimer = 0;

function syncThemeColor(): void {
  const meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) return;
  const value = getComputedStyle(document.documentElement).getPropertyValue("--theme-color").trim();
  if (value) meta.setAttribute("content", value);
}

function markSwitch(): void {
  if (window.matchMedia(REDUCED_MOTION_QUERY).matches) return;
  const root = document.documentElement;
  root.classList.add(SWITCH_CLASS);
  window.clearTimeout(switchTimer);
  switchTimer = window.setTimeout(() => root.classList.remove(SWITCH_CLASS), SWITCH_MS);
}

function persist(preference: ThemePreference): void {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, preference);
  } catch {
    // Storage may be unavailable; the choice simply does not persist.
  }
}

function paint(next: Theme): void {
  const changed = document.documentElement.dataset.theme !== next;
  theme.value = next;
  document.documentElement.dataset.theme = next;
  syncThemeColor();
  if (changed) markSwitch();
}

export function applyTheme(next: Theme, options: { persist?: boolean } = {}): void {
  if (options.persist !== false) {
    themePreference.value = next;
    persist(next);
  }
  paint(next);
}

export function setThemePreference(preference: ThemePreference): void {
  themePreference.value = preference;
  persist(preference);
  paint(resolveTheme(preference));
}

export function setTheme(next: Theme): void {
  setThemePreference(next);
}

export function toggleTheme(): Theme {
  const next: Theme = resolveTheme(themePreference.value) === "dark" ? "light" : "dark";
  setThemePreference(next);
  return next;
}

window.addEventListener("storage", (event) => {
  if (event.key !== THEME_STORAGE_KEY) return;
  const preference = isThemePreference(event.newValue) ? event.newValue : "auto";
  themePreference.value = preference;
  paint(resolveTheme(preference));
});

function syncWithSystem(): void {
  if (themePreference.value !== "auto") return;
  const next = systemTheme();
  if (next !== theme.value) paint(next);
}

window.matchMedia(DARK_QUERY).addEventListener("change", syncWithSystem);

// A background tab can miss the media change event, and some platforms only
// report the new value once the window is focused again.
window.addEventListener("focus", syncWithSystem);
document.addEventListener("visibilitychange", () => {
  if (!document.hidden) syncWithSystem();
});
