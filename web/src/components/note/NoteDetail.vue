<script setup lang="ts">
import { computed } from "vue";
import type { Note } from "../../api/types";
import { openPostForm } from "../../os";
import NoteItem from "./Note.vue";

// The single note body shared by the note page and the deck overlay. The reply
// context above it comes from Note.vue, which renders `note.reply` as a
// NoteParent row; nothing older exists on the API, so no further ancestors or
// reply tree are invented here.
const props = defineProps<{ note: Note }>();

const emit = defineEmits<{ deleted: [id: string] }>();

const remoteHost = computed(() => props.note.user.host ?? "");
const remoteUrl = computed(() => props.note.url ?? "");

function onReply(note: Note) {
  openPostForm({ reply: note });
}

function onDeleted(id: string) {
  emit("deleted", id);
}
</script>

<template>
  <div class="note-detail">
    <p v-if="remoteHost" class="notice info remote">
      <span>这条笔记来自 {{ remoteHost }}。</span>
      <a v-if="remoteUrl" :href="remoteUrl" target="_blank" rel="noopener noreferrer">去原站查看</a>
    </p>

    <NoteItem :note="note" detail @reply="onReply" @deleted="onDeleted" />
  </div>
</template>

<style scoped>
.note-detail {
  padding-bottom: var(--space-8);
  border-bottom: 1px solid var(--divider);
  min-width: 0;
}

.remote {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-8);
  margin: var(--space-12) 0 0;
}

.remote a {
  color: var(--accent);
  font-weight: 600;
}

.remote a:hover {
  color: var(--accent-hover);
}
</style>
