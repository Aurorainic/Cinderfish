<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import type { Note as NoteModel } from "../../api/types";
import { focusNext, focusPrev } from "../../lib/focus-sibling";
import { makeHotkey } from "../../lib/hotkeys";
import { absoluteTime } from "../../lib/time";
import NoteContent from "./NoteContent.vue";
import NoteFooter from "./NoteFooter.vue";
import NoteHeader from "./NoteHeader.vue";
import NoteParent from "./NoteParent.vue";
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

const router = useRouter();
const root = ref<HTMLElement | null>(null);
const footer = ref<InstanceType<typeof NoteFooter> | null>(null);

// A note carrying a renote target and nothing of its own is a pure renote:
// the bar names the renoter and the renoted note supplies header, content and
// footer. Anything with text, a CW, files or a poll is a quote and keeps its
// own content with the quoted note rendered as a card.
const pureRenote = computed(() => {
  const note = props.note;
  if (!note.renote) return false;
  const hasOwnContent =
    !!note.text?.trim() || !!note.cw || (note.files?.length ?? 0) > 0 || !!note.hasPoll;
  return !hasOwnContent;
});

const appearNote = computed<NoteModel>(() =>
  pureRenote.value && props.note.renote ? props.note.renote : props.note,
);

const parentNote = computed(() => appearNote.value.reply ?? null);
const quoteNote = computed(() => appearNote.value.renote ?? null);

const quotePreview = computed(() => {
  const target = quoteNote.value;
  if (!target) return "";
  return target.text?.trim().slice(0, 80) || target.user.username;
});

const accessibleLabel = computed(() => {
  const note = appearNote.value;
  const parts: string[] = [];
  if (pureRenote.value) parts.push(`${props.note.user.username} 转贴`);
  parts.push(note.user.username);
  if (note.cw) parts.push(`内容警告：${note.cw}`);
  if (note.text) parts.push(note.text.slice(0, 140));
  parts.push(absoluteTime(note.createdAt));
  return parts.join("；");
});

const { reactions, myReaction, reactionCount, isMine, busy, notice, toggleReaction, renote, remove } =
  useNote(appearNote);

async function onRenote() {
  const created = await renote();
  if (created) emit("renoted", created);
}

async function onRemove() {
  if (await remove()) emit("deleted", props.note.id);
}

function openQuote(event: MouseEvent) {
  const target = event.target instanceof Element ? event.target : null;
  if (target?.closest("a, button")) return;
  if (quoteNote.value) router.push(`/notes/${quoteNote.value.id}`);
}

// A click on the row body focuses the note so the shortcuts below apply; a
// click on a link or control keeps its own focus.
function onClick(event: MouseEvent) {
  const target = event.target instanceof Element ? event.target : null;
  if (target?.closest("a, button, input, textarea, select, [contenteditable]")) return;
  root.value?.focus({ preventScroll: true });
}

const hotkeys = makeHotkey({
  r: () => footer.value?.reply(),
  "e|a|plus": () => footer.value?.react(),
  q: () => footer.value?.renote(),
  "up|k": () => {
    if (root.value) focusPrev(root.value);
  },
  "down|j": () => {
    if (root.value) focusNext(root.value);
  },
  esc: () => root.value?.blur(),
  "m|o": () => footer.value?.menu(),
});

onMounted(() => root.value?.addEventListener("keydown", hotkeys));
onBeforeUnmount(() => root.value?.removeEventListener("keydown", hotkeys));
</script>

<template>
  <article
    :id="appearNote.id"
    ref="root"
    class="note"
    :class="{ nested, detail }"
    tabindex="-1"
    :aria-label="accessibleLabel"
    @click="onClick"
  >
    <div class="note-inner">
      <NoteParent v-if="parentNote" :note="parentNote" />

      <RenoteBar v-if="pureRenote" :note="note" />

      <NoteHeader :note="appearNote" :detail="detail" />

      <NoteContent :note="appearNote" :nested="nested" :detail="detail">
        <template #quote>
          <RouterLink v-if="quoteNote && !detail" :to="`/notes/${quoteNote.id}`" class="quote-link" @click.stop>
            引用：{{ quotePreview }}
          </RouterLink>
          <div v-else-if="quoteNote" class="quote" @click.stop="openQuote">
            <NoteHeader :note="quoteNote" radius="8px" />
            <NoteContent :note="quoteNote" nested />
          </div>
        </template>
      </NoteContent>

      <NoteFooter
        ref="footer"
        :note="appearNote"
        :reactions="reactions"
        :my-reaction="myReaction"
        :reaction-count="reactionCount"
        :busy="busy"
        :is-mine="isMine"
        @reply="emit('reply', appearNote)"
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
  position: relative;
  container-name: note;
  container-type: inline-size;
  padding: 0;
  border-bottom: 1px solid var(--divider);
  background: transparent;
  min-width: 0;
}

.note-inner {
  --avatar-size: 48px;
  padding: 12px 14px 8px;
}

.note.detail {
  border-bottom: 0;
}

.note.detail .note-inner {
  padding: 16px 4px 12px;
}

/* Keyboard focus draws a ring inside the row instead of a browser outline
   that would be clipped by the container. */
.note:focus-visible {
  outline: none;
}

.note:focus-visible::after {
  content: "";
  pointer-events: none;
  position: absolute;
  z-index: 10;
  inset: 0;
  margin: auto;
  width: calc(100% - 8px);
  height: calc(100% - 8px);
  border: 1px solid var(--accent);
  border-radius: var(--radius-sm);
  box-sizing: border-box;
}

/* The action row rests dimmed; hovering or focusing the row lifts it. The
   reaction strip above it is not dimmed. */
.note:hover :deep(.actions),
.note:focus-within :deep(.actions) {
  opacity: 1;
}

.quote {
  --avatar-size: 38px;
  margin-top: 8px;
  padding: 16px;
  border: 1px solid var(--renote);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.quote:hover,
.quote:focus-within {
  background: var(--panel-2);
}

.quote-link {
  display: block;
  margin-top: 6px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--renote);
  font-size: 0.85rem;
}

.quote-link:hover {
  color: var(--accent);
}

.notice {
  margin-top: 8px;
  color: var(--danger);
}

/* The container is the row itself, so only its descendants can query it. */
@container note (max-width: 500px) {
  .note-inner {
    --avatar-size: 46px;
    font-size: 0.975em;
    padding: 11px 12px 7px;
  }

  .quote {
    padding: 12px;
  }
}

@container note (max-width: 380px) {
  .note-inner {
    --avatar-size: 40px;
    font-size: 0.94em;
    padding: 10px 10px 6px;
  }

  .quote {
    --avatar-size: 32px;
    padding: 10px;
  }
}
</style>
