<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import AppIcon from "../AppIcon.vue";
import WidgetEditToggle from "./WidgetEditToggle.vue";
import WidgetStack from "./WidgetStack.vue";

// Below 1090px the shell hides the widget column, so the same stack moves into
// a right-hand drawer behind a floating button. Above 1090px the button is
// hidden and the drawer is unreachable; a resize past the breakpoint closes it
// so an open panel can never sit on top of the restored column.
const open = ref(false);
const trigger = ref<HTMLElement | null>(null);
const panel = ref<HTMLElement | null>(null);
const route = useRoute();

let desktop: MediaQueryList | null = null;

function close(restoreFocus = true): void {
  if (!open.value) return;
  open.value = false;
  if (restoreFocus) trigger.value?.focus();
}

async function openDrawer(): Promise<void> {
  open.value = true;
  await nextTick();
  panel.value?.focus();
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") close();
}

function onViewportChange(event: MediaQueryListEvent): void {
  if (event.matches) close();
}

// Navigation inside the drawer (a link in the about widget, a note in the
// timeline) should reveal the page instead of staying behind the panel. Focus
// is not returned to the button there: the reader is now on another page.
watch(
  () => route.fullPath,
  () => close(false),
);

watch(open, (value) => {
  document.body.style.overflow = value ? "hidden" : "";
});

onMounted(() => {
  document.addEventListener("keydown", onKeydown);
  desktop = window.matchMedia("(min-width: 1091px)");
  desktop.addEventListener("change", onViewportChange);
});

onBeforeUnmount(() => {
  document.removeEventListener("keydown", onKeydown);
  desktop?.removeEventListener("change", onViewportChange);
  desktop = null;
  document.body.style.overflow = "";
});
</script>

<template>
  <button
    ref="trigger"
    class="widget-fab"
    type="button"
    title="小组件"
    aria-haspopup="dialog"
    :aria-expanded="open"
    @click="openDrawer"
  >
    <AppIcon name="settings" />
    <span class="fab-label">小组件</span>
  </button>

  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="widget-drawer">
        <div class="backdrop" @click="close()" />
        <aside
          ref="panel"
          class="panel"
          role="dialog"
          aria-modal="true"
          aria-label="小组件"
          tabindex="-1"
        >
          <header class="panel-head">
            <h2>小组件</h2>
            <div class="panel-actions">
              <WidgetEditToggle />
              <button class="panel-close" type="button" title="关闭" aria-label="关闭" @click="close()">
                <AppIcon name="close" />
              </button>
            </div>
          </header>
          <div class="panel-body">
            <WidgetStack />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.widget-fab {
  position: fixed;
  z-index: 40;
  inset-block-end: 24px;
  inset-inline-end: 20px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-raised);
  box-shadow: var(--shadow-sm);
  color: var(--text);
  font-size: 0.82rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.widget-fab:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.widget-fab .icon {
  width: 16px;
  height: 16px;
}

.widget-drawer {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  justify-content: flex-end;
}

.backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
}

.panel {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(320px, 100vw - 48px);
  height: 100dvh;
  background: var(--bg-raised);
  border-left: 1px solid var(--border);
  outline: none;
}

.panel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--divider);
}

.panel-head h2 {
  font-size: 1rem;
}

.panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.panel-close {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.panel-close:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.panel-close .icon {
  width: 15px;
  height: 15px;
}

.panel-body {
  flex: 1;
  min-height: 0;
  padding: 14px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.2s ease;
}

.drawer-enter-active .panel,
.drawer-leave-active .panel {
  transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .panel,
.drawer-leave-to .panel {
  transform: translateX(100%);
}

@media (max-width: 500px) {
  .widget-fab {
    inset-block-end: calc(72px + env(safe-area-inset-bottom) + 12px);
    inset-inline-end: 12px;
    gap: 0;
    width: 42px;
    height: 42px;
    padding: 0;
    justify-content: center;
  }

  .fab-label {
    display: none;
  }
}

/* The universal shell shows the real column from here up, and deck mode hides
   it altogether; either way the floating entry has no job. */
@media (min-width: 1091px) {
  .widget-fab {
    display: none;
  }
}
</style>
