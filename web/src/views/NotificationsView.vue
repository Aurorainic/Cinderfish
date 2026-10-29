<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api } from "../api/client";
import type { Notification } from "../api/types";
import { session } from "../stores/session";
import { setUnreadNotifications, streaming, unreadNotifications } from "../stores/streaming";
import AppIcon from "../components/AppIcon.vue";
import NotificationItem from "../components/NotificationItem.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams } from "../components/ui/PagedFeed.vue";

// Type tabs are a pure client-side view filter: the backend endpoint takes no
// type parameter we need, and PagedFeed keeps paginating on the unfiltered
// page while the filter only decides what is painted.
type Filter = "all" | "reaction" | "mention" | "renote";

const PAGE_SIZE = 20;

const notifications = ref<Notification[]>([]);
const marking = ref(false);
const error = ref("");
const filter = ref<Filter>("all");

const filters: { id: Filter; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "reaction", label: "反应" },
  { id: "mention", label: "提及" },
  { id: "renote", label: "转贴" },
];

const itemFilter = computed(() => {
  const type = filter.value;
  return type === "all" ? undefined : (item: unknown) => (item as Notification).type === type;
});

const emptyText = computed(() => {
  switch (filter.value) {
    case "reaction":
      return "还没有反应通知。";
    case "mention":
      return "还没有提及通知。";
    case "renote":
      return "还没有转贴通知。";
    default:
      return "还没有通知。";
  }
});

async function fetcher(p: PagedFeedParams): Promise<Notification[]> {
  return api.notifications({ limit: PAGE_SIZE, untilId: p.untilId || undefined });
}

// The unread badge lives in the store, shared with the shell navigation. Seed
// it from the session once the account is loaded; live notifications bump it
// in the store itself.
let seeded = false;
watch(
  () => session.me?.unreadNotificationsCount,
  (count) => {
    if (seeded || typeof count !== "number") return;
    seeded = true;
    setUnreadNotifications(count);
  },
  { immediate: true },
);

function onItems(items: unknown[]) {
  notifications.value = items as Notification[];
}

function markLocalRead() {
  for (const n of notifications.value) n.isRead = true;
  setUnreadNotifications(0);
  if (session.me) {
    session.me.unreadNotificationsCount = 0;
    session.me.hasUnreadNotification = false;
  }
}

async function markAllRead() {
  if (marking.value || unreadNotifications.value === 0) return;
  marking.value = true;
  error.value = "";
  try {
    await api.markAllNotificationsRead();
    markLocalRead();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "标记已读失败";
  } finally {
    marking.value = false;
  }
}

// "Mark all as read" is announced on the main channel, so a tab opened in a
// second client clears its dots without polling.
let subscription = "";
let offStream: (() => void) | null = null;

onMounted(() => {
  subscription = streaming.subscribe("main");
  offStream = streaming.on((channel, type) => {
    if (channel === "main" && type === "readAllNotifications") markLocalRead();
  });
});

onBeforeUnmount(() => {
  if (subscription) streaming.unsubscribe(subscription);
  subscription = "";
  offStream?.();
  offStream = null;
});
</script>

<template>
  <div v-if="!session.me" class="card placeholder">
    <AppIcon name="bell" />
    <h2>需要登录</h2>
    <p class="small">登录后可以查看回复、反应、关注与提及。</p>
    <RouterLink to="/login" class="button small">登录</RouterLink>
  </div>

  <template v-else>
    <header class="page-title head">
      <h1>通知</h1>
      <span v-if="unreadNotifications" class="count">{{ unreadNotifications }} 条未读</span>
      <button
        class="button ghost small mark"
        type="button"
        :disabled="marking || unreadNotifications === 0"
        @click="markAllRead"
      >
        <AppIcon name="check" />
        {{ marking ? "处理中…" : "全部标为已读" }}
      </button>
    </header>

    <nav class="filters" aria-label="通知类型">
      <button
        v-for="tab in filters"
        :key="tab.id"
        type="button"
        class="filter"
        :class="{ active: filter === tab.id }"
        :aria-pressed="filter === tab.id"
        @click="filter = tab.id"
      >
        {{ tab.label }}
      </button>
    </nav>

    <p v-if="error" class="notice error">{{ error }}</p>

    <div class="feed">
      <PagedFeed
        :fetcher="fetcher"
        stream-channel="main"
        stream-type="notification"
        new-items-label="新通知"
        :filter="itemFilter"
        @items="onItems"
      >
        <template #item="{ item }">
          <NotificationItem :notification="item as Notification" />
        </template>
        <template #empty>{{ emptyText }}</template>
      </PagedFeed>
    </div>
  </template>
</template>

<style scoped>
.head {
  flex-wrap: wrap;
}

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
  gap: 4px;
  padding: 4px;
  margin-bottom: 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel);
  width: fit-content;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}

.filters::-webkit-scrollbar {
  display: none;
}

.filter {
  flex: none;
  padding: 6px 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  font-size: 0.88rem;
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

/* Borderless rows carry their own hairline, so they must not be spaced. */
.feed :deep(.paged-feed) {
  gap: 0;
}

@media (max-width: 520px) {
  .mark {
    margin-left: 0;
  }

  .filter {
    padding: 6px 13px;
  }
}
</style>
