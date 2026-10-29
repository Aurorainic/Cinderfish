<script setup lang="ts">
import { computed, toRef } from "vue";
import type { Note as NoteModel } from "../../api/types";
import NoteContent from "./NoteContent.vue";
import NoteFooter from "./NoteFooter.vue";
import NoteHeader from "./NoteHeader.vue";
import RenoteBar from "./RenoteBar.vue";
import { useNote } from "./use-note";

const props = withDefaults(
  defineProps<{ note: NoteModel; nested?: boolean; detail?: boolean }>(),
  { nested: false, detail: false },
);

const emit = defineEmits<{
  reply: [note: NoteModel];
  deleted: [id: string];
  renoted: [note: NoteModel];
}>();

const {
  reactions,
  myReaction,
  reactionCount,
  isMine,
  busy,
  notice,
  toggleReaction,
  renote,
  remove,
} = useNote(toRef(props, "note"));

// A note with a renote target and nothing of its own is a pure renote: the
// bar names the renoter and the renoted note is rendered in full below it.
// Anything with text, a CW, files or a poll is a quote and stays inline.
const pureRenote = computed(() => {
  const note = props.note;
  if (!note.renote) return false;
  const hasOwnContent =
    !!note.text?.trim() || !!note.cw || (note.files?.length ?? 0) > 0 || !!note.hasPoll;
  return !hasOwnContent;
});

async function onRenote() {
  const created = await renote();
  if (created) emit("renoted", created);
}

async function onRemove() {
  if (await remove()) emit("deleted", props.note.id);
}
</script>

<template>
  <article v-if="pureRenote && note.renote" class="note is-renote">
    <RenoteBar :note="note" />
    <Note
      :note="note.renote"
      :detail="detail"
      @reply="emit('reply', $event)"
      @deleted="emit('deleted', $event)"
      @renoted="emit('renoted', $event)"
    />
  </article>

  <article v-else class="note" :class="{ nested, detail }">
    <div class="note-inner">
      <NoteHeader :note="note" :nested="nested" :detail="detail" />

      <p v-if="note.reply" class="reply-context">
        回复
        <RouterLink :to="`/@${note.reply.user.username}`">@{{ note.reply.user.username }}</RouterLink>
      </p>

      <NoteContent
        :note="note"
        :nested="nested"
        :reactions="reactions"
        :my-reaction="myReaction"
        @react="toggleReaction"
      >
        <template #quote>
          <div v-if="note.renote" class="quote">
            <Note :note="note.renote" nested />
          </div>
        </template>
      </NoteContent>

      <NoteFooter
        v-if="!nested"
        :note="note"
        :reaction-count="reactionCount"
        :my-reaction="myReaction"
        :busy="busy"
        :is-mine="isMine"
        @reply="emit('reply', note)"
        @renote="onRenote"
        @react="toggleReaction"
        @remove="onRemove"
      />

      <p v-if="notice" class="notice small">{{ notice }}</p>
    </div>
  </article>
</template>

<style scoped>
/* Each row is its own size container, so the note adapts to a 350px deck
   column and a 680px main column without knowing anything about the shell. */
.note {
  container-name: note;
  container-type: inline-size;
  padding: 0;
  border-bottom: 1px solid var(--divider);
  border-radius: 0;
  background: transparent;
  min-width: 0;
}

.note-inner {
  padding: 12px 14px 8px;
}

/* Quoted note inside another note: indented, dashed rule, no card. */
.note.nested {
  border-bottom: 0;
}

.note.nested .note-inner {
  padding: 10px 0 0 12px;
  border-left: 2px dashed var(--border);
}

/* Pure renote: the bar and the renoted note paint their own padding and
   hairline, so the wrapper stays empty. */
.note.is-renote {
  border-bottom: 0;
}

.note.detail {
  border-bottom: 0;
}

.note.detail .note-inner {
  padding: 16px 4px 12px;
}

/* Actions surface when the row is hovered or holds keyboard focus; the
   footer itself keeps :focus-within so Tab reaches it while invisible. */
.note:hover :deep(.foot),
.note:focus-within :deep(.foot) {
  opacity: 1;
}

/* The detail view never hides its actions. */
.note.detail :deep(.foot) {
  opacity: 1;
}

.reply-context {
  margin-top: 4px;
  font-size: 0.8rem;
  color: var(--muted);
}

.reply-context a {
  color: var(--mention);
}

.quote {
  margin-top: 10px;
}

.notice {
  margin-top: 8px;
  color: var(--danger);
}

/* The container is the row itself, so only its descendants can query it. */
@container note (max-width: 580px) {
  .note-inner {
    padding: 11px 12px 7px;
  }
}

@container note (max-width: 450px) {
  .note-inner {
    padding: 10px 10px 6px;
  }
}
</style>
