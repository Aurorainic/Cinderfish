<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import { WIDGET_NAMES, WIDGETS, type WidgetName } from "../../widgets/registry";
import { addWidget, moveWidget, removeWidget, resetWidgets, widgetStack } from "../../stores/widgets";
import SettingsPage from "./SettingsPage.vue";

// The widget column is an ordered list of one instance per widget: reorder here
// with explicit buttons, add a missing one from the picker below the list.
const available = computed(() =>
  WIDGET_NAMES.filter((name) => !widgetStack.value.some((widget) => widget.name === name)),
);

function move(id: string, delta: number): void {
  const index = widgetStack.value.findIndex((widget) => widget.id === id);
  if (index < 0) return;
  moveWidget(id, index + delta);
}

function add(name: WidgetName): void {
  addWidget(name);
}
</script>

<template>
  <SettingsPage>
    <section class="card">
      <h2 class="card-title">小组件</h2>
      <p class="muted small">
        小工具列按这里的顺序从上往下排列，只保存在这台设备上。窄屏时它们收进右侧的小组件抽屉。
      </p>

      <ul v-if="widgetStack.length" class="rows">
        <li v-for="(widget, index) in widgetStack" :key="widget.id" class="row">
          <span class="row-icon"><AppIcon :name="WIDGETS[widget.name].icon" /></span>
          <span class="row-text">
            <span class="row-label">{{ WIDGETS[widget.name].label }}</span>
            <span class="row-desc">{{ WIDGETS[widget.name].description }}</span>
          </span>
          <span class="row-actions">
            <button
              type="button"
              class="row-button"
              title="上移"
              aria-label="上移"
              :disabled="index === 0"
              @click="move(widget.id, -1)"
            >
              <span class="arrow up"><AppIcon name="arrowLeft" /></span>
            </button>
            <button
              type="button"
              class="row-button"
              title="下移"
              aria-label="下移"
              :disabled="index >= widgetStack.length - 1"
              @click="move(widget.id, 1)"
            >
              <span class="arrow down"><AppIcon name="arrowLeft" /></span>
            </button>
            <button
              type="button"
              class="row-button danger"
              title="移除"
              aria-label="移除"
              @click="removeWidget(widget.id)"
            >
              <AppIcon name="close" />
            </button>
          </span>
        </li>
      </ul>

      <p v-else class="empty">还没有小组件，从下面的列表里添加。</p>

      <div v-if="available.length" class="adder">
        <span class="adder-label">可以添加</span>
        <button
          v-for="name in available"
          :key="name"
          type="button"
          class="chip"
          :title="WIDGETS[name].description"
          @click="add(name)"
        >
          <AppIcon name="plus" />
          {{ WIDGETS[name].label }}
        </button>
      </div>
      <p v-else class="muted small adder-note">所有小组件都已添加。</p>

      <button type="button" class="button ghost small reset" @click="resetWidgets">恢复默认</button>
    </section>
  </SettingsPage>
</template>

<style scoped>
.rows {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  margin: var(--space-12) 0 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-8) var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel-2);
}

.row-icon {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--accent);
}

.row-icon .icon {
  width: 16px;
  height: 16px;
}

.row-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.row-label {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  line-height: 1.35;
}

.row-desc {
  overflow: hidden;
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  flex: none;
  gap: var(--space-2);
}

.row-button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-weak);
  cursor: pointer;
  transition: color var(--dur-base) ease, border-color var(--dur-base) ease,
    opacity var(--dur-base) ease;
}

.row-button:hover:not(:disabled) {
  border-color: var(--border);
  color: var(--accent);
}

.row-button.danger:hover:not(:disabled) {
  color: var(--danger);
}

.row-button:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.row-button:disabled {
  opacity: 0.35;
  cursor: default;
}

.row-button .icon {
  width: 13px;
  height: 13px;
}

.arrow {
  display: grid;
  place-items: center;
}

.arrow.up {
  transform: rotate(90deg);
}

.arrow.down {
  transform: rotate(-90deg);
}

.empty {
  margin-top: var(--space-12);
  padding: var(--space-16) var(--space-12);
  border: 1px dashed var(--border);
  border-radius: var(--radius-md);
  color: var(--text-weak);
  font-size: var(--fs-sm);
  text-align: center;
}

.adder {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  flex-wrap: wrap;
  margin-top: var(--space-12);
  padding-top: var(--space-12);
  border-top: 1px solid var(--divider);
}

.adder-label {
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-semibold);
}

.adder-note {
  margin-top: var(--space-12);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-10);
  border: 1px dashed var(--border);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-weak);
  font-size: var(--fs-xs);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: color var(--dur-base) ease, border-color var(--dur-base) ease;
}

.chip:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.chip:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.chip .icon {
  width: 12px;
  height: 12px;
}

.reset {
  margin-top: var(--space-16);
}
</style>
