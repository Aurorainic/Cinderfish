<script setup lang="ts">
import { computed, ref } from "vue";
import type { DriveFile } from "../../api/types";
import { isImageType } from "../../lib/mfm";

// NoteMedia takes every file of a note and renders only the images; the
// non-image attachments stay with NoteContent so a note with a single PDF
// does not reserve a media grid.
const props = defineProps<{ files: DriveFile[]; nested?: boolean }>();

const revealed = ref(false);

const images = computed(() => props.files.filter((file) => isImageType(file.type)));
const layout = computed(() => `media-${Math.min(images.value.length, 4)}`);
const hasSensitive = computed(() => images.value.some((file) => file.isSensitive));
const concealed = computed(() => hasSensitive.value && !revealed.value);
</script>

<template>
  <div v-if="images.length" class="media" :class="[layout, { nested, concealed }]">
    <a
      v-for="file in images"
      :key="file.id"
      :href="file.url"
      target="_blank"
      rel="noopener"
      class="item"
      :tabindex="concealed ? -1 : 0"
    >
      <img :src="file.thumbnailUrl || file.webpublicUrl || file.url" :alt="file.name" loading="lazy" />
    </a>
    <button v-if="concealed" type="button" class="veil" @click="revealed = true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
        <path d="M2 12s3.6-6.5 10-6.5S22 12 22 12s-3.6 6.5-10 6.5S2 12 2 12Z" />
        <circle cx="12" cy="12" r="2.6" />
        <path d="M4 20 20 4" />
      </svg>
      <span>敏感内容 · 点击显示</span>
    </button>
  </div>
</template>

<style scoped>
.media {
  position: relative;
  display: grid;
  gap: 4px;
  margin-top: 10px;
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.media-1 {
  grid-template-columns: 1fr;
}

.media-2,
.media-3,
.media-4 {
  grid-template-columns: 1fr 1fr;
}

.media-3 .item:first-child {
  grid-column: span 2;
}

.item {
  display: block;
  line-height: 0;
  background: var(--bg-raised);
}

.item img {
  width: 100%;
  max-height: 340px;
  object-fit: cover;
  display: block;
}

.media-1 .item img {
  max-height: 420px;
}

.concealed .item img {
  filter: blur(22px);
  transform: scale(1.04);
}

.veil {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px;
  border: 0;
  background: color-mix(in srgb, var(--bg) 55%, transparent);
  color: var(--text-strong);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.veil svg {
  width: 22px;
  height: 22px;
}

.veil:hover,
.veil:focus-visible {
  background: color-mix(in srgb, var(--bg) 40%, transparent);
}

.nested {
  margin-top: 8px;
}

.nested .item img {
  max-height: 180px;
}

@container note (max-width: 380px) {
  .item img {
    max-height: 240px;
  }

  .media-1 .item img {
    max-height: 280px;
  }
}
</style>
