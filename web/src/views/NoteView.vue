<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note } from "../api/types";
import NoteItem from "../components/note/Note.vue";
import NoteDetail from "../components/note/NoteDetail.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams, PagedFeedResult } from "../components/ui/PagedFeed.vue";
import { usePageTitle } from "../composables/use-page-title";
import { absoluteTime } from "../lib/time";
import { openPostForm } from "../os";

const route = useRoute();
const router = useRouter();

const MORE_PAGE_SIZE = 10;

const note = ref<Note | null>(null);
const loading = ref(false);
const error = ref("");
const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

const authorName = computed(() => note.value?.user.name || note.value?.user.username || "");
const pageTitle = computed(() => (authorName.value ? `${authorName.value} 的笔记` : "笔记"));
const subtitle = computed(() => (note.value ? absoluteTime(note.value.createdAt) : ""));

usePageTitle(() => pageTitle.value);

async function load() {
  const id = String(route.params.id ?? "");
  if (!id) return;
  loading.value = true;
  error.value = "";
  try {
    note.value = await api.showNote(id);
  } catch (err) {
    note.value = null;
    error.value = err instanceof Error ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
}

// Older notes by the same author are the only paginated context the backend
// offers: the reply tree needs notes/children, which does not exist yet.
async function fetcher(p: PagedFeedParams): Promise<PagedFeedResult> {
  const current = note.value;
  if (!current) return { items: [], hasMore: false };
  const page = await api.usersNotes({
    userId: current.userId,
    limit: MORE_PAGE_SIZE,
    untilId: p.untilId,
  });
  return {
    items: page.filter((item) => item.id !== current.id),
    hasMore: page.length >= MORE_PAGE_SIZE,
  };
}

function onDeleted(id: string) {
  if (note.value?.id === id) router.push("/");
}

function onFeedDeleted(id: string) {
  feed.value?.remove(id);
}

function onReply(reply: Note) {
  openPostForm({ reply });
}

watch(() => route.params.id, load, { immediate: true });
</script>

<template>
  <PageHeader
    :back="true"
    :title="pageTitle"
    :subtitle="subtitle"
    :avatar="note?.user ?? null"
  />

  <div class="note-page">
    <div v-if="loading" class="skeleton"></div>

    <p v-else-if="error" class="notice error">
      <span>{{ error }}</span>
      <button type="button" class="button ghost small retry" @click="load">重试</button>
    </p>

    <template v-else-if="note">
      <NoteDetail :note="note" @deleted="onDeleted" />

      <p v-if="note.repliesCount > 0" class="hint replies-hint">当前版本暂不支持查看回复列表。</p>

      <section class="more" aria-labelledby="more-head">
        <h2 id="more-head" class="more-head">更多来自 @{{ note.user.username }}</h2>
        <PagedFeed ref="feed" :key="note.id" :fetcher="fetcher" :page-size="MORE_PAGE_SIZE">
          <template #item="{ item }">
            <NoteItem :note="(item as Note)" @reply="onReply" @deleted="onFeedDeleted" />
          </template>
          <template #empty>
            <span>没有更早的笔记了。</span>
          </template>
        </PagedFeed>
      </section>
    </template>
  </div>
</template>

<style scoped>
.note-page {
  padding-top: var(--space-8);
  min-width: 0;
}

.skeleton {
  height: 180px;
  border-radius: var(--radius-lg);
  background: var(--panel-2);
}

.retry {
  margin-inline-start: var(--space-10);
}

.replies-hint {
  margin: var(--space-10) var(--space-14) 0;
}

.more {
  margin-top: var(--space-20);
  min-width: 0;
}

.more-head {
  padding: 0 var(--space-14) var(--space-6);
  color: var(--text-weak);
  font-size: var(--fs-xs);
  font-weight: 700;
  letter-spacing: 0.04em;
}

.more :deep(.paged-feed) {
  gap: 0;
}
</style>
