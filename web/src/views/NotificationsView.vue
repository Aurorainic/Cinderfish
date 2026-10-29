<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api } from "../api/client";
import type { Notification } from "../api/types";
import { session } from "../stores/session";
import { setUnreadNotifications, streaming } from "../stores/streaming";
import AppIcon from "../components/AppIcon.vue";
import NotificationItem from "../components/NotificationItem.vue";

type Filter = "all" | "unread";

const pageSize = 20;

const items = ref<Notification[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const marking = ref(false);
const error = ref("");
const untilId = ref("");
const hasMore = ref(false);
const filter = ref<Filter>("all");

const filters: { id: Filter; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "unread", label: "未读" },
];

const unreadCount = computed(() => items.value.filter((n) => !n.isRead).length);
const visible = computed(() =>
  filter.value === "unread" ? items.value.filter((n) => !n.isRead) : items.value,
);
const emptyText = computed(() =>
  filter.value === "unread" ? "没有未读通知。" : "还没有通知。",
);

async function load() {
  if (!session.me) return;
  loading.value = true;
  error.value = "";
  try {
    const page = await api.notifications({ limit: pageSize });
    items.value = page;
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
    setUnreadNotifications(unreadCount.value);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载通知失败";
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (!untilId.value || loadingMore.value || !session.me) return;
  loadingMore.value = true;
  error.value = "";
  try {
    const page = await api.notifications({ limit: pageSize, untilId: untilId.value });
    const seen = new Set(items.value.map((n) => n.id));
    items.value = [...items.value, ...page.filter((n) => !seen.has(n.id))];
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载更多通知失败";
  } finally {
    loadingMore.value = false;
  }
}

async function markAllRead() {
  if (marking.value || unreadCount.value === 0) return;
  marking.value = true;
  error.value = "";
  try {
    await api.markAllNotificationsRead();
    for (const n of items.value) n.isRead = true;
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

let off: (() => void) | null = null;

onMounted(() => {
  load();
  off = streaming.on((channel, type, body) => {
    if (channel !== "main" || type !== "notification" || !session.me) return;
    const incoming = body as Notification;
    if (!incoming?.id || items.value.some((n) => n.id === incoming.id)) return;
    items.value = [incoming, ...items.value];
    setUnreadNotifications(unreadCount.value);
  });
});

onBeforeUnmount(() => off?.());
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

    <div v-if="loading" class="panel skeleton-panel">
      <div v-for="i in 5" :key="i" class="skeleton"></div>
    </div>

    <div v-else-if="visible.length" class="panel">
      <NotificationItem v-for="n in visible" :key="n.id" :notification="n" class="row" />
    </div>

    <p v-else-if="!error" class="placeholder">
      <AppIcon name="bell" />
      <span>{{ emptyText }}</span>
    </p>

    <div v-if="hasMore && !loading" class="more">
      <button class="button ghost small" type="button" :disabled="loadingMore" @click="loadMore">
        {{ loadingMore ? "加载中…" : "加载更多" }}
      </button>
    </div>
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

.panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
}

.row + .row {
  border-top: 1px solid var(--border);
}

.more {
  display: flex;
  justify-content: center;
  padding: 14px 0 0;
}

.skeleton-panel {
  gap: 0;
  padding: 0;
}

.skeleton {
  height: 66px;
  background: linear-gradient(90deg, var(--panel) 25%, var(--panel-2) 50%, var(--panel) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite linear;
}

.skeleton + .skeleton {
  border-top: 1px solid var(--border);
}

@keyframes shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
