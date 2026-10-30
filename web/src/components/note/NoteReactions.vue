<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { emojiName } from "../../lib/emoji-catalog";
import { isLoggedIn } from "../../stores/session";

// The reaction strip above the action row: one line that scrolls sideways
// instead of wrapping, so a note with many reactions never grows taller.
const props = withDefaults(
  defineProps<{
    reactions: Record<string, number>;
    myReaction?: string;
    busy?: boolean;
  }>(),
  { myReaction: undefined, busy: false },
);

const emit = defineEmits<{ react: [emoji: string] }>();

const entries = computed(() => Object.entries(props.reactions).filter(([, count]) => count > 0));

// Remote reactions (`reaction@host`) belong to the remote instance's custom
// emoji set and cannot be toggled from here.
function canToggle(emoji: string): boolean {
  return isLoggedIn.value && !emoji.includes("@");
}

function toggle(emoji: string) {
  if (props.busy || !canToggle(emoji)) return;
  emit("react", emoji);
}

// Reactions that were not on screen when the strip first mounted animate in;
// ones that were already there do not. Mirrors Firefish's `newlyAdded`.
const seen = new Set(Object.keys(props.reactions));
const fresh = ref(new Set<string>());

watch(
  () => Object.keys(props.reactions),
  (keys) => {
    const added = keys.filter((key) => !seen.has(key));
    if (!added.length) return;
    for (const key of added) {
      seen.add(key);
      fresh.value.add(key);
    }
    window.setTimeout(() => {
      for (const key of added) fresh.value.delete(key);
    }, 400);
  },
);
</script>

<template>
  <div v-if="entries.length" class="reactions" role="group" aria-label="反应">
    <button
      v-for="[emoji, count] in entries"
      :key="emoji"
      type="button"
      class="reaction"
      :class="{ mine: myReaction === emoji, toggleable: canToggle(emoji), fresh: fresh.has(emoji) }"
      :title="`反应 ${emojiName(emoji)}`"
      :aria-label="`${emojiName(emoji)}，${count} 个反应${myReaction === emoji ? '，已选择' : ''}`"
      :aria-pressed="myReaction === emoji"
      :aria-disabled="!canToggle(emoji)"
      @click.stop="toggle(emoji)"
    >
      <span class="emoji">{{ emoji }}</span>
      <span class="count">{{ count }}</span>
    </button>
  </div>
</template>

<style scoped>
.reactions {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 2px;
  width: auto;
  overflow-x: auto;
  overflow-y: hidden;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
  padding-inline: 10px;
  margin-inline: -10px;
  /* Both edges fade: the strip can be scrolled in either direction. */
  mask: linear-gradient(to right, transparent 0, black 10px, black calc(100% - 10px), transparent 100%);
  -webkit-mask: linear-gradient(to right, transparent 0, black 10px, black calc(100% - 10px), transparent 100%);
}

.reactions::-webkit-scrollbar {
  display: none;
}

.reaction {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: none;
  block-size: 32px;
  padding: 0 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1;
  transition: background 0.15s ease, color 0.15s ease;
}

.reaction.toggleable {
  background: color-mix(in srgb, var(--text) 7%, transparent);
}

.reaction.toggleable:hover,
.reaction.toggleable:focus-visible {
  background: color-mix(in srgb, var(--text) 12%, transparent);
  color: var(--text);
}

.reaction:not(.toggleable) {
  cursor: default;
}

.reaction .emoji {
  font-size: 1.5em;
  line-height: 1;
  transition: transform 0.25s cubic-bezier(0, 0, 0, 3);
}

.reaction:active .emoji {
  transform: scale(0.85);
}

.reaction.mine {
  order: -1;
  background: var(--accent);
  color: var(--on-accent);
}

.reaction.mine:hover,
.reaction.mine:focus-visible {
  background: var(--accent-strong);
  color: var(--on-accent);
}

.reaction .count {
  font-size: 0.9em;
  font-weight: 500;
}

.reaction.mine .count {
  font-weight: 700;
}

.reaction.fresh {
  animation: reaction-scale-in 0.3s cubic-bezier(0, 0, 0, 1.2);
}

.reaction.fresh .emoji {
  animation: reaction-emoji-in 0.4s cubic-bezier(0.7, 0, 0, 1.5);
}

/* Transform-only: a scale-in that never leaves the pill transparent, so a
   reaction still shows up when animations are throttled (hidden tab) or
   skipped. */
@keyframes reaction-scale-in {
  from {
    transform: scale(0.6);
  }
  to {
    transform: scale(1);
  }
}

@keyframes reaction-emoji-in {
  from {
    transform: scale(0.4);
  }
  to {
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .reaction,
  .reaction .emoji {
    transition: none;
  }

  .reaction.fresh,
  .reaction.fresh .emoji {
    animation: none;
  }
}

@container note (max-width: 380px) {
  .reaction {
    block-size: 30px;
    padding: 0 7px;
  }
}
</style>
