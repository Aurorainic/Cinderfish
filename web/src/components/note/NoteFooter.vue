<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api, ApiError } from "../../api/client";
import type { Note } from "../../api/types";
import { openReactionPicker } from "../../os";
import { session } from "../../stores/session";
import AppIcon from "../AppIcon.vue";

const props = defineProps<{
  note: Note;
  reactionCount: number;
  myReaction?: string;
  busy?: boolean;
  isMine?: boolean;
}>();

const emit = defineEmits<{
  reply: [];
  renote: [];
  react: [emoji: string];
  remove: [];
}>();

const LIKE_ONLY = new Set([
  "likeOnly",
  "likeOnlyForRemote",
  "nonSensitiveOnlyForLocalLikeOnlyForRemote",
]);

const acceptance = computed<string | null>(() => props.note.reactionAcceptance ?? null);

const likeOnly = computed(() => acceptance.value !== null && LIKE_ONLY.has(acceptance.value));

const reactionCount = computed(() => props.reactionCount);

const reactButton = ref<HTMLButtonElement | null>(null);

function onReact() {
  if (props.busy) return;
  if (likeOnly.value) {
    emit("react", "⭐");
    return;
  }
  openReactionPicker((emoji) => emit("react", emoji), reactButton.value);
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
const pinned = computed(() => session.me?.pinnedNoteIds?.includes(props.note.id) ?? false);
const pinLabel = computed(() => (pinned.value ? "取消置顶" : "置顶笔记"));

async function togglePin() {
  menuOpen.value = false;
  if (pinning.value || !session.me) return;
  pinning.value = true;
  pinNotice.value = "";
  try {
    const next = pinned.value ? await api.unpin(props.note.id) : await api.pin(props.note.id);
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
</script>

<template>
  <footer class="foot" :class="{ busy }">
    <button type="button" class="action" title="回复" :disabled="busy" @click="emit('reply')">
      <AppIcon name="reply" />
      <span v-if="note.repliesCount">{{ note.repliesCount }}</span>
    </button>
    <button type="button" class="action" title="转贴" :disabled="busy" @click="emit('renote')">
      <AppIcon name="renote" />
      <span v-if="note.renoteCount">{{ note.renoteCount }}</span>
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
    <span v-if="pinNotice" class="pin-notice" role="status">{{ pinNotice }}</span>
    <span class="spacer"></span>
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
  </footer>
</template>

<style scoped>
/* Hidden until the row is hovered, but never hidden from keyboard users:
   :focus-within keeps the actions reachable by Tab. */
.foot {
  position: relative;
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: 6px;
  margin-left: -6px;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.foot:focus-within,
.foot:has(:focus-visible) {
  opacity: 1;
}

.action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 0.8rem;
  line-height: 1.5;
}

.action:hover,
.action:focus-visible {
  color: var(--accent);
  background: var(--accent-soft);
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

.spacer {
  flex: 1;
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
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.28);
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
  border-radius: 999px;
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
  font-size: 0.75rem;
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

/* Touch devices have no hover state, so the actions stay visible. */
@media (hover: none) {
  .foot {
    opacity: 1;
  }
}

@container note (max-width: 580px) {
  .foot {
    gap: 0;
    margin-left: -4px;
  }

  .action {
    padding: 4px 7px;
  }
}

@container note (max-width: 450px) {
  .action {
    padding: 4px 6px;
    font-size: 0.76rem;
  }
}
</style>
