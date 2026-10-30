import { ref } from "vue";

export type Theme = "dark" | "light";

export const THEME_STORAGE_KEY = "cinderfish.theme";

const DARK_QUERY = "(prefers-color-scheme: dark)";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";
const SWITCH_CLASS = "theme-switching";
const SWITCH_MS = 350;

export function isTheme(value: unknown): value is Theme {
  return value === "dark" || value === "light";
}

function readDataset(): Theme | null {
  const current = document.documentElement.dataset.theme;
  return isTheme(current) ? current : null;
}

export function systemTheme(): Theme {
  return window.matchMedia(DARK_QUERY).matches ? "dark" : "light";
}

export function storedTheme(): Theme | null {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY);
    return isTheme(value) ? value : null;
  } catch {
    return null;
  }
}

export function resolveTheme(): Theme {
  return storedTheme() ?? systemTheme();
}

export const theme = ref<Theme>(readDataset() ?? resolveTheme());

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

function persist(next: Theme): void {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, next);
  } catch {
    // Storage may be unavailable; the choice simply does not persist.
  }
}

export function applyTheme(next: Theme, options: { persist?: boolean } = {}): void {
  const changed = document.documentElement.dataset.theme !== next;
  theme.value = next;
  document.documentElement.dataset.theme = next;
  if (options.persist !== false) persist(next);
  syncThemeColor();
  if (changed) markSwitch();
}

export function setTheme(next: Theme): void {
  applyTheme(next);
}

export function toggleTheme(): Theme {
  applyTheme(theme.value === "dark" ? "light" : "dark");
  return theme.value;
}

window.addEventListener("storage", (event) => {
  if (event.key !== THEME_STORAGE_KEY) return;
  applyTheme(isTheme(event.newValue) ? event.newValue : systemTheme(), { persist: false });
});

window.matchMedia(DARK_QUERY).addEventListener("change", () => {
  if (storedTheme()) return;
  applyTheme(systemTheme(), { persist: false });
});
