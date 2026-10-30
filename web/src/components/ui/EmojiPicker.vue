<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useId, watch } from "vue";
import {
  EMOJI_CATEGORIES,
  searchEmojis,
  useRecentEmojis,
  type EmojiEntry,
} from "../../lib/emoji-catalog";
import AppIcon from "../AppIcon.vue";

// A Misskey-style emoji picker: search box, category tabs and a recents row on
// top. Arrow keys walk the grid, Enter picks, Escape closes. The grid is eight
// columns wide in every context so arrow navigation stays predictable.

const emit = defineEmits<{ select: [char: string]; close: [] }>();

const COLUMNS = 8;
const RECENT_TAB = "__recent__";

const { recent, push } = useRecentEmojis();

const query = ref("");
const tab = ref<string>(recent.value.length ? RECENT_TAB : EMOJI_CATEGORIES[0].name);
const activeIndex = ref(0);

const pickerId = useId();
const listboxId = `emoji-listbox-${pickerId}`;
const searchInput = ref<HTMLInputElement | null>(null);
const grid = ref<HTMLElement | null>(null);

const searching = computed(() => query.value.trim().length > 0);

const recentEntries = computed<EmojiEntry[]>(() =>
  recent.value.map((char) => ({ char, name: char })),
);

const list = computed<EmojiEntry[]>(() => {
  if (searching.value) return searchEmojis(query.value);
  if (tab.value === RECENT_TAB) return recentEntries.value;
  return EMOJI_CATEGORIES.find((category) => category.name === tab.value)?.emojis ?? [];
});

const activeId = computed(() => `${listboxId}-${activeIndex.value}`);

const emptyText = computed(() => (searching.value ? "没有匹配的 emoji" : "还没有用过的反应"));

watch([query, tab], () => {
  activeIndex.value = 0;
});

onMounted(async () => {
  await nextTick();
  searchInput.value?.focus();
});

function selectTab(name: string): void {
  tab.value = name;
  query.value = "";
  searchInput.value?.focus();
}

function choose(emoji: EmojiEntry | undefined): void {
  if (!emoji) return;
  push(emoji.char);
  emit("select", emoji.char);
}

function focusCell(index: number): void {
  const cell = grid.value?.querySelector<HTMLElement>(`[data-index="${index}"]`);
  cell?.focus();
  cell?.scrollIntoView({ block: "nearest", inline: "nearest" });
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") {
    event.preventDefault();
    event.stopPropagation();
    emit("close");
    return;
  }

  const total = list.value.length;
  let delta = 0;
  if (event.key === "ArrowRight") delta = 1;
  else if (event.key === "ArrowLeft") delta = -1;
  else if (event.key === "ArrowDown") delta = COLUMNS;
  else if (event.key === "ArrowUp") delta = -COLUMNS;
  else if (event.key === "Home") delta = -total;
  else if (event.key === "End") delta = total;
  else if (event.key === "Enter") {
    event.preventDefault();
    choose(list.value[activeIndex.value]);
    return;
  } else return;

  event.preventDefault();
  if (!total) return;

  const next = Math.min(total - 1, Math.max(0, activeIndex.value + delta));
  activeIndex.value = next;
  // Keep focus in the search box while typing; only chase the grid when the
  // user had already tabbed or clicked into it.
  if (document.activeElement !== searchInput.value) focusCell(next);
}
</script>

<template>
  <div class="picker" role="dialog" aria-label="选择反应" @keydown="onKeydown">
    <div class="picker-head">
      <AppIcon name="search" class="search-icon" />
      <input
        ref="searchInput"
        v-model="query"
        class="search"
        type="text"
        placeholder="搜索 emoji"
        role="combobox"
        aria-autocomplete="list"
        :aria-expanded="true"
        :aria-controls="listboxId"
        :aria-activedescendant="activeId"
        autocomplete="off"
        spellcheck="false"
      />
    </div>

    <div class="tabs" role="tablist" aria-label="emoji 分类">
      <button
        v-if="recent.length"
        type="button"
        role="tab"
        class="tab"
        :class="{ active: tab === RECENT_TAB && !searching }"
        :aria-selected="tab === RECENT_TAB && !searching"
        @click="selectTab(RECENT_TAB)"
      >
        最近
      </button>
      <button
        v-for="category in EMOJI_CATEGORIES"
        :key="category.name"
        type="button"
        role="tab"
        class="tab"
        :class="{ active: tab === category.name && !searching }"
        :aria-selected="tab === category.name && !searching"
        @click="selectTab(category.name)"
      >
        {{ category.name }}
      </button>
    </div>

    <div :id="listboxId" ref="grid" class="grid" role="listbox" :aria-label="searching ? '搜索结果' : 'emoji'">
      <button
        v-for="(emoji, index) in list"
        :key="emoji.char"
        type="button"
        role="option"
        class="cell"
        :class="{ active: index === activeIndex }"
        :data-index="index"
        :title="emoji.name"
        :aria-label="emoji.name"
        :aria-selected="index === activeIndex"
        :tabindex="index === activeIndex ? 0 : -1"
        @click="choose(emoji)"
        @focus="activeIndex = index"
      >
        <span class="char">{{ emoji.char }}</span>
      </button>
      <p v-if="!list.length" class="empty">{{ emptyText }}</p>
    </div>

    <p class="hint">↑↓←→ 选择 · Enter 确认 · Esc 关闭</p>
  </div>
</template>

<style scoped>
.picker {
  display: flex;
  flex-direction: column;
  width: 336px;
  max-width: calc(100vw - 16px);
  max-height: min(420px, calc(100vh - 16px));
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  color: var(--text);
}

.picker-head {
  position: relative;
  padding: 10px 10px 6px;
}

.search-icon {
  position: absolute;
  top: 50%;
  left: 20px;
  transform: translateY(-40%);
  color: var(--muted);
}

.search {
  width: 100%;
  padding: 8px 10px 8px 32px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-raised);
  color: var(--text);
  font: inherit;
  font-size: 0.85rem;
  outline: none;
}

.search:focus {
  border-color: var(--accent);
}

.tabs {
  display: flex;
  gap: 2px;
  padding: 0 10px 6px;
  overflow-x: auto;
  scrollbar-width: none;
}

.tabs::-webkit-scrollbar {
  display: none;
}

.tab {
  flex: none;
  padding: 3px 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 0.78rem;
  cursor: pointer;
}

.tab:hover {
  color: var(--text);
  background: var(--panel-2);
}

.tab.active {
  color: var(--on-accent);
  background: var(--accent);
  font-weight: 600;
}

.tab:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 2px;
  padding: 4px 8px 8px;
  overflow-y: auto;
  flex: 1;
}

.cell {
  display: grid;
  place-items: center;
  width: 100%;
  aspect-ratio: 1;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  line-height: 1;
}

.char {
  font-size: 1.25rem;
  line-height: 1;
}

.cell:hover {
  background: var(--panel-2);
}

.cell.active {
  background: var(--accent-soft);
  box-shadow: inset 0 0 0 1px var(--accent);
}

.cell:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.empty {
  grid-column: 1 / -1;
  margin: 12px 4px;
  color: var(--muted);
  font-size: 0.82rem;
  text-align: center;
}

.hint {
  margin: 0;
  padding: 6px 10px;
  border-top: 1px solid var(--divider);
  color: var(--muted);
  font-size: 0.72rem;
}

@media (prefers-reduced-motion: reduce) {
  .cell,
  .tab {
    transition: none;
  }
}
</style>
