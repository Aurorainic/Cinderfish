<script setup lang="ts">
import { computed, ref } from "vue";
import { api } from "../api/client";
import type { Notification } from "../api/types";
import { session } from "../stores/session";
import { setUnreadNotifications } from "../stores/streaming";
import AppIcon from "../components/AppIcon.vue";
import NotificationItem from "../components/NotificationItem.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams } from "../components/ui/PagedFeed.vue";

type Filter = "all" | "unread";

const PAGE_SIZE = 20;

const notifications = ref<Notification[]>([]);
const unreadCount = ref(0);
const marking = ref(false);
const error = ref("");
const filter = ref<Filter>("all");

const filters: { id: Filter; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "unread", label: "未读" },
];

const itemFilter = computed(() =>
  filter.value === "unread" ? (item: unknown) => (item as Notification).isRead === false : undefined,
);
const emptyText = computed(() => (filter.value === "unread" ? "没有未读通知。" : "还没有通知。"));

async function fetcher(p: PagedFeedParams): Promise<Notification[]> {
  return api.notifications({ limit: PAGE_SIZE, untilId: p.untilId || undefined });
}

function onItems(items: unknown[]) {
  notifications.value = items as Notification[];
  const unread = notifications.value.filter((n) => !n.isRead).length;
  unreadCount.value = unread;
  setUnreadNotifications(unread);
}

async function markAllRead() {
  if (marking.value || unreadCount.value === 0) return;
  marking.value = true;
  error.value = "";
  try {
    await api.markAllNotificationsRead();
    for (const n of notifications.value) n.isRead = true;
    onItems(notifications.value);
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
  <div v-if="!session.me" class="card placeholder">
    <AppIcon name="bell" />
    <h2>需要登录</h2>
    <p class="small">登录后可以查看回复、反应、关注与提及。</p>
    <RouterLink to="/login" class="button small">登录</RouterLink>
  </div>

  <template v-else>
    <header class="page-title">
      <h1>通知</h1>
      <span v-if="unreadCount" class="count">{{ unreadCount }} 条未读</span>
      <button
        class="button ghost small mark"
        type="button"
        :disabled="marking || unreadCount === 0"
        @click="markAllRead"
      >
        <AppIcon name="check" />
        {{ marking ? "处理中…" : "全部标为已读" }}
      </button>
    </header>

    <nav class="filters">
      <button
        v-for="tab in filters"
        :key="tab.id"
        type="button"
        class="filter"
        :class="{ active: filter === tab.id }"
        @click="filter = tab.id"
      >
        {{ tab.label }}
        <span v-if="tab.id === 'unread' && unreadCount" class="badge">{{ unreadCount }}</span>
      </button>
    </nav>

    <p v-if="error" class="notice error">{{ error }}</p>

    <PagedFeed
      :fetcher="fetcher"
      stream-channel="main"
      stream-type="notification"
      new-items-label="新通知"
      :filter="itemFilter"
      @items="onItems"
    >
      <template #item="{ item }">
        <NotificationItem :notification="item as Notification" class="row" />
      </template>
      <template #empty>{{ emptyText }}</template>
    </PagedFeed>
  </template>
</template>

<style scoped>
.count {
  padding: 2px 9px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 0.78rem;
  font-weight: 700;
}

.mark {
  margin-left: auto;
}

.mark .icon {
  width: 14px;
  height: 14px;
}

.filters {
  display: flex;
  gap: 6px;
  padding: 5px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel);
  width: fit-content;
}

.filter {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 18px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.filter:hover {
  color: var(--text);
}

.filter.active {
  background: var(--accent);
  color: var(--on-accent);
}

.badge {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--panel-2);
  color: var(--accent);
  font-size: 0.72rem;
  font-weight: 700;
}

.filter.active .badge {
  background: rgba(0, 0, 0, 0.22);
  color: var(--on-accent);
}
</style>
