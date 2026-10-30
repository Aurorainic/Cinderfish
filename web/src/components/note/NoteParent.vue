<script setup lang="ts">
import type { Note } from "../../api/types";
import UserAvatar from "../UserAvatar.vue";
import NoteContent from "./NoteContent.vue";
import NoteHeader from "./NoteHeader.vue";

// The note this one replies to, rendered above it. Only the single parent
// carried by `note.reply` is shown: the API has no conversation endpoint yet,
// so no further ancestors are invented.
defineProps<{ note: Note }>();
</script>

<template>
  <div class="parent">
    <div class="avatar-column">
      <UserAvatar :user="note.user" size="inherit" radius="8px" />
      <span class="line" aria-hidden="true"></span>
    </div>
    <div class="body">
      <NoteHeader :note="note" :avatar="false" />
      <NoteContent :note="note" nested />
    </div>
  </div>
</template>

<style scoped>
.parent {
  --avatar-size: 38px;
  position: relative;
  display: grid;
  grid-template-columns: var(--avatar-size) minmax(0, 1fr);
  align-items: start;
  gap: 10px;
  margin-bottom: 8px;
  min-width: 0;
}

.avatar-column {
  color: var(--muted);
}

.avatar-column :deep(.avatar) {
  display: block;
}

/* Runs from under the parent avatar past the row's bottom edge, joining the
   reply below. Absolutely placed against the row so it never feeds back into
   the row height. */
.line {
  position: absolute;
  left: calc(var(--avatar-size) / 2 - 1px);
  top: calc(var(--avatar-size) + 4px);
  bottom: -8px;
  width: 2px;
  border-radius: 1px;
  background: currentColor;
  opacity: 0.3;
}

.body {
  min-width: 0;
}

@container note (max-width: 380px) {
  .parent {
    --avatar-size: 32px;
    gap: 8px;
  }
}
</style>
