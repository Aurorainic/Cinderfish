<script setup lang="ts">
import { ref } from "vue";
import AppIcon from "../components/AppIcon.vue";
import DeckColumn from "../components/deck/DeckColumn.vue";
import DeckPageWindow from "../components/deck/DeckPageWindow.vue";
import { popupMenu } from "../os";
import { addColumn, COLUMN_META, COLUMN_TYPES, deckColumns } from "../stores/deck";
import type { MenuItem } from "../types/menu";

// Deck mode replaces the routed main area with a horizontal strip of
// independently scrolling columns. Adding a column opens the shared popup menu
// (os.popupMenu) instead of a locally drawn one: the local variant carried its
// own Escape and pointerdown listeners, which meant two menu lifecycles and two
// answers to the same key press.
const addButton = ref<HTMLElement | null>(null);

const ADD_ITEMS: MenuItem[] = COLUMN_TYPES.map((type) => ({
  type: "button" as const,
  text: `${COLUMN_META[type].label} — ${COLUMN_META[type].description}`,
  icon: COLUMN_META[type].icon,
  action: () => addColumn(type),
}));

function openAddMenu(): void {
  void popupMenu(ADD_ITEMS, addButton.value, { anchor: { x: "left", y: "top" } });
}
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

      <div class="deck-add">
        <button
          ref="addButton"
          class="add-button"
          type="button"
          title="添加列"
          aria-haspopup="menu"
          @click="openAddMenu"
        >
          <AppIcon name="plus" />
          <span>添加列</span>
        </button>
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
  padding: var(--space-12);
}

/* The strip is the only horizontal scroller: columns never widen the page, so
   the shell keeps its own width and the viewport never gets a second
   horizontal scrollbar. Below 840px one column owns the strip (DeckColumn sets
   its basis to 100%) and the snap points turn the strip into one page per
   column. */
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
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.add-button:hover,
.add-button:focus-visible {
  color: var(--accent);
  border-color: var(--accent);
}

.add-button:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

/* On phones the shell stacks and reserves room for the bottom navigation, so
   the strip only loses its width; the height still comes from the shell. */
@media (max-width: 500px) {
  .deck {
    padding: var(--space-8);
  }

  .deck-add {
    width: 96px;
  }
}
</style>
