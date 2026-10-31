<script setup lang="ts">
import WidgetStack from "../widgets/WidgetStack.vue";

// The right-hand widget column, the Cinderfish counterpart of
// fork/ui/universal.widgets.vue. It only mounts the existing widget stack —
// ordering, editing and data fetching stay in the widgets module — and it
// disappears entirely below 1090px, where WidgetsDrawer takes over the same
// stack behind the floating button.
</script>

<template>
  <aside class="widget-panel">
    <WidgetStack />
  </aside>
</template>

<style scoped>
/* The column scrolls independently of the page so a tall stack never
   stretches the shell. */
.widget-panel {
  position: sticky;
  top: var(--space-24);
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  min-width: 0;
  max-height: calc(100dvh - var(--space-48));
  overflow-y: auto;
  overscroll-behavior: contain;
}

@media (max-width: 1090px) {
  .widget-panel {
    display: none;
  }
}
</style>
