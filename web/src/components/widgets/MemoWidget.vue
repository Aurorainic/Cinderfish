<script setup lang="ts">
import { onBeforeUnmount, ref } from "vue";
import AppIcon from "../AppIcon.vue";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ widgetProps?: Record<string, unknown> }>();

// A scratchpad that never leaves the browser: it lives in its own storage key
// rather than in the widget props so removing and re-adding the widget keeps
// the text.
const STORAGE_KEY = "cinderfish.ui.memo";
const SAVE_DELAY_MS = 600;

function readMemo(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? "";
  } catch {
    return "";
  }
}

function writeMemo(value: string): void {
  try {
    localStorage.setItem(STORAGE_KEY, value);
  } catch {
    // Storage may be unavailable; the text simply does not persist.
  }
}

const text = ref(readMemo());
const saved = ref(true);
let timer: ReturnType<typeof setTimeout> | null = null;

function clearTimer(): void {
  if (timer) {
    clearTimeout(timer);
    timer = null;
  }
}

function save(): void {
  clearTimer();
  writeMemo(text.value);
  saved.value = true;
}

function scheduleSave(): void {
  saved.value = false;
  clearTimer();
  timer = setTimeout(save, SAVE_DELAY_MS);
}

onBeforeUnmount(() => {
  if (!saved.value) save();
});
</script>

<template>
  <WidgetCard title="便签" icon="feather" hint="只保存在本机">
    <template #actions>
      <button
        class="save"
        type="button"
        :disabled="saved"
        :title="saved ? '已保存' : '保存'"
        :aria-label="saved ? '已保存' : '保存'"
        @click="save"
      >
        <AppIcon name="check" />
      </button>
    </template>

    <textarea
      v-model="text"
      class="memo"
      rows="4"
      placeholder="记点什么…"
      aria-label="便签内容"
      @input="scheduleSave"
      @blur="save"
    />
  </WidgetCard>
</template>

<style scoped>
.save {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: transparent;
  color: var(--accent);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

.save:hover:not(:disabled) {
  border-color: var(--accent);
}

.save:disabled {
  color: var(--muted);
  opacity: 0.6;
  cursor: default;
}

.save .icon {
  width: 13px;
  height: 13px;
}

.memo {
  display: block;
  width: 100%;
  min-height: 92px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-raised);
  color: var(--text);
  font: inherit;
  font-size: 0.86rem;
  line-height: 1.65;
  outline: none;
  resize: vertical;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.memo:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
</style>
