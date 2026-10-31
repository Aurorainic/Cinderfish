<script setup lang="ts">
import AppIcon from "../AppIcon.vue";
import { toggleWidgetEditing, widgetEditing } from "../../stores/widgets";

// The one control both widget surfaces put in their header. The edit mode it
// toggles reads and orders the same stack, so the button has to come from the
// store rather than from local component state.
</script>

<template>
  <button
    class="edit-toggle"
    :class="{ active: widgetEditing }"
    type="button"
    :aria-pressed="widgetEditing"
    :title="widgetEditing ? '完成编辑' : '编辑小组件'"
    :aria-label="widgetEditing ? '完成编辑' : '编辑小组件'"
    @click="toggleWidgetEditing"
  >
    <AppIcon :name="widgetEditing ? 'check' : 'settings'" />
  </button>
</template>

<style scoped>
.edit-toggle {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--panel);
  color: var(--muted);
  cursor: pointer;
  transition: color var(--dur-fast) ease, border-color var(--dur-fast) ease,
    background var(--dur-fast) ease;
}

.edit-toggle:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.edit-toggle.active {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--on-accent);
}

.edit-toggle:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.edit-toggle .icon {
  width: 15px;
  height: 15px;
}
</style>
