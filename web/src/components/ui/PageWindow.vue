<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import AppIcon from "../AppIcon.vue";

// A modal page container: title bar + close button + default slot.
// In deck mode (task F5) this is the carrier for "open note/profile in a new
// page"; os.ts uses it for its overlay panels as well.
defineProps<{ title: string }>();

const emit = defineEmits<{ close: [] }>();

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") emit("close");
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="page-window-backdrop" @click.self="emit('close')">
    <section class="page-window" role="dialog" aria-modal="true">
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
  height: 80vh;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm, 0 4px 16px rgba(0, 0, 0, 0.3));
  overflow: hidden;
}

.page-window-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.page-window-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-strong);
}

.page-window-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--muted);
  cursor: pointer;
}

.page-window-close:hover {
  color: var(--text);
  background: var(--panel-2);
}

.page-window-close:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.page-window-body {
  flex: 1;
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
