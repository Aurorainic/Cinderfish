// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/widgets/trends.vue.
//
// Firefish reads hashtags/trending from the backend and renders each tag with
// a MkMiniChart. Cinderfish has no hashtags/trending endpoint (TODO: missing
// backend capability), so the widget degrades: the most-used tags of the
// recent local timeline are counted client-side and the chart is omitted.
// With no data the widget shows its empty state.
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import MkLoading from "../ui/MkLoading.vue";
import MkError from "../ui/MkError.vue";

interface TagStat {
  tag: string;
  usersCount: number;
}

const stats = ref<TagStat[]>([]);
const fetching = ref(true);
const failed = ref(false);
let timer: number | undefined;

const HASHTAG = /(?:^|\s)#([^\s.,!?'":;#【】\[\]]+)/g;

function extractTags(note: Note): string[] {
  const found = new Set<string>();
  for (const match of note.text?.matchAll(HASHTAG) ?? []) {
    const tag = match[1];
    if (tag) found.add(tag);
  }
  return [...found];
}

async function fetchStats(): Promise<void> {
  fetching.value = true;
  failed.value = false;
  try {
    // TODO: Firefish calls the hashtags/trending endpoint here; Cinderfish
    // lacks it, so the recent local timeline is scanned instead.
    const notes = await api.localTimeline({ limit: 100 });
    const usersByTag = new Map<string, Set<string>>();
    for (const note of notes) {
      for (const tag of extractTags(note)) {
        let users = usersByTag.get(tag);
        if (!users) {
          users = new Set();
          usersByTag.set(tag, users);
        }
        users.add(note.userId);
      }
    }
    stats.value = [...usersByTag.entries()]
      .filter(([, users]) => users.size > 0)
      .map(([tag, users]) => ({ tag, usersCount: users.size }))
      .sort((a, b) => b.usersCount - a.usersCount)
      .slice(0, 5);
  } catch {
    failed.value = true;
  } finally {
    fetching.value = false;
  }
}

onMounted(() => {
  void fetchStats();
  timer = window.setInterval(() => void fetchStats(), 1000 * 60 * 10);
});

onBeforeUnmount(() => window.clearInterval(timer));
</script>

<template>
  <div class="trends">
    <MkLoading v-if="fetching" />
    <MkError v-else-if="failed" @retry="fetchStats" />
    <template v-else>
      <div v-if="stats.length" class="tags">
        <div v-for="stat in stats" :key="stat.tag" class="tag">
          <span class="name" :title="stat.tag">#{{ stat.tag }}</span>
          <p class="count">{{ stat.usersCount }} 人提及</p>
        </div>
      </div>
      <p v-else class="empty">最近没有热门话题</p>
    </template>
  </div>
</template>

<style scoped>
.trends {
  padding: 12px 14px;
}

.tags {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tag .name {
  display: block;
  color: var(--accent);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag .count {
  margin: 2px 0 0;
  font-size: 0.82rem;
  color: var(--text-weak, var(--muted, var(--fg)));
}

.empty {
  margin: 0;
  color: var(--text-weak, var(--muted, var(--fg)));
  font-size: 0.84rem;
  text-align: center;
}
</style>
