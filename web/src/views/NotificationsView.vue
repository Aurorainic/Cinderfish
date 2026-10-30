<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Notification } from "../api/types";
import { session } from "../stores/session";
import { setUnreadNotifications, streaming, unreadNotifications } from "../stores/streaming";
import AppIcon from "../components/AppIcon.vue";
import NotificationItem from "../components/NotificationItem.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams } from "../components/ui/PagedFeed.vue";
import { usePageTitle } from "../composables/use-page-title";

// Type tabs are pushed down to the endpoint as includeTypes, so every page
// the feed receives is already of the selected type and an empty result means
// the type really has no notifications.
type Filter = "all" | "reaction" | "mention" | "renote";

const FILTERS: readonly Filter[] = ["all", "reaction", "mention", "renote"];
const LABELS: Record<Filter, string> = {
  all: "全部",
  reaction: "反应",
  mention: "提及",
  renote: "转贴",
};

const PAGE_SIZE = 20;

const route = useRoute();
const router = useRouter();

usePageTitle("通知");

const tabs = FILTERS.map((key) => ({ key, label: LABELS[key] }));

const notifications = ref<Notification[]>([]);
const marking = ref(false);
const error = ref("");

const tab = computed<string>({
  get() {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    return (FILTERS as readonly string[]).includes(raw) ? raw : "all";
  },
  set(value) {
    if (route.query.tab === value) return;
    void router.push({ query: { ...route.query, tab: value } });
  },
});

function isFilter(value: string): value is Filter {
  return (FILTERS as readonly string[]).includes(value);
}

const filter = computed<Filter>(() => (isFilter(tab.value) ? tab.value : "all"));

watch(
  () => route.query.tab,
  () => {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    if (!raw || raw === tab.value) return;
    void router.replace({ query: { ...route.query, tab: tab.value } });
  },
  { immediate: true },
);

function includeTypesFor(type: Filter): string[] | undefined {
  return type === "all" ? undefined : [type];
}

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

// PagedFeed is keyed on the tab, so a switch remounts it and starts from an
// empty, freshly fetched page for the new type instead of paginating on top
// of the previous type's pages.
async function fetcher(p: PagedFeedParams): Promise<Notification[]> {
  return api.notifications({
    limit: PAGE_SIZE,
    untilId: p.untilId || undefined,
    includeTypes: includeTypesFor(filter.value),
    markAsRead: false,
  });
}

function matchesFilter(item: unknown): boolean {
  const types = includeTypesFor(filter.value);
  if (!types) return true;
  const type = (item as Notification | null)?.type;
  return typeof type === "string" && types.includes(type);
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
  <PageHeader v-model:tab="tab" title="通知" :tabs="session.me ? tabs : null">
    <template v-if="session.me" #actions>
      <span v-if="unreadNotifications" class="count" :title="`${unreadNotifications} 条未读`">
        {{ unreadNotifications }}
      </span>
      <button
        class="mark"
        type="button"
        :disabled="marking || unreadNotifications === 0"
        :title="marking ? '处理中…' : '全部标为已读'"
        :aria-label="marking ? '处理中…' : '全部标为已读'"
        @click="markAllRead"
      >
        <AppIcon name="check" />
        <span class="mark-label">{{ marking ? "处理中…" : "全部标为已读" }}</span>
      </button>
    </template>
  </PageHeader>

  <div v-if="!session.me" class="card placeholder">
    <AppIcon name="bell" />
    <h2>需要登录</h2>
    <p class="small">登录后可以查看回复、反应、关注与提及。</p>
    <RouterLink to="/login" class="button small">登录</RouterLink>
  </div>

  <div v-else class="page-body">
    <p v-if="error" class="notice error">{{ error }}</p>

    <div class="feed">
      <PagedFeed
        :key="filter"
        :fetcher="fetcher"
        stream-channel="main"
        stream-type="notification"
        new-items-label="新通知"
        :stream-filter="matchesFilter"
        @items="onItems"
      >
        <template #item="{ item }">
          <NotificationItem :notification="item as Notification" />
        </template>
        <template #empty>{{ emptyText }}</template>
      </PagedFeed>
    </div>
  </div>
</template>

<style scoped>
.page-body {
  padding-top: var(--space-12);
}

.count {
  padding: 2px 9px;
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-xs);
  font-weight: 700;
}

.mark {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-weak);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: 600;
  cursor: pointer;
  transition: color var(--dur-base) ease, background var(--dur-base) ease;
}

.mark .icon {
  width: 15px;
  height: 15px;
}

.mark:hover:not(:disabled) {
  color: var(--text);
  background: var(--surface-hover);
}

.mark:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.mark:disabled {
  opacity: 0.5;
  cursor: default;
}

/* Borderless rows carry their own hairline, so they must not be spaced. */
.feed :deep(.paged-feed) {
  gap: 0;
}

@media (max-width: 520px) {
  .mark-label {
    display: none;
  }

  .mark {
    width: 34px;
    padding: 0;
    justify-content: center;
  }
}
</style>
