import { isAdmin } from "./stores/session";
import { unreadNotifications } from "./stores/streaming";

// The single navigation registry. The rail, the mobile bar and the drawer all
// render from NAV_ITEM_DEFS, so a route only ever needs adding here once.
// Icons must be keys of the hand-written set in components/AppIcon.vue.
export type NavIcon =
  | "home"
  | "bell"
  | "compass"
  | "settings"
  | "shield"
  | "user"
  | "globe"
  | "search"
  | "feather"
  | "more"
  | "plus"
  | "logout"
  | "sun"
  | "moon"
  | "close"
  | "flame";

export interface NavItemDef {
  label: string;
  icon: NavIcon;
  to: string;
  show?: () => boolean;
  badge?: () => number;
}

export type NavKey = "timeline" | "notifications" | "explore" | "settings" | "admin";

export const NAV_ITEM_DEFS: Record<NavKey, NavItemDef> = {
  timeline: { label: "时间线", icon: "home", to: "/" },
  notifications: {
    label: "通知",
    icon: "bell",
    to: "/notifications",
    badge: () => unreadNotifications.value,
  },
  explore: { label: "探索", icon: "compass", to: "/explore" },
  settings: { label: "设置", icon: "settings", to: "/settings" },
  admin: { label: "管理", icon: "shield", to: "/admin", show: () => isAdmin.value },
};

export const NAV_ITEM_KEYS: readonly NavKey[] = [
  "timeline",
  "notifications",
  "explore",
  "settings",
  "admin",
];

// The drawer carries settings and administration in its own footer row, so the
// main list skips them.
export const DRAWER_FOOTER_KEYS: readonly NavKey[] = ["settings", "admin"];

export const DEFAULT_NAV: readonly NavKey[] = [...NAV_ITEM_KEYS];

export const DEFAULT_MOBILE_NAV: readonly NavKey[] = ["timeline", "notifications", "explore"];

export const MAX_MOBILE_NAV = 4;

export interface NavEntry {
  key: NavKey;
  def: NavItemDef;
  badge: number;
}

export function isNavKey(value: unknown): value is NavKey {
  return typeof value === "string" && (NAV_ITEM_KEYS as readonly string[]).includes(value);
}

export function isNavItemVisible(key: NavKey): boolean {
  const def = NAV_ITEM_DEFS[key];
  return def.show ? def.show() : true;
}

// Reads show()/badge(), so call it from a computed or render scope to track them.
export function navItemsFor(keys: readonly string[]): NavEntry[] {
  const entries: NavEntry[] = [];
  for (const key of keys) {
    if (!isNavKey(key) || !isNavItemVisible(key)) continue;
    const def = NAV_ITEM_DEFS[key];
    entries.push({ key, def, badge: def.badge ? def.badge() : 0 });
  }
  return entries;
}
