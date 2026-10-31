import type { RouteLocationRaw } from "vue-router";
import type AppIcon from "../AppIcon.vue";

// The shape both the settings and the admin column render: grouped entries with
// an icon, a label and a target. It is the Cinderfish counterpart of the fork's
// MkSuperMenu groups (`fork/components/MkSuperMenu.vue`), which the fork feeds
// from `menuDef` in pages/settings/index.vue and pages/admin/index.vue — one
// menu component for both surfaces, so the two columns cannot drift apart.
export type SectionMenuIcon = InstanceType<typeof AppIcon>["$props"]["name"];

export interface SectionMenuItem {
  key: string;
  label: string;
  icon: SectionMenuIcon;
  to: RouteLocationRaw;
  // A short right-aligned remark, e.g. "需登录" for a gated entry.
  note?: string;
}

export interface SectionMenuGroup {
  title?: string;
  items: SectionMenuItem[];
}
