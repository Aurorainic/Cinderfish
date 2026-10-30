import { ref, watch } from "vue";
import {
  DEFAULT_MOBILE_NAV,
  DEFAULT_NAV,
  MAX_MOBILE_NAV,
  NAV_ITEM_KEYS,
  isNavKey,
  type NavKey,
} from "../nav";

// Which registry items appear in the rail (desktop) and in the bottom bar
// (mobile). Both lists are stored inside the shared `cinderfish.ui` record, so
// every write must merge with what is already there — otherwise the shell
// store's `shellMode` and this store's `nav` erase each other.
export interface NavPrefs {
  desktop: NavKey[];
  mobile: NavKey[];
}

const STORAGE_KEY = "cinderfish.ui";

type UiRecord = Record<string, unknown>;

function readPrefs(): UiRecord {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as UiRecord;
    }
  } catch {
    // Unreadable storage (private mode, corrupted value) falls back to {}.
  }
  return {};
}

function sanitizeList(value: unknown, limit: number): NavKey[] {
  if (!Array.isArray(value)) return [];
  const seen = new Set<string>();
  const keys: NavKey[] = [];
  for (const entry of value) {
    if (keys.length >= limit) break;
    if (!isNavKey(entry) || seen.has(entry)) continue;
    seen.add(entry);
    keys.push(entry);
  }
  return keys;
}

// Unknown keys and duplicates are dropped, the mobile list is capped at
// MAX_MOBILE_NAV, and an empty or unusable list falls back to the default.
export function sanitizeNavPrefs(value: unknown): NavPrefs {
  const record =
    value && typeof value === "object" && !Array.isArray(value) ? (value as UiRecord) : {};
  const desktop = sanitizeList(record.desktop, NAV_ITEM_KEYS.length);
  const mobile = sanitizeList(record.mobile, MAX_MOBILE_NAV);
  return {
    desktop: desktop.length ? desktop : [...DEFAULT_NAV],
    mobile: mobile.length ? mobile : [...DEFAULT_MOBILE_NAV],
  };
}

function readNavPrefs(): NavPrefs {
  return sanitizeNavPrefs(readPrefs().nav);
}

export const navPrefs = ref<NavPrefs>(readNavPrefs());

// Persist through the same merge the shell store uses, so `shellMode` written
// by another tab or by the rail survives.
watch(
  navPrefs,
  (prefs) => {
    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ ...readPrefs(), nav: sanitizeNavPrefs(prefs) }),
      );
    } catch {
      // Storage may be unavailable; the preference simply does not persist.
    }
  },
  { deep: true },
);

export function setDesktopNav(keys: readonly string[]): void {
  navPrefs.value = sanitizeNavPrefs({ desktop: keys, mobile: navPrefs.value.mobile });
}

export function setMobileNav(keys: readonly string[]): void {
  navPrefs.value = sanitizeNavPrefs({ desktop: navPrefs.value.desktop, mobile: keys });
}

export function toggleDesktopItem(key: NavKey): void {
  const current = navPrefs.value.desktop;
  const next = current.includes(key) ? current.filter((entry) => entry !== key) : [...current, key];
  if (!next.length) return;
  setDesktopNav(next);
}

export function toggleMobileItem(key: NavKey): void {
  const current = navPrefs.value.mobile;
  if (current.includes(key)) {
    if (current.length <= 1) return;
    setMobileNav(current.filter((entry) => entry !== key));
    return;
  }
  if (current.length >= MAX_MOBILE_NAV) return;
  setMobileNav([...current, key]);
}

export function resetNav(): void {
  navPrefs.value = { desktop: [...DEFAULT_NAV], mobile: [...DEFAULT_MOBILE_NAV] };
}
