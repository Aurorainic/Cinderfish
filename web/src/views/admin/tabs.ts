export type AdminTabKey = "overview" | "users" | "files" | "settings";

export interface AdminTabDef {
  key: AdminTabKey;
  label: string;
  icon: "globe" | "user" | "image" | "settings";
  group: "实例" | "配置";
}

export const ADMIN_TABS: readonly AdminTabDef[] = [
  { key: "overview", label: "概览", icon: "globe", group: "实例" },
  { key: "users", label: "用户", icon: "user", group: "实例" },
  { key: "files", label: "文件", icon: "image", group: "实例" },
  { key: "settings", label: "设置", icon: "settings", group: "配置" },
];

export const DEFAULT_ADMIN_TAB: AdminTabKey = "overview";

export function resolveAdminTab(value: unknown): AdminTabKey {
  const raw = Array.isArray(value) ? value[0] : value;
  const match = ADMIN_TABS.find((item) => item.key === raw);
  return match ? match.key : DEFAULT_ADMIN_TAB;
}
