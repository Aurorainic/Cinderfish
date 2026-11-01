// Menu item union for the popup and context menus.
//
// Ported from firefish@02c31e0a2804de48c9d73028df525e197d8df622:packages/client/src/types/menu.ts
// (AGPL-3.0). The union describes the menu UI, not the backend contract, so it
// is kept verbatim except that Firefish's `entities.User` is Cinderfish's
// `UserLite`. There is deliberately no `radio` item: the pinned Firefish types
// have none (`type: "radio"` only exists in its form DSL, which is dropped).
import type { Ref } from "vue";
import type { UserLite } from "../api/types";

export type MenuAction = (ev: MouseEvent) => void;

export type MenuDivider = null;
export type MenuNull = undefined;
export interface MenuLabel {
  type: "label";
  text: string;
  textStyle?: string;
}
export interface MenuLink {
  type: "link";
  to: string;
  text: string;
  textStyle?: string;
  icon?: string;
  indicate?: boolean;
  avatar?: UserLite;
}
export interface MenuA {
  type: "a";
  href: string;
  target?: string;
  download?: string;
  text: string;
  textStyle?: string;
  icon?: string;
  indicate?: boolean;
}
export interface MenuUser {
  type: "user";
  user: UserLite;
  active?: boolean;
  indicate?: boolean;
  hidden?: boolean;
  action: MenuAction;
}
export interface MenuSwitch {
  type: "switch";
  ref: Ref<boolean>;
  text: string;
  textStyle?: string;
  disabled?: boolean;
}
export interface MenuButton {
  type?: "button";
  text: string;
  textStyle?: string;
  icon?: string;
  indicate?: boolean;
  danger?: boolean;
  accent?: boolean;
  active?: boolean;
  hidden?: boolean;
  avatar?: UserLite;
  action: MenuAction;
}
export interface MenuParent {
  type: "parent";
  text: string;
  textStyle?: string;
  icon?: string;
  children: OuterMenuItem[];
}

export interface MenuPending {
  type: "pending";
}

type OuterMenuItem =
  | MenuDivider
  | MenuNull
  | MenuLabel
  | MenuLink
  | MenuA
  | MenuUser
  | MenuSwitch
  | MenuButton
  | MenuParent;
type OuterPromiseMenuItem = Promise<
  MenuLabel | MenuLink | MenuA | MenuUser | MenuSwitch | MenuButton | MenuParent
>;
export type MenuItem = OuterMenuItem | OuterPromiseMenuItem;
export type InnerMenuItem =
  | MenuDivider
  | MenuPending
  | MenuLabel
  | MenuLink
  | MenuA
  | MenuUser
  | MenuSwitch
  | MenuButton
  | MenuParent;
