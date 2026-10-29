<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { streaming } from "../../stores/streaming";

export interface PagedFeedParams {
  untilId?: string;
  sinceId?: string;
}

const props = withDefaults(
  defineProps<{
    fetcher: (p: PagedFeedParams) => Promise<unknown[]>;
    streamChannel?: string;
    streamType?: string;
    pageSize?: number;
    disabled?: boolean;
    newItemsLabel?: string;
    filter?: (item: unknown) => boolean;
  }>(),
  {
    streamChannel: "",
    streamType: "note",
    pageSize: 20,
    disabled: false,
    newItemsLabel: "新笔记",
    filter: undefined,
  },
);

const emit = defineEmits<{ items: [unknown[]] }>();

const items = ref<unknown[]>([]);
const pending = ref<unknown[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const error = ref("");
const hasMore = ref(false);

const root = ref<HTMLElement | null>(null);
const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function idOf(item: unknown): string {
  const id = (item as { id?: unknown } | null)?.id;
  return typeof id === "string" ? id : "";
}

const seen = new Set<string>();

function takeFresh(list: unknown[]): unknown[] {
  const fresh = list.filter((item) => {
    const id = idOf(item);
    return !id || !seen.has(id);
  });
  for (const item of fresh) {
    const id = idOf(item);
    if (id) seen.add(id);
  }
  return fresh;
}

function publish() {
  emit("items", items.value.slice());
}

async function load() {
  if (props.disabled) return;
  loading.value = true;
  error.value = "";
  try {
    const page = await props.fetcher({});
    seen.clear();
    items.value = takeFresh(page);
    hasMore.value = page.length >= props.pageSize;
    pending.value = [];
    publish();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
  void checkMore();
}

async function loadMore() {
  if (props.disabled || loading.value || loadingMore.value || !hasMore.value) return;
  const last = items.value[items.value.length - 1];
  const untilId = idOf(last);
  if (!untilId) return;
  loadingMore.value = true;
  try {
    const page = await props.fetcher({ untilId });
    const fresh = takeFresh(page);
    if (fresh.length) items.value = [...items.value, ...fresh];
    // Stop when the server has no more, and also when a page produced nothing
    // new: without that guard a fully-duplicated page would re-request the
    // same cursor forever.
    hasMore.value = page.length >= props.pageSize && fresh.length > 0;
    publish();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载更多失败";
  } finally {
    loadingMore.value = false;
  }
  void checkMore();
}

function refresh() {
  return load();
}

function remove(id: string) {
  const next = items.value.filter((item) => idOf(item) !== id);
  if (next.length === items.value.length) return;
  seen.delete(id);
  items.value = next;
  publish();
}

async function flushPending() {
  if (!pending.value.length) return;
  const fresh = takeFresh(pending.value);
  pending.value = [];
  if (fresh.length) items.value = [...fresh, ...items.value];
  publish();
  await nextTick();
  scrollToTop();
}

function scrollToTop() {
  let el: HTMLElement | null = root.value;
  while (el) {
    const style = getComputedStyle(el);
    if (/(auto|scroll|overlay)/.test(style.overflowY) && el.scrollHeight > el.clientHeight) {
      el.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }
    el = el.parentElement;
  }
  window.scrollTo({ top: 0, behavior: "smooth" });
}

// Date separators: a centered small row is inserted whenever the UTC date of
// two adjacent items differs. Items without createdAt never split the list.
type Row = { kind: "date"; key: string; label: string } | { kind: "item"; key: string; item: unknown };

function dayOf(item: unknown): string {
  const createdAt = (item as { createdAt?: unknown } | null)?.createdAt;
  return typeof createdAt === "string" && /^\d{4}-\d{2}-\d{2}/.test(createdAt) ? createdAt.slice(0, 10) : "";
}

function dateLabel(day: string): string {
  const year = Number(day.slice(0, 4));
  const month = Number(day.slice(5, 7));
  const date = Number(day.slice(8, 10));
  return `${year}年${month}月${date}日`;
}

const rows = computed<Row[]>(() => {
  const list = props.filter ? items.value.filter(props.filter) : items.value;
  const out: Row[] = [];
  let prevDay = "";
  list.forEach((item, index) => {
    const day = dayOf(item);
    if (day && prevDay && day !== prevDay) {
      out.push({ kind: "date", key: `d-${day}-${index}`, label: dateLabel(day) });
    }
    if (day) prevDay = day;
    out.push({ kind: "item", key: idOf(item) || `row-${index}`, item });
  });
  return out;
});

const pendingCount = computed(() => pending.value.length);

function checkMore() {
  const el = sentinel.value;
  if (!el) return;
  const rect = el.getBoundingClientRect();
  if (rect.top <= window.innerHeight + 240) void loadMore();
}

function handleStream(channel: string, type: string, body: unknown) {
  if (channel === "" && type === "__reconnected") {
    if (!props.disabled) void load();
    return;
  }
  if (channel !== props.streamChannel || type !== props.streamType) return;
  if (!body || typeof body !== "object") return;
  const id = idOf(body);
  if (!id || seen.has(id)) return;
  if (pending.value.some((item) => idOf(item) === id)) return;
  pending.value = [body, ...pending.value];
}

let subscription = "";
let offStream: (() => void) | null = null;

function syncSubscription() {
  if (subscription) {
    streaming.unsubscribe(subscription);
    subscription = "";
  }
  if (!props.streamChannel || props.disabled) return;
  subscription = streaming.subscribe(props.streamChannel);
}

watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) {
      items.value = [];
      pending.value = [];
      seen.clear();
      hasMore.value = false;
      error.value = "";
    } else {
      void load();
    }
    syncSubscription();
  },
);

watch(
  () => props.fetcher,
  () => {
    void load();
  },
);

watch(hasMore, async (more) => {
  await nextTick();
  if (more && sentinel.value) observer?.observe(sentinel.value);
});

onMounted(() => {
  offStream = streaming.on(handleStream);
  syncSubscription();
  if (typeof IntersectionObserver !== "undefined") {
    observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) void loadMore();
        }
      },
      { rootMargin: "240px 0px" },
    );
  }
  if (!props.disabled) void load();
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
  if (subscription) streaming.unsubscribe(subscription);
  subscription = "";
  offStream?.();
  offStream = null;
});

defineExpose({ refresh, remove });
</script>

<template>
  <div ref="root" class="paged-feed">
    <p v-if="error" class="notice error">{{ error }}</p>

    <div v-if="pendingCount" class="pill-row">
      <button type="button" class="new-pill" @click="flushPending">
        {{ pendingCount }} 条{{ newItemsLabel }}
      </button>
    </div>

    <div v-if="loading && !items.length" class="skeleton-list">
      <div v-for="i in 3" :key="i" class="skeleton"></div>
    </div>

    <template v-else>
      <template v-for="row in rows" :key="row.key">
        <p v-if="row.kind === 'date'" class="date-sep">{{ row.label }}</p>
        <slot v-else name="item" :item="row.item" />
      </template>

      <div v-if="!rows.length && !error" class="placeholder">
        <slot name="empty">这里还没有内容。</slot>
      </div>
    </template>

    <div v-if="hasMore" ref="sentinel" class="sentinel">
      <span v-if="loadingMore" class="muted small">加载中…</span>
    </div>
  </div>
</template>

<style scoped>
.paged-feed {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pill-row {
  display: flex;
  justify-content: center;
}

.new-pill {
  padding: 6px 18px;
  border: 1px solid var(--accent);
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.new-pill:hover,
.new-pill:focus-visible {
  background: var(--accent);
  color: var(--on-accent);
}

.date-sep {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 4px 0;
  color: var(--muted);
  font-size: 0.78rem;
  white-space: nowrap;
}

.date-sep::before,
.date-sep::after {
  content: "";
  flex: 1;
  height: 1px;
  background: var(--border);
}

.empty {
  color: var(--muted);
}

.sentinel {
  display: flex;
  justify-content: center;
  padding: 8px 0;
  min-height: 16px;
}

.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.skeleton {
  height: 120px;
  border-radius: var(--radius-lg);
  background: linear-gradient(90deg, var(--panel) 25%, var(--panel-2) 50%, var(--panel) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite linear;
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
