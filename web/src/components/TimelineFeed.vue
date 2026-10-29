<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api } from "../api/client";
import type { Note } from "../api/types";
import { isLoggedIn } from "../stores/session";
import { streaming } from "../stores/streaming";
import AppIcon from "./AppIcon.vue";
import NoteCard from "./NoteCard.vue";
import NoteComposer from "./NoteComposer.vue";

const props = withDefaults(
  defineProps<{ source: "local" | "global" | "hybrid" | "home"; tag?: string }>(),
  { tag: "" },
);

const notes = ref<Note[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const error = ref("");
const untilId = ref("");
const hasMore = ref(false);
const replyTo = ref<Note | null>(null);
const pageSize = 20;

const needsAuth = computed(() => props.source === "hybrid" || props.source === "home");
const visibleNotes = computed(() => {
  if (!props.tag) return notes.value;
  const needle = `#${props.tag}`.toLowerCase();
  return notes.value.filter((n) => (n.text ?? "").toLowerCase().includes(needle));
});

async function fetchPage(before?: string): Promise<Note[]> {
  const params = { limit: pageSize, untilId: before || undefined };
  switch (props.source) {
    case "local":
      return api.localTimeline(params);
    case "global":
      return api.globalTimeline(params);
    case "hybrid":
      return api.hybridTimeline(params);
    default:
      return api.homeTimeline(params);
  }
}

async function load() {
  if (needsAuth.value && !isLoggedIn.value) return;
  loading.value = true;
  error.value = "";
  try {
    const page = await fetchPage();
    notes.value = page;
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (!untilId.value || loadingMore.value) return;
  loadingMore.value = true;
  try {
    const page = await fetchPage(untilId.value);
    const seen = new Set(notes.value.map((n) => n.id));
    notes.value = [...notes.value, ...page.filter((n) => !seen.has(n.id))];
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载更多失败";
  } finally {
    loadingMore.value = false;
  }
}

function onPosted(note: Note) {
  notes.value = [note, ...notes.value];
  replyTo.value = null;
}

function onRenoted(note: Note) {
  if (!notes.value.some((n) => n.id === note.id)) notes.value = [note, ...notes.value];
}

function onDeleted(id: string) {
  notes.value = notes.value.filter((n) => n.id !== id);
}

const channelName = computed(
  () =>
    ({
      local: "localTimeline",
      global: "globalTimeline",
      hybrid: "hybridTimeline",
      home: "homeTimeline",
    })[props.source],
);

let subscription = "";
let off: (() => void) | null = null;

function subscribe() {
  unsubscribe();
  if (needsAuth.value && !isLoggedIn.value) return;
  subscription = streaming.subscribe(channelName.value);
}

function unsubscribe() {
  if (subscription) streaming.unsubscribe(subscription);
  subscription = "";
}

watch(() => [props.source, props.tag], () => {
  load();
  subscribe();
});

onMounted(() => {
  load();
  subscribe();
  off = streaming.on((channel, type, body) => {
    if (channel === "" && type === "__reconnected") {
      load();
      return;
    }
    if (channel !== channelName.value || type !== "note") return;
    const note = body as Note;
    if (!note?.id || notes.value.some((n) => n.id === note.id)) return;
    notes.value = [note, ...notes.value];
  });
});

onBeforeUnmount(() => {
  unsubscribe();
  off?.();
});
</script>

<template>
  <div class="feed">
    <div v-if="needsAuth && !isLoggedIn" class="card placeholder">
      <AppIcon name="globe" />
      <h2>需要登录</h2>
      <p class="small">这个频道只对已登录用户开放。</p>
      <RouterLink to="/login" class="button small">登录</RouterLink>
    </div>

    <template v-else>
      <NoteComposer v-if="isLoggedIn" :reply-to="replyTo" @posted="onPosted" @cancel="replyTo = null" />

      <div class="feed-head">
        <span class="muted small">{{ loading ? "加载中…" : `${visibleNotes.length} 条笔记` }}</span>
        <button class="ghost-link small" type="button" :disabled="loading" @click="load">刷新</button>
      </div>

      <p v-if="error" class="notice error">{{ error }}</p>

      <div v-if="loading && !notes.length" class="skeleton-list">
        <div v-for="i in 3" :key="i" class="skeleton"></div>
      </div>

      <div v-else class="notes">
        <NoteCard
          v-for="note in visibleNotes"
          :key="note.id"
          :note="note"
          @reply="replyTo = $event"
          @renoted="onRenoted"
          @deleted="onDeleted"
        />
      </div>

      <p v-if="!loading && !visibleNotes.length && !error" class="placeholder">
        <AppIcon name="feather" />
        <span>这里还没有笔记。</span>
      </p>

      <div v-if="hasMore" class="more">
        <button class="button ghost small" type="button" :disabled="loadingMore" @click="loadMore">
          {{ loadingMore ? "加载中…" : "加载更多" }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.feed {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feed-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}

.ghost-link {
  border: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.ghost-link:hover {
  color: var(--accent);
}

.notes {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.more {
  display: flex;
  justify-content: center;
  padding: 12px 0 0;
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
