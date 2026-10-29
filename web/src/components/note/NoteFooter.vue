<script setup lang="ts">
import { computed, ref } from "vue";
import type { Note } from "../../api/types";
import { openReactionPicker } from "../../os";
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

// `reactionAcceptance` is packed by the notes handler but is not part of the
// shared Note type yet, so it is read through a narrow intersection.
const acceptance = computed<string | null>(
  () => (props.note as Note & { reactionAcceptance?: string | null }).reactionAcceptance ?? null,
);

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
    <span class="spacer"></span>
    <button v-if="isMine" type="button" class="action danger" title="删除" :disabled="busy" @click="emit('remove')">
      <AppIcon name="close" />
    </button>
  </footer>
</template>

<style scoped>
/* Hidden until the row is hovered, but never hidden from keyboard users:
   :focus-within keeps the actions reachable by Tab. */
.foot {
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
