import type AppIcon from "../../components/AppIcon.vue";

// The icon names come from AppIcon's hand-written set. NavIcon is narrower
// than that set, and the settings menu needs a couple of icons it does not
// carry, so the tab type reads the component's own prop type instead.
type SettingsIcon = InstanceType<typeof AppIcon>["$props"]["name"];

// The settings menu is the single list both the layout and the menu render
// from: the tab key is the route segment under /settings, so a tab added here
// only needs its child route in router/index.ts.
export type SettingsTabKey =
  | "profile"
  | "appearance"
  | "navigation"
  | "widgets"
  | "notifications"
  | "privacy"
  | "account"
  | "about";

export type SettingsGroup = "账户" | "客户端" | "其他";

export interface SettingsTab {
  key: SettingsTabKey;
  label: string;
  icon: SettingsIcon;
  group: SettingsGroup;
  requiresAuth?: boolean;
}

export const SETTINGS_GROUPS: readonly SettingsGroup[] = ["账户", "客户端", "其他"];

export const DEFAULT_SETTINGS_TAB: SettingsTabKey = "profile";

export const SETTINGS_TABS: readonly SettingsTab[] = [
  { key: "profile", label: "个人资料", icon: "user", group: "账户", requiresAuth: true },
  { key: "appearance", label: "外观", icon: "sun", group: "客户端" },
  { key: "navigation", label: "导航", icon: "compass", group: "客户端" },
  { key: "widgets", label: "小组件", icon: "image", group: "客户端" },
  { key: "notifications", label: "通知", icon: "bell", group: "客户端", requiresAuth: true },
  { key: "privacy", label: "隐私", icon: "shield", group: "其他", requiresAuth: true },
  { key: "account", label: "账户", icon: "settings", group: "其他", requiresAuth: true },
  { key: "about", label: "关于", icon: "globe", group: "其他" },
];

export function findSettingsTab(key: string): SettingsTab {
  return (
    SETTINGS_TABS.find((tab) => tab.key === key) ??
    SETTINGS_TABS.find((tab) => tab.key === DEFAULT_SETTINGS_TAB) ??
    SETTINGS_TABS[0]
  );
}
