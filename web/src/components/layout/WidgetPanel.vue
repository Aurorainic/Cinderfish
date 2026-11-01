<script setup lang="ts">
import WidgetEditToggle from "../widgets/WidgetEditToggle.vue";
import WidgetStack from "../widgets/WidgetStack.vue";

// The right-hand widget column, the Cinderfish counterpart of
// fork/ui/universal.widgets.vue. It owns the panel title and the actions row and
// leaves the ordered list to WidgetStack, so each widget card can carry its own
// title, actions and content. The panel disappears entirely below 1090px, where
// the WidgetsDrawer (mounted by AppShell) offers the same stack behind a
// floating fab. Both slots are replaceable when a surface wants a different
// chrome.
</script>

<template>
  <aside class="widget-panel" aria-labelledby="widget-panel-title">
    <header class="panel-head">
      <h2 id="widget-panel-title" class="panel-title">小组件</h2>
      <div class="panel-actions">
        <slot name="actions">
          <WidgetEditToggle />
        </slot>
      </div>
    </header>

    <div class="panel-body">
      <slot>
        <WidgetStack />
      </slot>
    </div>
  </aside>
</template>

<style scoped>
/* The column scrolls independently of the page so a tall stack never
   stretches the shell. It is the shell's last column: full height, a divider
   instead of a gap, and the same 24px bleed PageHeader expects from the main
   column is not needed here because no sticky header sits inside it. */
.widget-panel {
  position: sticky;
  top: 0;
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: var(--space-12);
  min-width: 0;
  height: 100dvh;
  max-height: 100dvh;
  padding: var(--space-12) var(--space-14) var(--space-16);
  border-left: 1px solid var(--divider);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.panel-head {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  padding: 0 2px;
}

.panel-title {
  color: var(--muted);
  font-size: 0.76rem;
  font-weight: 700;
  letter-spacing: 0.04em;
}

.panel-actions {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  margin-left: auto;
}

.panel-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

@media (max-width: 1090px) {
  .widget-panel {
    display: none;
  }
}
</style>
