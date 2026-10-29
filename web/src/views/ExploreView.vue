<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import { api } from "../api/client";
import type { Note, TimelineParams } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteCard from "../components/NoteCard.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";

const route = useRoute();
const scope = ref<"local" | "global">("local");
const tag = computed(() => (typeof route.query.tag === "string" ? route.query.tag : ""));

const PAGE_SIZE = 20;

const streamChannels = { local: "localTimeline", global: "globalTimeline" };

const timelineApi = { local: api.localTimeline, global: api.globalTimeline };

// Client-side tag filter: keep a raw cursor so pages do not skip the notes
// dropped by the filter (PagedFeed's cursor comes from the filtered list).
let rawCursor = "";

async function fetcher(p: { untilId?: string; sinceId?: string }): Promise<Note[]> {
  const params: TimelineParams = { limit: PAGE_SIZE };
  if (tag.value) {
    if (p.untilId === undefined) rawCursor = "";
    params.untilId = rawCursor || undefined;
  } else if (p.untilId) {
    params.untilId = p.untilId;
  }
  const page = await timelineApi[scope.value](params);
  if (page.length) rawCursor = page[page.length - 1].id;
  if (!tag.value) return page;
  const needle = `#${tag.value}`.toLowerCase();
  return page.filter((n) => (n.text ?? "").toLowerCase().includes(needle));
}

const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

function toNote(item: unknown): Note {
  return item as Note;
}

function onDeleted(id: string) {
  feed.value?.remove(id);
}
</script>

<template>
  <header class="page-title">
    <h1>探索</h1>
  </header>

  <p v-if="tag" class="notice info">筛选话题：#{{ tag }}</p>

  <nav class="tabs">
    <button type="button" class="tab" :class="{ active: scope === 'local' }" @click="scope = 'local'">本地</button>
    <button type="button" class="tab" :class="{ active: scope === 'global' }" @click="scope = 'global'">全球</button>
  </nav>

  <PagedFeed :key="`${scope}:${tag}`" ref="feed" :fetcher="fetcher" :stream-channel="streamChannels[scope]">
    <template #item="{ item }">
      <NoteCard :note="toNote(item)" @deleted="onDeleted" />
    </template>
    <template #empty>
      <div class="placeholder">
        <AppIcon name="feather" />
        <span>这里还没有笔记。</span>
      </div>
    </template>
  </PagedFeed>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 6px;
  padding: 5px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel);
  width: fit-content;
}

.tab {
  padding: 7px 18px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  cursor: pointer;
}

.tab.active {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
