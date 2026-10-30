<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { Note } from "../../api/types";
import { isImageType, renderMFM } from "../../lib/mfm";
import AppIcon from "../AppIcon.vue";
import NoteMedia from "./NoteMedia.vue";

const props = withDefaults(
  defineProps<{ note: Note; nested?: boolean; detail?: boolean }>(),
  { nested: false, detail: false },
);

const revealed = ref(false);
const expanded = ref(false);

const bodyHtml = computed(() => renderMFM(props.note.text));
const files = computed(() => props.note.files ?? []);
const attachments = computed(() => files.value.filter((file) => !isImageType(file.type)));
const text = computed(() => props.note.text ?? "");

// Long posts fold to a fixed height in feeds; the detail view and content
// warnings keep their full text.
const isLong = computed(() => {
  if (props.detail || props.note.cw) return false;
  return text.value.split("\n").length > 10 || text.value.length > 800 || files.value.length > 4;
});

const folded = computed(() => isLong.value && !expanded.value);

watch(
  () => props.note.id,
  () => {
    revealed.value = false;
    expanded.value = false;
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
      <p v-else class="cw-strip">内容警告：{{ note.cw }}</p>
    </div>

    <div class="fold" :class="{ collapsed: folded }">
      <div v-if="bodyHtml && (!note.cw || revealed)" class="body" v-html="bodyHtml"></div>

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
    </div>

    <button v-if="folded" type="button" class="show-more" @click="expanded = true">
      <AppIcon name="plus" />
      <span>展开全文</span>
    </button>
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

/* A long post is cut at 15em of text plus one media row, with the bottom
   fading out so the cut reads as "there is more" rather than a truncation. */
.fold.collapsed {
  position: relative;
  max-block-size: calc(15em + 100px);
  overflow: hidden;
  mask: linear-gradient(black calc(100% - 64px), transparent);
  -webkit-mask: linear-gradient(black calc(100% - 64px), transparent);
}

.show-more {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  padding: 2px 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--accent);
  font-size: 0.85rem;
  cursor: pointer;
}

.show-more:hover,
.show-more:focus-visible {
  background: var(--accent-soft);
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

@container note (max-width: 500px) {
  .body {
    line-height: 1.58;
  }

  .cw-banner,
  .cw-strip {
    font-size: 0.92em;
  }
}

@container note (max-width: 380px) {
  .body {
    line-height: 1.54;
  }

  .files a {
    font-size: 0.76rem;
  }

  .fold.collapsed {
    max-block-size: calc(15em + 72px);
  }
}
</style>
