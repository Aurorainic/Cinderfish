<script setup lang="ts">
import { computed, ref } from "vue";
import { api } from "../api/client";
import type { Note, TimelineParams } from "../api/types";
import { isLoggedIn } from "../stores/session";
import AppIcon from "./AppIcon.vue";
import PostForm from "./composer/PostForm.vue";
import NoteItem from "./note/Note.vue";
import PagedFeed from "./ui/PagedFeed.vue";
import type { PagedFeedParams, PagedFeedResult } from "./ui/PagedFeed.vue";

const props = withDefaults(
  defineProps<{ source: "local" | "global" | "hybrid" | "home"; tag?: string }>(),
  { tag: "" },
);

const needsAuth = computed(() => props.source === "hybrid" || props.source === "home");

const PAGE_SIZE = 20;

const timelineApi = {
  local: api.localTimeline,
  global: api.globalTimeline,
  hybrid: api.hybridTimeline,
  home: api.homeTimeline,
};

const streamChannels = {
  local: "localTimeline",
  global: "globalTimeline",
  hybrid: "hybridTimeline",
  home: "homeTimeline",
};

// When a tag filter is active the fetcher applies it client-side; PagedFeed
// derives its cursor from the last item it received, which would skip
// unfiltered notes between pages. The fetcher therefore keeps its own raw
// cursor and ignores PagedFeed's untilId while filtering, and reports
// hasMore from the raw page so an empty filtered page keeps paginating.
let rawCursor = "";

async function fetcher(p: PagedFeedParams): Promise<PagedFeedResult> {
  const params: TimelineParams = { limit: PAGE_SIZE };
  if (props.tag) {
    if (p.untilId === undefined) rawCursor = "";
    params.untilId = rawCursor || undefined;
  } else if (p.untilId) {
    params.untilId = p.untilId;
  }
  const page = await timelineApi[props.source](params);
  if (page.length) rawCursor = page[page.length - 1].id;
  const hasMore = page.length >= PAGE_SIZE;
  if (!props.tag) return page;
  const needle = `#${props.tag}`.toLowerCase();
  return { items: page.filter((n) => (n.text ?? "").toLowerCase().includes(needle)), hasMore };
}

// Streamed notes behind the pill must match the active tag filter too.
const streamFilter = computed(() => {
  if (!props.tag) return undefined;
  const needle = `#${props.tag}`.toLowerCase();
  return (item: unknown) => ((item as Note).text ?? "").toLowerCase().includes(needle);
});

const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

function onDeleted(id: string) {
  feed.value?.remove(id);
}

function toNote(item: unknown): Note {
  return item as Note;
}
</script>

<template>
  <div v-if="needsAuth && !isLoggedIn" class="card placeholder">
    <AppIcon name="globe" />
    <h2>需要登录</h2>
    <p class="small">这个频道只对已登录用户开放。</p>
    <RouterLink to="/login" class="button small">登录</RouterLink>
  </div>

  <div v-else class="timeline-feed">
    <!-- The section's own composer. Replies and quotes go through
         os.openPostForm so there is only ever one post dialog. New notes
         arrive through the streaming channel. -->
    <PostForm v-if="isLoggedIn" fixed />
    <PagedFeed
      ref="feed"
      :key="`${source}:${tag}`"
      :fetcher="fetcher"
      :stream-channel="streamChannels[source]"
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
</template>

<style scoped>
/* PagedFeed spaces its rows for cards; borderless rows butt against their
   own hairline instead. */
.timeline-feed :deep(.paged-feed) {
  gap: 0;
}
</style>
