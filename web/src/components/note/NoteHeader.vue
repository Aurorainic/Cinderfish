<script setup lang="ts">
import { computed } from "vue";
import type { Note } from "../../api/types";
import { absoluteTime, relativeTime } from "../../lib/time";
import UserAvatar from "../UserAvatar.vue";
import NoteVisibility from "./NoteVisibility.vue";

const props = withDefaults(
  defineProps<{ note: Note; nested?: boolean; detail?: boolean }>(),
  { nested: false, detail: false },
);

const avatarSize = computed(() => (props.nested ? 30 : 42));
const timeText = computed(() =>
  props.detail ? absoluteTime(props.note.createdAt) : relativeTime(props.note.createdAt),
);
const timeTitle = computed(() => absoluteTime(props.note.createdAt));
</script>

<template>
  <header class="head">
    <RouterLink :to="`/@${note.user.username}`" class="avatar" tabindex="-1">
      <UserAvatar :user="note.user" :size="avatarSize" />
    </RouterLink>
    <div class="who">
      <RouterLink :to="`/@${note.user.username}`" class="name">
        {{ note.user.name || note.user.username }}
      </RouterLink>
      <span class="handle">
        @{{ note.user.username }}<template v-if="note.user.host">@{{ note.user.host }}</template>
      </span>
    </div>
    <RouterLink :to="`/notes/${note.id}`" class="time" :title="timeTitle">{{ timeText }}</RouterLink>
    <NoteVisibility :visibility="note.visibility" :local-only="note.localOnly" class="vis" />
  </header>
</template>

<style scoped>
.head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.avatar {
  flex: none;
  display: block;
  line-height: 0;
}

.who {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  flex: 1;
  overflow: hidden;
}

.name {
  font-weight: 700;
  color: var(--text-strong);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 60%;
}

.name:hover {
  color: var(--accent);
}

.handle {
  color: var(--muted);
  font-size: 0.82rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.time {
  flex: none;
  color: var(--muted);
  font-size: 0.8rem;
  margin-left: auto;
}

.time:hover {
  color: var(--accent);
}

.vis {
  flex: none;
}

@container note (max-width: 580px) {
  .head {
    gap: 9px;
  }

  .time {
    font-size: 0.78rem;
  }
}

@container note (max-width: 500px) {
  .head {
    gap: 8px;
  }

  .name {
    max-width: 52%;
  }

  .handle {
    font-size: 0.78rem;
  }

  .time {
    font-size: 0.76rem;
  }
}

@container note (max-width: 450px) {
  .name {
    max-width: 46%;
  }

  .handle {
    min-width: 0;
    flex: 1;
  }
}
</style>
