<script setup lang="ts">
import { computed, ref } from "vue";
import type { Note } from "../../api/types";
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

// F7 replaces this fixed palette with the full emoji picker; until then the
// footer keeps the small quick set the old NoteCard offered.
const quickReactions = ["👍", "❤️", "😂", "🎉", "👀", "🤔", "🔥", "✨"];

const paletteOpen = ref(false);

const reactionCount = computed(() => props.reactionCount);

function pick(emoji: string) {
  paletteOpen.value = false;
  emit("react", emoji);
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
    <div class="react-wrap">
      <button
        type="button"
        class="action"
        title="反应"
        :disabled="busy"
        :aria-expanded="paletteOpen"
        @click="paletteOpen = !paletteOpen"
      >
        <AppIcon name="heart" />
        <span v-if="reactionCount">{{ reactionCount }}</span>
      </button>
      <div v-if="paletteOpen" class="palette" role="menu">
        <button
          v-for="emoji in quickReactions"
          :key="emoji"
          type="button"
          role="menuitem"
          :class="{ mine: myReaction === emoji }"
          @click="pick(emoji)"
        >
          {{ emoji }}
        </button>
      </div>
    </div>
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

.react-wrap {
  position: relative;
}

.palette {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  z-index: 10;
  display: flex;
  gap: 2px;
  padding: 6px;
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--panel);
  box-shadow: var(--shadow-sm);
}

.palette button {
  border: 0;
  background: transparent;
  font-size: 1.1rem;
  padding: 4px 6px;
  border-radius: 6px;
  cursor: pointer;
}

.palette button:hover,
.palette button:focus-visible,
.palette button.mine {
  background: var(--accent-soft);
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
