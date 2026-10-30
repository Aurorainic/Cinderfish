<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { Notification } from "../../api/types";
import {
  COLUMN_META,
  COLUMN_TYPES,
  COLUMN_WIDTHS,
  moveColumn,
  removeColumn,
  setColumnType,
  toggleColumnWidth,
  type ColumnConfig,
  type ColumnType,
} from "../../stores/deck";
import { isLoggedIn } from "../../stores/session";
import AppIcon from "../AppIcon.vue";
import NotificationItem from "../NotificationItem.vue";
import TimelineFeed from "../TimelineFeed.vue";
import PagedFeed from "../ui/PagedFeed.vue";
import WidgetStack from "../widgets/WidgetStack.vue";
import { interceptDeckClick } from "./deck-links";

type TimelineSource = "home" | "local" | "hybrid" | "global";

const props = defineProps<{ column: ColumnConfig; index: number; count: number }>();

const TIMELINE_TYPES: readonly ColumnType[] = ["home", "local", "hybrid", "global"];
const NOTIFICATION_PAGE_SIZE = 20;

const meta = computed(() => COLUMN_META[props.column.type]);
const isTimeline = computed(() => TIMELINE_TYPES.includes(props.column.type));
const timelineSource = computed(() => props.column.type as TimelineSource);
const widthAction = computed(() =>
  props.column.width === COLUMN_WIDTHS[1]
    ? `收窄列 · ${COLUMN_WIDTHS[0]}px`
    : `加宽列 · ${COLUMN_WIDTHS[1]}px`,
);

// The explore column is a timeline column whose source the reader picks.
const exploreScope = ref<"local" | "global">("local");

function notificationFetcher(p: { untilId?: string }): Promise<Notification[]> {
  return api.notifications({ limit: NOTIFICATION_PAGE_SIZE, untilId: p.untilId, markAsRead: false });
}

const menuOpen = ref(false);
const root = ref<HTMLElement | null>(null);
const body = ref<HTMLElement | null>(null);

function toggleMenu(): void {
  menuOpen.value = !menuOpen.value;
}

function chooseType(type: ColumnType): void {
  setColumnType(props.column.id, type);
  menuOpen.value = false;
}

function shift(delta: number): void {
  moveColumn(props.column.id, props.index + delta);
  menuOpen.value = false;
}

function toggleWidth(): void {
  toggleColumnWidth(props.column.id);
  menuOpen.value = false;
}

function goTop(): void {
  menuOpen.value = false;
  body.value?.scrollTo({
    top: 0,
    behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
  });
}

function closeColumn(): void {
  removeColumn(props.column.id);
}

// A popover that is open should not survive a click anywhere else.
function onDocumentPointerDown(event: PointerEvent): void {
  if (!menuOpen.value) return;
  const el = root.value;
  if (el && !el.contains(event.target as Node)) menuOpen.value = false;
}

onMounted(() => document.addEventListener("pointerdown", onDocumentPointerDown));
onBeforeUnmount(() => document.removeEventListener("pointerdown", onDocumentPointerDown));
</script>

<template>
  <section
    ref="root"
    class="deck-column"
    :style="{ '--column-width': `${column.width}px` }"
    :aria-label="`${meta.label}列`"
    @keydown.escape="menuOpen = false"
  >
    <header class="col-head">
      <span class="col-icon" :title="meta.description"><AppIcon :name="meta.icon" /></span>
      <h2 class="col-title">{{ meta.label }}</h2>
      <button
        class="col-menu-button"
        type="button"
        title="列菜单"
        aria-haspopup="menu"
        :aria-expanded="menuOpen"
        @click="toggleMenu"
      >
        <AppIcon name="more" />
      </button>

      <div v-if="menuOpen" class="col-menu" role="menu">
        <button type="button" role="menuitem" class="menu-item" @click="goTop">回到顶部</button>
        <div class="menu-sep" role="separator"></div>
        <p class="menu-label">数据源</p>
        <button
          v-for="type in COLUMN_TYPES"
          :key="type"
          type="button"
          role="menuitemradio"
          class="menu-item"
          :class="{ active: column.type === type }"
          :aria-checked="column.type === type"
          @click="chooseType(type)"
        >
          <AppIcon :name="COLUMN_META[type].icon" />
          <span>{{ COLUMN_META[type].label }}</span>
          <AppIcon v-if="column.type === type" name="check" class="menu-check" />
        </button>
        <div class="menu-sep" role="separator"></div>
        <button type="button" role="menuitem" class="menu-item" :disabled="index === 0" @click="shift(-1)">
          上移
        </button>
        <button type="button" role="menuitem" class="menu-item" :disabled="index >= count - 1" @click="shift(1)">
          下移
        </button>
        <button type="button" role="menuitem" class="menu-item" @click="toggleWidth">
          {{ widthAction }}
        </button>
        <button type="button" role="menuitem" class="menu-item danger" @click="closeColumn">关闭列</button>
      </div>
    </header>

    <div
      ref="body"
      class="col-body"
      :class="{ 'col-body--widgets': column.type === 'widgets' }"
      @click.capture="interceptDeckClick"
    >
      <TimelineFeed v-if="isTimeline" :source="timelineSource" :composer="false" />

      <template v-else-if="column.type === 'notifications'">
        <PagedFeed
          v-if="isLoggedIn"
          :fetcher="notificationFetcher"
          stream-channel="main"
          stream-type="notification"
          new-items-label="新通知"
        >
          <template #item="{ item }">
            <NotificationItem :notification="(item as Notification)" class="row" />
          </template>
          <template #empty>还没有通知。</template>
        </PagedFeed>
        <div v-else class="placeholder">
          <AppIcon name="bell" />
          <span>登录后可以查看通知。</span>
          <RouterLink to="/login" class="button small">登录</RouterLink>
        </div>
      </template>

      <WidgetStack v-else-if="column.type === 'widgets'" :heading="false" />

      <template v-else>
        <nav class="scope" aria-label="探索范围">
          <button
            type="button"
            class="scope-button"
            :class="{ active: exploreScope === 'local' }"
            @click="exploreScope = 'local'"
          >
            本地
          </button>
          <button
            type="button"
            class="scope-button"
            :class="{ active: exploreScope === 'global' }"
            @click="exploreScope = 'global'"
          >
            全球
          </button>
        </nav>
        <TimelineFeed :key="exploreScope" :source="exploreScope" :composer="false" />
      </template>
    </div>
  </section>
</template>

<style scoped>
/* Fixed-width column that snap-scrolls as one page on narrow viewports; the
   strip in DeckShell owns the scrolling, the column owns its size. The width
   comes from the persisted column config, so a strip can mix both sizes. */
.deck-column {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 0 0 var(--column-width, 350px);
  min-width: 0;
  height: 100%;
  background: var(--bg-raised);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  scroll-snap-align: start;
}

.col-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px 10px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--panel-2);
  flex: none;
}

.col-icon {
  display: inline-flex;
  color: var(--accent);
}

.col-title {
  flex: 1;
  min-width: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-menu-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.col-menu-button:hover,
.col-menu-button[aria-expanded="true"] {
  background: var(--panel-2);
  color: var(--text);
}

.col-menu-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.col-menu {
  position: absolute;
  top: 46px;
  right: 8px;
  z-index: 20;
  width: 190px;
  padding: 6px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.menu-label {
  margin: 4px 8px 6px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: 0.88rem;
  text-align: left;
  cursor: pointer;
}

.menu-item:hover:not(:disabled),
.menu-item:focus-visible {
  background: var(--panel-2);
}

.menu-item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.menu-item:disabled {
  opacity: 0.4;
  cursor: default;
}

.menu-item.active {
  color: var(--accent);
  font-weight: 600;
}

.menu-item.danger {
  color: var(--danger);
}

.menu-check {
  margin-left: auto;
  width: 1em;
  height: 1em;
}

.menu-sep {
  height: 1px;
  margin: 6px 8px;
  background: var(--border);
}

/* The feed scrolls inside the column, so the column keeps its header in view
   and the page itself never grows taller than the deck strip. */
.col-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
}

/* Feeds own their own row padding; the widget stack does not. */
.col-body--widgets {
  padding: var(--space-12);
}

.scope {
  display: flex;
  gap: 4px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--divider);
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--bg-raised);
}

.scope-button {
  padding: 5px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
}

.scope-button.active {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--on-accent);
}

.scope-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

/* Below 1100px one column owns the strip: the reader flips between columns
   with a horizontal swipe instead of reading two cramped ones. */
@media (max-width: 1099px) {
  .deck-column {
    flex-basis: 100%;
  }
}
</style>
