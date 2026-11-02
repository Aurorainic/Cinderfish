<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import MkButton from "../../components/ui/MkButton.vue";
import { WIDGET_NAMES, WIDGETS, type WidgetName } from "../../widgets/registry";
import { addWidget, moveWidget, removeWidget, resetWidgets, widgetStack } from "../../stores/widgets";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

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
    <SettingsSection title="小组件顺序" persist-key="settings-widgets-order">
      <p class="settings-hint">
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
      <p v-else class="settings-hint">所有小组件都已添加。</p>
    </SettingsSection>

    <div class="settings-actions">
      <MkButton inline @click="resetWidgets">恢复默认</MkButton>
    </div>
  </SettingsPage>
</template>

<style scoped>
.rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-8) var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panelHighlight);
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
  color: var(--fgHighlighted);
  font-size: 0.9em;
  font-weight: 600;
  line-height: 1.35;
}

.row-desc {
  overflow: hidden;
  color: var(--fgTransparentWeak);
  font-size: 0.78em;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  flex: none;
  gap: 2px;
}

.row-button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--fgTransparentWeak);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

.row-button:hover:not(:disabled) {
  border-color: var(--divider);
  color: var(--accent);
}

.row-button.danger:hover:not(:disabled) {
  color: var(--error);
}

.row-button:focus-visible {
  outline: 2px solid var(--focus);
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
  margin: 0;
  padding: var(--space-16) var(--space-12);
  border: 1px dashed var(--divider);
  border-radius: var(--radius);
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  text-align: center;
}

.adder {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.adder-label {
  color: var(--fgTransparentWeak);
  font-size: 0.78em;
  font-weight: 600;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px var(--space-10);
  border: 1px dashed var(--divider);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--fgTransparentWeak);
  font-size: 0.8em;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.chip:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.chip:focus-visible {
  outline: 2px solid var(--focus);
  outline-offset: 2px;
}

.chip .icon {
  width: 12px;
  height: 12px;
}
</style>
