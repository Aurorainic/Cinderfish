<script setup lang="ts">
import { computed } from "vue";
import type { Note } from "../api/types";
import { isImageType, renderMFM } from "../lib/mfm";
import { relativeTime } from "../lib/time";
import UserAvatar from "./UserAvatar.vue";

const props = defineProps<{ note: Note }>();

const html = computed(() => renderMFM(props.note.text));
const images = computed(() => props.note.files.filter((f) => isImageType(f.type)));
</script>

<template>
  <article class="quote">
    <header class="quote-head">
      <RouterLink :to="`/@${note.user.username}`">
        <UserAvatar :user="note.user" :size="28" />
      </RouterLink>
      <RouterLink :to="`/@${note.user.username}`" class="name">
        {{ note.user.name || note.user.username }}
      </RouterLink>
      <span class="handle">
        @{{ note.user.username }}<template v-if="note.user.host">@{{ note.user.host }}</template>
      </span>
      <RouterLink :to="`/notes/${note.id}`" class="time">{{ relativeTime(note.createdAt) }}</RouterLink>
    </header>
    <div v-if="html" class="body" v-html="html"></div>
    <div v-if="images.length" class="images">
      <img v-for="file in images" :key="file.id" :src="file.url" :alt="file.name" loading="lazy" />
    </div>
  </article>
</template>

<style scoped>
.quote {
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
}

.quote-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  min-width: 0;
}

.name {
  font-weight: 700;
  color: var(--text-strong);
}

.handle,
.time {
  color: var(--muted);
  font-size: 0.8rem;
}

.time {
  margin-left: auto;
}

.body {
  word-break: break-word;
}

.body :deep(.mfm-link),
.body :deep(.mfm-mention),
.body :deep(.mfm-hashtag) {
  color: var(--link);
}

.images {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}

.images img {
  width: 96px;
  height: 96px;
  object-fit: cover;
  border-radius: var(--radius-sm);
}
</style>
