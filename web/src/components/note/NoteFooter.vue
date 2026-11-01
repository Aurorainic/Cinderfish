<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api, ApiError } from "../../api/client";
import type { Note } from "../../api/types";
import { openReactionPicker } from "../../os";
import { session } from "../../stores/session";
import AppIcon from "../AppIcon.vue";
import NoteReactions from "./NoteReactions.vue";

const props = defineProps<{
  note: Note;
  // The note the row stands for when it differs from the displayed one, as in
  // a pure renote where `note` is the renoted note. Pinning applies to it.
  actionNote?: Note;
  reactions: Record<string, number>;
  reactionCount: number;
  myReaction?: string;
  busy?: boolean;
  isMine?: boolean;
  // Whether the displayed note may be renoted: Misskey allows public and
  // home notes, and a followers note only for its author (the backend
  // enforces CANNOT_RENOTE_DUE_TO_VISIBILITY). Specified notes are never
  // renotable. False renders the locked disabled button like MkRenoteButton.
  renotable?: boolean;
}>();

const emit = defineEmits<{
  reply: [];
  renote: [];
  react: [emoji: string];
  quote: [];
  remove: [];
}>();

const LIKE_ONLY = new Set([
  "likeOnly",
  "likeOnlyForRemote",
  "nonSensitiveOnlyForLocalLikeOnlyForRemote",
]);

const acceptance = computed<string | null>(() => props.note.reactionAcceptance ?? null);

const likeOnly = computed(() => acceptance.value !== null && LIKE_ONLY.has(acceptance.value));

const reactButton = ref<HTMLButtonElement | null>(null);

function onReact() {
  if (props.busy) return;
  if (likeOnly.value) {
    emit("react", "⭐");
    return;
  }
  openReactionPicker((emoji) => emit("react", emoji), reactButton.value);
}

function reply() {
  emit("reply");
}

function renote() {
  // The button is disabled when the target is not renotable; the gate lives
  // here as well so the `q` hotkey and any other caller are refused
  // identically instead of firing a request the backend must reject.
  if (props.renotable === false) return;
  emit("renote");
}

// Pinning is an action on the note itself, not on its content, so the menu
// lives in the footer. The session copy of the user carries pinnedNoteIds:
// pin/unpin answer with the packed MeDetailed user, and replacing the
// session makes every footer's menu label flip without another request.
const menuButton = ref<HTMLButtonElement | null>(null);
const menuOpen = ref(false);
const pinning = ref(false);
const pinNotice = ref("");

const canPin = computed(() => !!props.isMine && !!session.me);
const actionNoteId = computed(() => props.actionNote?.id ?? props.note.id);
const pinned = computed(() => session.me?.pinnedNoteIds?.includes(actionNoteId.value) ?? false);
const pinLabel = computed(() => (pinned.value ? "取消置顶" : "置顶笔记"));

function menu() {
  if (!canPin.value) return;
  menuOpen.value = true;
}

async function togglePin() {
  menuOpen.value = false;
  if (pinning.value || !session.me) return;
  pinning.value = true;
  pinNotice.value = "";
  try {
    const next = pinned.value ? await api.unpin(actionNoteId.value) : await api.pin(actionNoteId.value);
    session.me = next;
  } catch (err) {
    if (err instanceof ApiError && err.code === "PIN_LIMIT_EXCEEDED") {
      flashPin("最多只能置顶 5 条笔记。");
    } else if (err instanceof ApiError && err.code === "NO_SUCH_NOTE") {
      flashPin("这条笔记已经不存在了。");
    } else {
      flashPin(err instanceof Error && err.message ? err.message : "操作失败");
    }
  } finally {
    pinning.value = false;
  }
}

function onDocumentPointer(event: PointerEvent) {
  if (!menuOpen.value) return;
  const target = event.target as Node | null;
  if (target && menuButton.value?.parentElement?.contains(target)) return;
  menuOpen.value = false;
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") menuOpen.value = false;
}

let noticeTimer: number | undefined;

function flashPin(text: string) {
  pinNotice.value = text;
  window.clearTimeout(noticeTimer);
  noticeTimer = window.setTimeout(() => (pinNotice.value = ""), 4000);
}

onMounted(() => {
  document.addEventListener("pointerdown", onDocumentPointer);
  document.addEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", onDocumentPointer);
  document.removeEventListener("keydown", onKeydown);
  window.clearTimeout(noticeTimer);
});

defineExpose({ reply, react: onReact, renote, menu });
</script>

<template>
  <footer class="foot">
    <NoteReactions
      :reactions="reactions"
      :my-reaction="myReaction"
      :busy="busy"
      @react="emit('react', $event)"
    />

    <div class="actions">
      <button type="button" class="action" title="回复" :disabled="busy" @click="reply">
        <AppIcon name="reply" />
        <span v-if="note.repliesCount">{{ note.repliesCount }}</span>
      </button>
      <button
        v-if="renotable !== false"
        type="button"
        class="action"
        title="转贴"
        :disabled="busy"
        @click="renote"
      >
        <AppIcon name="renote" />
        <span v-if="note.renoteCount">{{ note.renoteCount }}</span>
      </button>
      <button
        v-else
        type="button"
        class="action"
        title="此笔记的可见性不允许转贴"
        disabled
      >
        <AppIcon name="lock" />
      </button>
      <button
        ref="reactButton"
        type="button"
        class="action"
        :class="{ reacted: !!myReaction }"
        :title="likeOnly ? (myReaction ? '取消星标' : '星标') : '反应'"
        :aria-label="likeOnly ? '星标' : '添加反应'"
        :aria-haspopup="likeOnly ? undefined : 'dialog'"
        :disabled="busy"
        @click="onReact"
      >
        <span v-if="likeOnly" class="star" aria-hidden="true">⭐</span>
        <AppIcon v-else name="heart" />
        <span v-if="reactionCount">{{ reactionCount }}</span>
      </button>

      <button
        v-if="renotable !== false"
        type="button"
        class="action"
        title="引用"
        :disabled="busy"
        @click="emit('quote')"
      >
        <AppIcon name="quote" />
      </button>

      <div class="tail">
        <div v-if="canPin" class="menu-wrap">
          <button
            ref="menuButton"
            type="button"
            class="action"
            title="更多"
            aria-label="更多操作"
            aria-haspopup="menu"
            :aria-expanded="menuOpen"
            :disabled="busy || pinning"
            @click="menuOpen = !menuOpen"
          >
            <AppIcon name="more" />
          </button>
          <ul v-if="menuOpen" class="menu" role="menu">
            <li role="none">
              <button type="button" role="menuitem" class="menu-item" :disabled="pinning" @click="togglePin">
                {{ pinLabel }}
              </button>
            </li>
          </ul>
        </div>
        <button v-if="isMine" type="button" class="action danger" title="删除" :disabled="busy" @click="emit('remove')">
          <AppIcon name="close" />
        </button>
      </div>
    </div>

    <p v-if="pinNotice" class="pin-notice" role="status">{{ pinNotice }}</p>
  </footer>
</template>

<style scoped>
/* Firefish MkNoteFooter.vue .footer: flex row, margin-top 0.4em, buttons with
   flex-grow 1 capped at 3.5em, opacity 0.7 at rest, count offset 8px. The
   reaction strip sits above the action row; only the actions dim at rest. */
.foot {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 0.4em;
  min-width: 0;
}

.actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
  margin-inline-start: -0.5em;
  opacity: 0.7;
  transition: opacity 0.2s ease;
}

/* Equal shares, capped at 3.5em so a wide note does not stretch three icons
   across the whole row the way a full-width bar would. */
.actions > .action {
  flex: 1 1 0;
  max-width: 3.5em;
  min-width: max-content;
}

.tail {
  display: flex;
  align-items: center;
  gap: 2px;
  flex: none;
  margin-inline-start: auto;
}

.action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  padding: 8px;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1.5;
  transition: opacity 0.2s ease;
}

/* Firefish offsets the count from the icon with an 8px margin. */
.action .count {
  margin-inline-start: 8px;
  opacity: 0.7;
}

.action:hover,
.action:focus-visible {
  color: var(--fgHighlighted);
  background: transparent;
}

.action.reacted {
  color: var(--accent);
}

.star {
  font-size: 0.95rem;
  line-height: 1;
}

.action:disabled {
  opacity: 0.5;
  cursor: default;
}

.action.danger:hover,
.action.danger:focus-visible {
  color: var(--danger);
  background: color-mix(in srgb, var(--danger) 12%, transparent);
}

.menu-wrap {
  position: relative;
}

/* Opens upward: the note rows sit in a scrolling column, so a dropdown
   below the footer would be clipped by the next row. */
.menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 4px);
  z-index: 20;
  min-width: 148px;
  margin: 0;
  padding: 4px;
  list-style: none;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.menu-item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: calc(var(--radius) - 4px);
  background: transparent;
  color: var(--text);
  font-size: 0.85rem;
  text-align: left;
  cursor: pointer;
}

.menu-item:hover,
.menu-item:focus-visible {
  background: var(--accent-soft);
  color: var(--accent);
}

.menu-item:disabled {
  opacity: 0.6;
  cursor: default;
}

.pin-notice {
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

@container note (max-width: 500px) {
  .actions {
    margin-inline-start: -4px;
  }

  .action {
    padding: 4px 7px;
  }
}

@container note (max-width: 380px) {
  .action {
    padding: 4px 6px;
    font-size: 0.76rem;
  }
}
</style>
