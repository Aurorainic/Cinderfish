<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "../../api/client";
import type { Notification } from "../../api/types";
import { isLoggedIn, session } from "../../stores/session";
import { setUnreadNotifications, unreadNotifications } from "../../stores/streaming";
import AppIcon from "../AppIcon.vue";
import NotificationItem from "../NotificationItem.vue";
import MkButton from "../ui/MkButton.vue";
import MkInfo from "../ui/MkInfo.vue";
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
      <MkButton link mini to="/login">登录</MkButton>
    </div>

    <template v-else>
      <MkInfo v-if="error" class="inline-notice" warn>{{ error }}</MkInfo>
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
          <template #empty>
            <div class="_fullinfo empty-placeholder">
              <p>还没有通知。</p>
            </div>
          </template>
        </PagedFeed>
      </div>
    </template>
  </WidgetCard>
</template>

<style scoped>
.action {
  display: grid;
  place-items: center;
  inline-size: 26px;
  block-size: 26px;
  padding: 0;
  border: solid 0.5px var(--divider);
  border-radius: 50%;
  background: none;
  color: var(--accent);
  cursor: pointer;
  transition: color 0.1s ease, border-color 0.1s ease, opacity 0.1s ease;
}

a.action {
  text-decoration: none;
}

.action:hover:not(:disabled) {
  border-color: var(--accent);
}

.action:disabled {
  color: var(--fgTransparentWeak);
  opacity: 0.6;
  cursor: default;
}

.action:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 2px;
}

.action .icon {
  inline-size: 13px;
  block-size: 13px;
}

/* Rows carry their own hairline, so PagedFeed must not space them. */
.scroll {
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--divider) transparent;
}

.scroll :deep(.paged-feed) {
  gap: 0;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding-block: 22px;
  padding-inline: 14px;
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
}

.inline-notice {
  margin-block-start: 10px;
  margin-inline: 12px;
}

.empty-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--fgTransparentWeak);
}

.empty-placeholder > p {
  margin: 0;
  font-size: 0.95em;
  line-height: 1.5;
}
</style>
