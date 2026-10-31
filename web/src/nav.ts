import { isAdmin, isLoggedIn } from "./stores/session";
import { unreadNotifications } from "./stores/streaming";

// The single navigation registry. The rail, the mobile bar, the drawer and the
// settings shortcuts all render from NAV_ITEM_DEFS, so a route only ever needs
// adding here once. Icons must be keys of the hand-written set in
// components/AppIcon.vue. `requiresAuth` mirrors the route's `meta.requiresAuth`
// so the registry and the router guard agree, and `mobile` marks whether the
// item belongs on the small-screen bar by default.
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

// Unread indicators are declared by key instead of per-item badge closures, so
// every surface reads the same store value: adding a badge means adding one
// source here and pointing a def at it.
export type UnreadKey = "notifications";

const UNREAD_SOURCES: Record<UnreadKey, () => number> = {
  notifications: () => unreadNotifications.value,
};

export function unreadCountFor(key: UnreadKey): number {
  return UNREAD_SOURCES[key]();
}

export interface NavItemDef {
  id: NavKey;
  label: string;
  icon: NavIcon;
  to: string;
  requiresAuth: boolean;
  mobile: boolean;
  unreadKey?: UnreadKey;
  show?: () => boolean;
}

export const NAV_ITEM_DEFS: Record<NavKey, NavItemDef> = {
  timeline: {
    id: "timeline",
    label: "时间线",
    icon: "home",
    to: "/",
    requiresAuth: false,
    mobile: true,
  },
  notifications: {
    id: "notifications",
    label: "通知",
    icon: "bell",
    to: "/notifications",
    requiresAuth: false,
    mobile: true,
    unreadKey: "notifications",
  },
  explore: {
    id: "explore",
    label: "探索",
    icon: "compass",
    to: "/explore",
    requiresAuth: false,
    mobile: true,
  },
  settings: {
    id: "settings",
    label: "设置",
    icon: "settings",
    to: "/settings",
    requiresAuth: false,
    mobile: false,
  },
  admin: {
    id: "admin",
    label: "管理",
    icon: "shield",
    to: "/admin",
    requiresAuth: true,
    mobile: false,
    show: () => isAdmin.value,
  },
};

export const NAV_ITEM_KEYS: readonly NavKey[] = Object.keys(NAV_ITEM_DEFS) as NavKey[];

// The drawer carries settings and administration in its own footer row, so the
// main list skips them.
export const DRAWER_FOOTER_KEYS: readonly NavKey[] = ["settings", "admin"];

export const DEFAULT_NAV: readonly NavKey[] = [...NAV_ITEM_KEYS];

// Derived from the `mobile` flags: an item declares once where it lives.
export const DEFAULT_MOBILE_NAV: readonly NavKey[] = NAV_ITEM_KEYS.filter(
  (key) => NAV_ITEM_DEFS[key].mobile,
);

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
  if (def.requiresAuth && !isLoggedIn.value) return false;
  return def.show ? def.show() : true;
}

// Reads isLoggedIn/show()/badge(), so call it from a computed or render scope
// to track them.
export function navItemsFor(keys: readonly string[]): NavEntry[] {
  const entries: NavEntry[] = [];
  for (const key of keys) {
    if (!isNavKey(key) || !isNavItemVisible(key)) continue;
    const def = NAV_ITEM_DEFS[key];
    entries.push({
      key,
      def,
      badge: def.unreadKey ? unreadCountFor(def.unreadKey) : 0,
    });
  }
  return entries;
}
