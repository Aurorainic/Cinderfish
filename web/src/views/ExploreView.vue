<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note, TimelineParams } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteItem from "../components/note/Note.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams, PagedFeedResult } from "../components/ui/PagedFeed.vue";
import { usePageTitle } from "../composables/use-page-title";

type Scope = "local" | "global";

const SCOPES: readonly Scope[] = ["local", "global"];
const LABELS: Record<Scope, string> = { local: "本地", global: "全球" };

const PAGE_SIZE = 20;

const route = useRoute();
const router = useRouter();

usePageTitle("探索");

const tabs = SCOPES.map((key) => ({ key, label: LABELS[key] }));

const tag = computed(() => (typeof route.query.tag === "string" ? route.query.tag : ""));

const tab = computed<string>({
  get() {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    return (SCOPES as readonly string[]).includes(raw) ? raw : "local";
  },
  set(value) {
    if (route.query.tab === value) return;
    void router.push({ query: { ...route.query, tab: value } });
  },
});

const scope = computed<Scope>(() => (tab.value === "global" ? "global" : "local"));

watch(
  () => route.query.tab,
  () => {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    if (!raw || raw === tab.value) return;
    void router.replace({ query: { ...route.query, tab: tab.value } });
  },
  { immediate: true },
);

const streamChannels = { local: "localTimeline", global: "globalTimeline" };

const timelineApi = { local: api.localTimeline, global: api.globalTimeline };

// Client-side tag filter: keep a raw cursor so pages do not skip the notes
// dropped by the filter (PagedFeed's cursor comes from the filtered list),
// and report hasMore from the raw page so an empty filtered page keeps
// paginating.
let rawCursor = "";

async function fetcher(p: PagedFeedParams): Promise<PagedFeedResult> {
  const params: TimelineParams = { limit: PAGE_SIZE };
  if (tag.value) {
    if (p.untilId === undefined) rawCursor = "";
    params.untilId = rawCursor || undefined;
  } else if (p.untilId) {
    params.untilId = p.untilId;
  }
  const page = await timelineApi[scope.value](params);
  if (page.length) rawCursor = page[page.length - 1].id;
  const hasMore = page.length >= PAGE_SIZE;
  if (!tag.value) return page;
  const needle = `#${tag.value}`.toLowerCase();
  return { items: page.filter((n) => (n.text ?? "").toLowerCase().includes(needle)), hasMore };
}

// Streamed notes behind the pill must match the active tag filter too.
const streamFilter = computed(() => {
  if (!tag.value) return undefined;
  const needle = `#${tag.value}`.toLowerCase();
  return (item: unknown) => ((item as Note).text ?? "").toLowerCase().includes(needle);
});

const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

function toNote(item: unknown): Note {
  return item as Note;
}

function onDeleted(id: string) {
  feed.value?.remove(id);
}
</script>

<template>
  <PageHeader v-model:tab="tab" title="探索" :tabs="tabs" />

  <div class="page-body">
    <p v-if="tag" class="notice info">筛选话题：#{{ tag }}</p>

    <div class="explore-feed">
      <PagedFeed
        :key="`${scope}:${tag}`"
        ref="feed"
        :fetcher="fetcher"
        :stream-channel="streamChannels[scope]"
        :stream-filter="streamFilter"
      >
        <template #item="{ item }">
          <NoteItem :note="toNote(item)" @deleted="onDeleted" />
        </template>
        <template #empty>
          <div class="placeholder">
            <AppIcon name="feather" />
            <span>这里还没有笔记。</span>
          </div>
        </template>
      </PagedFeed>
    </div>
  </div>
</template>

<style scoped>
.page-body {
  padding-top: var(--space-12);
}

/* Borderless rows supply their own hairline; drop PagedFeed's card gap. */
.explore-feed :deep(.paged-feed) {
  gap: 0;
}
</style>
