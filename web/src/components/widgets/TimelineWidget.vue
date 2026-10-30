<script setup lang="ts">
import { computed } from "vue";
import TimelineFeed from "../TimelineFeed.vue";
import WidgetCard from "./WidgetCard.vue";

type TimelineSource = "home" | "local" | "hybrid" | "global";

const props = defineProps<{ widgetProps?: Record<string, unknown> }>();
const emit = defineEmits<{ (event: "update-props", patch: Record<string, unknown>): void }>();

const SOURCES: { id: TimelineSource; label: string }[] = [
  { id: "home", label: "首页" },
  { id: "local", label: "本地" },
  { id: "hybrid", label: "社交" },
  { id: "global", label: "全球" },
];

const HEIGHTS = [240, 360, 520];
const DEFAULT_HEIGHT = 360;
const MIN_HEIGHT = 200;
const MAX_HEIGHT = 800;

function isSource(value: unknown): value is TimelineSource {
  return SOURCES.some((source) => source.id === value);
}

const source = computed<TimelineSource>(() => {
  const value = props.widgetProps?.source;
  return isSource(value) ? value : "local";
});

const height = computed(() => {
  const value = props.widgetProps?.height;
  if (typeof value !== "number" || !Number.isFinite(value)) return DEFAULT_HEIGHT;
  return Math.min(Math.max(Math.round(value), MIN_HEIGHT), MAX_HEIGHT);
});

// A stored height outside the presets (an older record) still needs an option,
// or the select would render blank.
const heightOptions = computed(() =>
  HEIGHTS.includes(height.value) ? HEIGHTS : [...HEIGHTS, height.value].sort((a, b) => a - b),
);

function choose(id: TimelineSource): void {
  if (id === source.value) return;
  emit("update-props", { source: id });
}

function resize(event: Event): void {
  const value = Number((event.target as HTMLSelectElement).value);
  if (!Number.isFinite(value)) return;
  emit("update-props", { height: value });
}
</script>

<template>
  <WidgetCard title="时间线" icon="home" flush>
    <div class="toolbar">
      <nav class="segments" aria-label="时间线来源">
        <button
          v-for="item in SOURCES"
          :key="item.id"
          type="button"
          class="segment"
          :class="{ active: source === item.id }"
          :aria-pressed="source === item.id"
          @click="choose(item.id)"
        >
          {{ item.label }}
        </button>
      </nav>
      <select class="height-select" :value="height" aria-label="小组件高度" @change="resize">
        <option v-for="option in heightOptions" :key="option" :value="option">{{ option }}px</option>
      </select>
    </div>

    <div class="scroll" :style="{ height: `${height}px` }">
      <TimelineFeed :key="source" :source="source" :composer="false" />
    </div>
  </WidgetCard>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--divider);
}

.segments {
  display: flex;
  flex: 1;
  min-width: 0;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-raised);
  overflow-x: auto;
  scrollbar-width: none;
}

.segments::-webkit-scrollbar {
  display: none;
}

.segment {
  flex: 1 0 auto;
  padding: 4px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 0.76rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.segment:hover {
  color: var(--text);
}

.segment.active {
  background: var(--accent);
  color: var(--on-accent);
}

.height-select {
  flex: none;
  padding: 3px 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-raised);
  color: var(--muted);
  font-size: 0.72rem;
  cursor: pointer;
}

.scroll {
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}
</style>
