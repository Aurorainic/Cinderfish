<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "../../api/client";
import type { Notification } from "../../api/types";
import { isLoggedIn, session } from "../../stores/session";
import { setUnreadNotifications, unreadNotifications } from "../../stores/streaming";
import AppIcon from "../AppIcon.vue";
import NotificationItem from "../NotificationItem.vue";
import PagedFeed from "../ui/PagedFeed.vue";
import WidgetCard from "./WidgetCard.vue";

const props = defineProps<{ widgetProps?: Record<string, unknown> }>();

const PAGE_SIZE = 20;
const DEFAULT_HEIGHT = 320;
const MIN_HEIGHT = 200;
const MAX_HEIGHT = 640;

// Height is a widget prop, so it survives a reload with the rest of the stack;
// the clamp keeps a hand-edited record from producing an unusable column.
const height = computed(() => {
  const value = props.widgetProps?.height;
  if (typeof value !== "number" || !Number.isFinite(value)) return DEFAULT_HEIGHT;
  return Math.min(Math.max(Math.round(value), MIN_HEIGHT), MAX_HEIGHT);
});

const hint = computed(() => (unreadNotifications.value ? `${unreadNotifications.value} 条未读` : ""));

// The shared counter is only seeded by the notifications page today, so a
// reader who lands on another page would see an empty hint and a permanently
// disabled "mark all read". Seeding never lowers a count the stream raised.
watch(
  () => session.me?.unreadNotificationsCount,
  (count) => {
    if (typeof count !== "number") return;
    if (count > unreadNotifications.value) setUnreadNotifications(count);
  },
  { immediate: true },
);

function fetcher(p: { untilId?: string }): Promise<Notification[]> {
  return api.notifications({ limit: PAGE_SIZE, untilId: p.untilId, markAsRead: false });
}

const marking = ref(false);
const error = ref("");

async function markAllRead(): Promise<void> {
  if (marking.value || unreadNotifications.value === 0) return;
  marking.value = true;
  error.value = "";
  try {
    await api.markAllNotificationsRead();
    setUnreadNotifications(0);
    if (session.me) {
      session.me.unreadNotificationsCount = 0;
      session.me.hasUnreadNotification = false;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "标记已读失败";
  } finally {
    marking.value = false;
  }
}
</script>

<template>
  <WidgetCard title="通知" icon="bell" :hint="hint" flush>
    <template #actions>
      <RouterLink class="action" to="/notifications" title="打开通知页" aria-label="打开通知页">
        <AppIcon name="bell" />
      </RouterLink>
      <button
        class="action"
        type="button"
        title="全部标为已读"
        aria-label="全部标为已读"
        :disabled="marking || unreadNotifications === 0"
        @click="markAllRead"
      >
        <AppIcon name="check" />
      </button>
    </template>

    <div v-if="!isLoggedIn" class="empty">
      <span>登录后查看通知。</span>
      <RouterLink to="/login" class="button small">登录</RouterLink>
    </div>

    <template v-else>
      <p v-if="error" class="notice error inline-notice">{{ error }}</p>
      <div class="scroll" :style="{ height: `${height}px` }">
        <PagedFeed
          :fetcher="fetcher"
          stream-channel="main"
          stream-type="notification"
          new-items-label="新通知"
        >
          <template #item="{ item }">
            <NotificationItem :notification="(item as Notification)" />
          </template>
          <template #empty>还没有通知。</template>
        </PagedFeed>
      </div>
    </template>
  </WidgetCard>
</template>

<style scoped>
.action {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: transparent;
  color: var(--accent);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

a.action {
  text-decoration: none;
}

.action:hover:not(:disabled) {
  border-color: var(--accent);
}

.action:disabled {
  color: var(--muted);
  opacity: 0.6;
  cursor: default;
}

.action:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.action .icon {
  width: 13px;
  height: 13px;
}

/* Rows carry their own hairline, so PagedFeed must not space them. */
.scroll {
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}

.scroll :deep(.paged-feed) {
  gap: 0;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 22px 14px;
  color: var(--muted);
  font-size: 0.85rem;
}

.inline-notice {
  margin: 10px 12px 0;
}
</style>
