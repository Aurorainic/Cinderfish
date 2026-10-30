import { defineAsyncComponent, type Component } from "vue";

// Widget registry: the single source of the widgets the right column can show.
// Every entry carries the async component, so feeds stay out of the initial
// bundle, and `defaultProps`, which the persisted props are merged onto — a
// hand-edited storage record can never leave a widget without its props.
export type WidgetName = "instance" | "links" | "notifications" | "timeline" | "memo";

// Names from AppIcon's hand-written set; typed here so a typo fails the build
// instead of rendering an empty <path>.
export type WidgetIcon = "flame" | "globe" | "feather" | "bell" | "home";

export interface WidgetDef {
  name: WidgetName;
  label: string;
  icon: WidgetIcon;
  description: string;
  component: Component;
  defaultProps: Record<string, unknown>;
}

// Every widget type the add picker offers, in picker order.
export const WIDGET_NAMES: readonly WidgetName[] = [
  "instance",
  "links",
  "notifications",
  "timeline",
  "memo",
];

// The stack a fresh client starts with, and the one "restore defaults" returns.
export const DEFAULT_WIDGET_NAMES: readonly WidgetName[] = ["instance", "links", "notifications"];

export const WIDGETS: Record<WidgetName, WidgetDef> = {
  instance: {
    name: "instance",
    label: "实例",
    icon: "flame",
    description: "实例名称、简介与快捷入口",
    component: defineAsyncComponent(() => import("../components/widgets/InstanceWidget.vue")),
    defaultProps: {},
  },
  links: {
    name: "links",
    label: "关于",
    icon: "globe",
    description: "探索入口与协议链接",
    component: defineAsyncComponent(() => import("../components/widgets/LinksWidget.vue")),
    defaultProps: {},
  },
  notifications: {
    name: "notifications",
    label: "通知",
    icon: "bell",
    description: "回复、反应与关注",
    component: defineAsyncComponent(() => import("../components/widgets/NotificationsWidget.vue")),
    defaultProps: { height: 320 },
  },
  timeline: {
    name: "timeline",
    label: "时间线",
    icon: "home",
    description: "首页、本地、社交与全球",
    component: defineAsyncComponent(() => import("../components/widgets/TimelineWidget.vue")),
    defaultProps: { source: "local", height: 360 },
  },
  memo: {
    name: "memo",
    label: "便签",
    icon: "feather",
    description: "只保存在这台设备上的记事板",
    component: defineAsyncComponent(() => import("../components/widgets/MemoWidget.vue")),
    defaultProps: {},
  },
};
