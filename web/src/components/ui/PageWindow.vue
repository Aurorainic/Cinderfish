<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import AppIcon from "../AppIcon.vue";

// A modal page container: title bar + close button + default slot.
// In deck mode (task F5) this is the carrier for "open note/profile in a new
// page"; os.ts uses it for its overlay panels as well.
//
// Firefish draws these as a panel window: a --panelHeaderBg bar with a hairline
// under it and a body that is as tall as its content. `compact` is the post
// form's narrower window (MkPostForm in _window_: 520px, content height).
defineProps<{ title: string; compact?: boolean }>();

const emit = defineEmits<{ close: [] }>();

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") emit("close");
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="page-window-backdrop" @click.self="emit('close')">
    <section class="page-window" :class="{ compact }" role="dialog" aria-modal="true">
      <header class="page-window-header">
        <h2 class="page-window-title">{{ title }}</h2>
        <button
          type="button"
          class="page-window-close"
          aria-label="关闭"
          @click="emit('close')"
        >
          <AppIcon name="close" />
        </button>
      </header>
      <div class="page-window-body">
        <slot />
      </div>
    </section>
  </div>
</template>

<style scoped>
.page-window-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
}

.page-window {
  display: flex;
  flex-direction: column;
  width: 720px;
  max-width: calc(100vw - 32px);
  /* Content height, capped: Firefish's windows size to their body and only
     scroll once the viewport is the limit. */
  max-height: min(80vh, 720px);
  background: var(--panel);
  border: solid 0.5px var(--divider);
  border-radius: var(--radius);
  box-shadow: 0 8px 32px var(--modalBg);
  overflow: hidden;
}

.page-window.compact {
  width: 520px;
}

.page-window-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  background: var(--panelHeaderBg);
  border-bottom: solid 0.5px var(--panelHeaderDivider, var(--divider));
  flex-shrink: 0;
}

.page-window-title {
  margin: 0;
  font-size: 0.95em;
  font-weight: 700;
  color: var(--fg);
}

.page-window-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px;
  background: none;
  border: none;
  border-radius: var(--radius);
  color: var(--fgTransparentWeak);
  cursor: pointer;
}

.page-window-close:hover {
  color: var(--fgHighlighted);
  background: var(--panelHighlight);
}

.page-window-close:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.page-window-body {
  flex: 1 1 auto;
  min-block-size: 0;
  overflow: auto;
}

@media (max-width: 760px) {
  .page-window-backdrop {
    align-items: stretch;
  }

  .page-window {
    width: 100%;
    max-width: none;
    height: 100vh;
    height: 100dvh;
    border: none;
    border-radius: 0;
  }
}
</style>
