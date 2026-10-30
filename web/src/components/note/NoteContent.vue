<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { Note } from "../../api/types";
import { emojiName } from "../../lib/emoji-catalog";
import { isImageType, renderMFM } from "../../lib/mfm";
import AppIcon from "../AppIcon.vue";
import NoteMedia from "./NoteMedia.vue";

const props = withDefaults(
  defineProps<{
    note: Note;
    nested?: boolean;
    reactions: Record<string, number>;
    myReaction?: string;
  }>(),
  { nested: false, myReaction: undefined },
);

const emit = defineEmits<{ react: [emoji: string] }>();

const revealed = ref(false);
const bodyHtml = computed(() => renderMFM(props.note.text));
const files = computed(() => props.note.files ?? []);
const attachments = computed(() => files.value.filter((file) => !isImageType(file.type)));
const sortedReactions = computed(() =>
  Object.entries(props.reactions)
    .filter(([, count]) => count > 0)
    .sort((a, b) => b[1] - a[1]),
);

// Reactions that were not on screen when the bar first mounted animate in;
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
  <div class="content" :class="{ nested }">
    <div v-if="note.cw" class="cw">
      <button v-if="!revealed" type="button" class="cw-banner" @click="revealed = true">
        <span class="cw-label">内容警告：{{ note.cw }}</span>
        <span class="cw-hint">点击显示</span>
      </button>
      <template v-else>
        <p class="cw-strip">内容警告：{{ note.cw }}</p>
        <div v-if="bodyHtml" class="body" v-html="bodyHtml"></div>
      </template>
    </div>
    <div v-else-if="bodyHtml" class="body" v-html="bodyHtml"></div>

    <NoteMedia v-if="files.length" :files="files" :nested="nested" />

    <ul v-if="attachments.length" class="files">
      <li v-for="file in attachments" :key="file.id">
        <a :href="file.url" target="_blank" rel="noopener">
          <AppIcon name="image" />
          <span>{{ file.name }}</span>
        </a>
      </li>
    </ul>

    <div v-if="note.hasPoll && note.poll" class="poll">
      <div v-for="(choice, i) in note.poll.choices" :key="i" class="poll-choice">
        <span>{{ choice.text }}</span>
        <span class="muted small">{{ choice.votes }}</span>
      </div>
      <p class="muted small">{{ note.poll.votesCount }} 票</p>
    </div>

    <slot name="quote"></slot>

    <div v-if="sortedReactions.length" class="reactions" role="group" aria-label="反应">
      <button
        v-for="[emoji, count] in sortedReactions"
        :key="emoji"
        type="button"
        class="reaction"
        :class="{ mine: myReaction === emoji, fresh: fresh.has(emoji) }"
        :title="`反应 ${emojiName(emoji)}`"
        :aria-label="`${emojiName(emoji)}，${count} 个反应${myReaction === emoji ? '，已选择' : ''}`"
        :aria-pressed="myReaction === emoji"
        @click="emit('react', emoji)"
      >
        <span class="emoji">{{ emoji }}</span>
        <span class="count">{{ count }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.content {
  min-width: 0;
}

.body {
  margin-top: 6px;
  word-break: break-word;
  white-space: normal;
  line-height: 1.62;
}

.body :deep(.mfm-link),
.body :deep(.mfm-mention),
.body :deep(.mfm-hashtag) {
  color: var(--link);
}

.body :deep(.mfm-code) {
  background: var(--bg);
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  overflow-x: auto;
  font-size: 0.85rem;
}

.body :deep(.mfm-inline-code) {
  background: var(--bg);
  padding: 1px 5px;
  border-radius: 5px;
  font-size: 0.88em;
}

.cw {
  margin-top: 8px;
}

.cw-banner {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: var(--cw-bg);
  color: var(--cw-fg);
  cursor: pointer;
  text-align: left;
}

.cw-label {
  font-weight: 600;
}

.cw-hint {
  flex: none;
  font-size: 0.8rem;
  opacity: 0.7;
}

.cw-banner:hover .cw-hint,
.cw-banner:focus-visible .cw-hint {
  opacity: 1;
}

.cw-strip {
  padding: 6px 12px;
  margin-bottom: 6px;
  border-left: 2px solid var(--cw-fg);
  background: var(--cw-bg);
  color: var(--cw-fg);
  font-size: 0.85rem;
  font-weight: 600;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.files {
  list-style: none;
  padding: 0;
  margin: 10px 0 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.files a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  font-size: 0.8rem;
  color: var(--muted);
  max-width: 100%;
}

.files a:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.files span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.poll {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.poll-choice {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 12px;
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
}

/* Reaction pills follow Firefish's MkReactionsViewer.reaction: 32px tall,
   1.5em emoji, one's own reaction pulled to the front with a solid accent
   fill and an inset hairline. */
.reactions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
}

.reaction {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  block-size: 32px;
  padding: 0 9px;
  border: 0;
  border-radius: 6px;
  background: color-mix(in srgb, var(--text) 7%, transparent);
  color: var(--muted);
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1;
  transition: background 0.15s ease, color 0.15s ease;
}

.reaction:hover,
.reaction:focus-visible {
  background: color-mix(in srgb, var(--text) 12%, transparent);
  color: var(--text);
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
  box-shadow: inset 0 0 0 1px var(--accent-strong);
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

@container note (max-width: 500px) {
  .body {
    font-size: 0.94em;
    line-height: 1.58;
  }
}

@container note (max-width: 450px) {
  .body {
    font-size: 0.92em;
  }

  .reaction {
    block-size: 30px;
    padding: 0 7px;
  }
}
</style>
