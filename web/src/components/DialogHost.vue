// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/os.ts (toast host) and
// firefish@02c31e0:packages/client/src/components/MkToast.vue (adapted).
//
// Renders the toast queue from lib/os.ts. Firefish stacks toasts bottom-right;
// each is dismissed by click or after the timeout the queue applies. The
// dialog side of the queue is rendered by components/ui/DialogHost.vue, which
// root os.ts's OsOverlay mounts with the head of the queue.
<script setup lang="ts">
import { dismissToast, toasts } from "../lib/os";
</script>

<template>
  <Teleport to="body">
    <div class="toast-host" aria-live="polite">
      <TransitionGroup name="toast">
        <button
          v-for="item in toasts"
          :key="item.id"
          type="button"
          class="toast"
          @click="dismissToast(item.id)"
        >
          {{ item.message }}
        </button>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-host {
  position: fixed;
  inset-block-end: calc(env(safe-area-inset-bottom, 0px) + 16px);
  inset-inline-end: 16px;
  z-index: 3000000; /* above the high z band used by overlays */
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  pointer-events: none;
}

.toast {
  pointer-events: auto;
  max-width: min(420px, calc(100vw - 32px));
  padding: 10px 16px;
  font: inherit;
  font-size: 0.9em;
  text-align: start;
  color: var(--fg);
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: var(--radius);
  box-shadow: 0 4px 16px var(--shadow);
  cursor: pointer;
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
