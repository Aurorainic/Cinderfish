<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import AppIcon from "../components/AppIcon.vue";
import DeckColumn from "../components/deck/DeckColumn.vue";
import DeckPageWindow from "../components/deck/DeckPageWindow.vue";
import { addColumn, COLUMN_META, COLUMN_TYPES, deckColumns, type ColumnType } from "../stores/deck";

// Deck mode replaces the routed main area with a horizontal strip of
// independently scrolling columns. Adding a column is the one control that
// lives outside the columns themselves.
const adding = ref(false);
const addRoot = ref<HTMLElement | null>(null);

function choose(type: ColumnType): void {
  addColumn(type);
  adding.value = false;
}

function onDocumentPointerDown(event: PointerEvent): void {
  if (!adding.value) return;
  const el = addRoot.value;
  if (el && !el.contains(event.target as Node)) adding.value = false;
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") adding.value = false;
}

onMounted(() => {
  document.addEventListener("pointerdown", onDocumentPointerDown);
  window.addEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", onDocumentPointerDown);
  window.removeEventListener("keydown", onKeydown);
});
</script>

<template>
  <div class="deck">
    <div class="deck-strip">
      <p v-if="!deckColumns.length" class="deck-hint">还没有列。点击「添加列」开始布置你的 Deck。</p>

      <DeckColumn
        v-for="(column, index) in deckColumns"
        :key="column.id"
        :column="column"
        :index="index"
        :count="deckColumns.length"
      />

      <div ref="addRoot" class="deck-add">
        <button
          class="add-button"
          type="button"
          title="添加列"
          aria-haspopup="menu"
          :aria-expanded="adding"
          @click="adding = !adding"
        >
          <AppIcon name="plus" />
          <span>添加列</span>
        </button>

        <!-- Inside the click-outside root: a pointerdown on an item would
             otherwise close the picker before the click reached it. -->
        <div v-if="adding" class="add-menu" role="menu" aria-label="列类型">
          <button
            v-for="type in COLUMN_TYPES"
            :key="type"
            type="button"
            role="menuitem"
            class="add-item"
            @click="choose(type)"
          >
            <span class="add-item-icon"><AppIcon :name="COLUMN_META[type].icon" /></span>
            <span class="add-item-text">
              <span class="add-item-label">{{ COLUMN_META[type].label }}</span>
              <span class="add-item-desc">{{ COLUMN_META[type].description }}</span>
            </span>
          </button>
        </div>
      </div>
    </div>

    <DeckPageWindow />
  </div>
</template>

<style scoped>
/* The deck fills whatever the main area gives it: AppShell's `.main` is a flex
   column in deck mode, so the height chain is flex and no viewport arithmetic
   appears here. */
.deck {
  position: relative;
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  height: 100%;
}

/* The strip is the only horizontal scroller: columns never widen the page, so
   the shell keeps its own width and the viewport never gets a second
   horizontal scrollbar. */
.deck-strip {
  display: flex;
  align-items: stretch;
  gap: var(--space-12);
  flex: 1 1 auto;
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x mandatory;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
  -webkit-overflow-scrolling: touch;
}

.deck-hint {
  flex: 1 0 260px;
  align-self: center;
  color: var(--muted);
  text-align: center;
  font-size: 0.9rem;
}

.deck-add {
  flex: 0 0 auto;
  width: 132px;
  display: flex;
  scroll-snap-align: start;
}

.add-button {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed var(--border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.add-button:hover,
.add-button[aria-expanded="true"] {
  color: var(--accent);
  border-color: var(--accent);
}

.add-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

/* Fixed so the picker is never clipped by the strip's own scroller. */
.add-menu {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 60;
  width: 260px;
  max-width: calc(100vw - 32px);
  padding: 6px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}

.add-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.add-item:hover,
.add-item:focus-visible {
  background: var(--panel-2);
}

.add-item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.add-item-icon {
  display: inline-flex;
  color: var(--accent);
}

.add-item-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.add-item-label {
  font-size: 0.9rem;
  font-weight: 600;
}

.add-item-desc {
  font-size: 0.76rem;
  color: var(--muted);
}

/* On phones the shell stacks and reserves room for the bottom navigation, so
   the strip only loses its width; the height still comes from the shell. */
@media (max-width: 720px) {
  .deck-add {
    width: 96px;
  }

  .add-menu {
    right: var(--space-12);
    bottom: calc(var(--bottom-bar-height) + var(--space-20) + env(safe-area-inset-bottom));
  }
}
</style>
