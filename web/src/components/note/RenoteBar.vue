<script setup lang="ts">
import { computed } from "vue";
import type { Note } from "../../api/types";
import { absoluteTime, relativeTime } from "../../lib/time";
import AppIcon from "../AppIcon.vue";
import NoteVisibility from "./NoteVisibility.vue";
import UserAvatar from "../UserAvatar.vue";

// The bar above a renoted note: who renoted, when, and with what reach. The
// renoted note itself renders below with its own header.
const props = defineProps<{ note: Note }>();

const target = computed(() => props.note.renote);
</script>

<template>
  <div v-if="target" class="renote-bar">
    <AppIcon name="renote" class="mark" />
    <RouterLink :to="`/@${note.user.username}`" class="avatar" tabindex="-1">
      <UserAvatar :user="note.user" size="inherit" />
    </RouterLink>
    <RouterLink :to="`/@${note.user.username}`" class="name">
      {{ note.user.name || note.user.username }}
    </RouterLink>
    <span class="verb">转贴了</span>
    <RouterLink :to="`/notes/${note.id}`" class="time" :title="absoluteTime(note.createdAt)">
      {{ relativeTime(note.createdAt) }}
    </RouterLink>
    <NoteVisibility :visibility="note.visibility" :local-only="note.localOnly" />
  </div>
</template>

<style scoped>
.renote-bar {
  --avatar-size: 20px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 0 6px;
  color: var(--renote);
  font-size: 0.8rem;
  min-width: 0;
}

.mark {
  width: 0.95rem;
  height: 0.95rem;
  flex: none;
}

.avatar {
  line-height: 0;
  flex: none;
}

.name {
  color: var(--renote);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 40%;
}

.name:hover {
  color: var(--accent);
}

.verb,
.time {
  color: var(--muted);
  flex: none;
}

.time {
  margin-left: auto;
}

.time:hover {
  color: var(--accent);
}

@container note (max-width: 500px) {
  .renote-bar {
    gap: 5px;
  }
}

@container note (max-width: 380px) {
  .renote-bar {
    font-size: 0.76rem;
  }

  .name {
    max-width: 32%;
  }
}
</style>
