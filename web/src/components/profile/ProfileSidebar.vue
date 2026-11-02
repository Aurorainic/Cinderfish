<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "../../api/client";
import type { DriveFile, Note, UserDetailed } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import { isImageType } from "../../lib/mfm";
import { relativeTime } from "../../lib/time";

// The profile side column: a media strip and a short activity digest. Both
// come from one users/notes read, so the column costs a single request and
// never invents data the API cannot answer.
const props = defineProps<{ user: UserDetailed }>();

const SIDE_LIMIT = 20;
const PHOTO_LIMIT = 9;
const ACTIVITY_LIMIT = 5;

interface Photo {
  note: Note;
  file: DriveFile;
}

const notes = ref<Note[]>([]);
const loading = ref(false);
const error = ref("");

const photos = computed<Photo[]>(() => {
  const out: Photo[] = [];
  for (const note of notes.value) {
    for (const file of note.files ?? []) {
      if (!isImageType(file.type)) continue;
      out.push({ note, file });
      if (out.length >= PHOTO_LIMIT) return out;
    }
  }
  return out;
});

const activity = computed(() => notes.value.slice(0, ACTIVITY_LIMIT));

function excerpt(note: Note): string {
  const text = (note.text ?? "").replace(/\s+/g, " ").trim();
  if (text) return text.length > 42 ? `${text.slice(0, 42)}…` : text;
  const files = note.files?.length ?? 0;
  if (files > 0) return `${files} 个附件`;
  if (note.renote) return `转贴 @${note.renote.user.username}`;
  return "（无正文）";
}

function photoUrl(file: DriveFile): string {
  return file.thumbnailUrl || file.url;
}

async function load() {
  notes.value = [];
  error.value = "";
  if (!props.user.id) return;
  loading.value = true;
  try {
    notes.value = await api.usersNotes({
      userId: props.user.id,
      limit: SIDE_LIMIT,
      withReplies: false,
    });
  } catch (err) {
    error.value = err instanceof Error && err.message ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.user.id,
  () => {
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <aside class="side">
    <section class="_panel side-card">
      <header class="side-head">
        <AppIcon name="image" />
        <h2>文件</h2>
      </header>

      <div v-if="loading" class="grid" aria-hidden="true">
        <span v-for="i in 6" :key="i" class="tile skeleton"></span>
      </div>
      <p v-else-if="error" class="side-empty">
        <span>{{ error }}</span>
        <button type="button" class="retry" @click="load">重试</button>
      </p>
      <div v-else-if="photos.length" class="grid">
        <RouterLink
          v-for="photo in photos"
          :key="photo.note.id + photo.file.id"
          class="tile"
          :to="`/notes/${photo.note.id}`"
          :title="photo.file.name"
        >
          <img :src="photoUrl(photo.file)" :alt="photo.file.name" loading="lazy" />
        </RouterLink>
      </div>
      <p v-else class="_caption side-empty">还没有图片。</p>
    </section>

    <section class="_panel side-card">
      <header class="side-head">
        <AppIcon name="feather" />
        <h2>动态</h2>
      </header>

      <ul v-if="activity.length" class="activity">
        <li v-for="note in activity" :key="note.id" class="activity-row">
          <RouterLink class="activity-text" :to="`/notes/${note.id}`">{{ excerpt(note) }}</RouterLink>
          <span class="activity-time">{{ relativeTime(note.createdAt) }}</span>
        </li>
      </ul>
      <p v-else-if="!loading" class="_caption side-empty">还没有动态。</p>
    </section>
  </aside>
</template>

<style scoped>
.side {
  display: flex;
  flex-direction: column;
  gap: var(--margin);
  min-width: 0;
}

.side-card {
  padding: 14px;
  min-width: 0;
}

.side-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-block-end: 10px;
  color: var(--fgTransparentWeak);
}

.side-head .icon {
  inline-size: 15px;
  block-size: 15px;
}

.side-head h2 {
  font-size: 0.85em;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
  gap: 4px;
}

.tile {
  display: block;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: 4px;
  background: var(--panelHighlight);
}

.tile img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tile:hover,
.tile:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: -2px;
}

.skeleton {
  background: linear-gradient(90deg, var(--panelHighlight) 25%, var(--panel) 50%, var(--panelHighlight) 75%);
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

.side-empty {
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
}

.retry {
  margin-inline-start: 8px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--accent);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.retry:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 2px;
}

.activity {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.activity-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.activity-text {
  color: var(--fg);
  font-size: 0.9em;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.activity-text:hover {
  color: var(--accent);
}

.activity-time {
  color: var(--fgTransparent);
  font-size: 0.8em;
}
</style>
