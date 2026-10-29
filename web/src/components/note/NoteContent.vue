<script setup lang="ts">
import { computed, ref } from "vue";
import type { Note } from "../../api/types";
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

    <div v-if="sortedReactions.length" class="reactions">
      <button
        v-for="[emoji, count] in sortedReactions"
        :key="emoji"
        type="button"
        class="reaction"
        :class="{ mine: myReaction === emoji }"
        :title="`反应 ${emoji}`"
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
  border-radius: 999px;
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

.reactions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 10px;
}

.reaction {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 9px;
  border: 1px solid var(--divider);
  border-radius: 999px;
  background: transparent;
  color: var(--text);
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1.5;
}

.reaction:hover,
.reaction:focus-visible {
  border-color: var(--accent);
}

.reaction.mine {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.count {
  font-size: 0.76rem;
  color: var(--muted);
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
    padding: 1px 7px;
  }
}
</style>
