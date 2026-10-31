<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import AppIcon from "../AppIcon.vue";
import {
  addWidget,
  moveWidget,
  removeWidget,
  resetWidgets,
  setWidgetProps,
  widgetEditing,
  widgetStack,
} from "../../stores/widgets";
import { WIDGET_NAMES, WIDGETS, type WidgetName } from "../../widgets/registry";

// The ordered list every widget surface renders: the right column and the
// narrow-screen drawer show the same stack, so the order and props a reader sets
// in one show up in the other. Ordering has no drag library here; the edit list
// moves items with explicit up/down buttons. The title and the edit toggle live
// in the surface header (WidgetPanel, WidgetsDrawer), which is why this
// component starts at the list.

const editing = widgetEditing;
const selected = ref<WidgetName | "">("");

// Below 1090px the shell hides the whole column with display:none, but a
// mounted component keeps running: its feeds would fetch and subscribe with
// nothing on screen. A hidden subtree has no layout, so a zero-width reading
// is the signal to stop rendering the widgets. The drawer's own stack sits in
// a laid-out panel and always measures.
const root = ref<HTMLElement | null>(null);
const laidOut = ref(false);

function measure(): void {
  laidOut.value = (root.value?.offsetWidth ?? 0) > 0;
}

onMounted(() => {
  measure();
  window.addEventListener("resize", measure);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", measure);
});

// Leaving edit mode while the drawer is closed should not keep a picker open;
// the selection is only meaningful inside the edit list.
watch(editing, (value) => {
  if (!value) selected.value = "";
});

// Only widgets that are not in the stack yet can be added; a stack holds one
// instance per widget.
const available = computed(() =>
  WIDGET_NAMES.filter((name) => !widgetStack.value.some((widget) => widget.name === name)),
);

function onSelect(event: Event): void {
  const value = (event.target as HTMLSelectElement).value;
  selected.value = WIDGET_NAMES.find((name) => name === value) ?? "";
}

function add(): void {
  if (!selected.value) return;
  addWidget(selected.value);
  selected.value = "";
}

function move(id: string, delta: number): void {
  const index = widgetStack.value.findIndex((widget) => widget.id === id);
  if (index < 0) return;
  moveWidget(id, index + delta);
}

function updateProps(id: string, patch: Record<string, unknown>): void {
  setWidgetProps(id, patch);
}
</script>

<template>
  <section ref="root" class="widget-stack" :class="{ editing }">
    <template v-if="editing">
      <div class="adder">
        <select
          class="adder-select"
          :value="selected"
          :disabled="!available.length"
          aria-label="选择要添加的小组件"
          @change="onSelect"
        >
          <option value="">{{ available.length ? "选择小组件…" : "已全部添加" }}</option>
          <option v-for="name in available" :key="name" :value="name">
            {{ WIDGETS[name].label }}
          </option>
        </select>
        <button class="button small" type="button" :disabled="!selected" @click="add">添加</button>
      </div>

      <p v-if="!widgetStack.length" class="stack-empty">还没有小组件，从上面的列表里添加。</p>

      <ul class="edit-list">
        <li v-for="(widget, index) in widgetStack" :key="widget.id" class="edit-item">
          <span class="edit-icon"><AppIcon :name="WIDGETS[widget.name].icon" /></span>
          <span class="edit-text">
            <span class="edit-label">{{ WIDGETS[widget.name].label }}</span>
            <span class="edit-desc">{{ WIDGETS[widget.name].description }}</span>
          </span>
          <span class="edit-actions">
            <button
              class="row-button"
              type="button"
              title="上移"
              aria-label="上移"
              :disabled="index === 0"
              @click="move(widget.id, -1)"
            >
              <span class="arrow up"><AppIcon name="arrowLeft" /></span>
            </button>
            <button
              class="row-button"
              type="button"
              title="下移"
              aria-label="下移"
              :disabled="index >= widgetStack.length - 1"
              @click="move(widget.id, 1)"
            >
              <span class="arrow down"><AppIcon name="arrowLeft" /></span>
            </button>
            <button
              class="row-button danger"
              type="button"
              title="删除"
              aria-label="删除"
              @click="removeWidget(widget.id)"
            >
              <AppIcon name="close" />
            </button>
          </span>
        </li>
      </ul>

      <button class="button ghost small reset" type="button" @click="resetWidgets">恢复默认</button>
      <RouterLink to="/settings/widgets" class="manage">在设置中管理小组件</RouterLink>
    </template>

    <div v-else-if="laidOut" class="stack-list">
      <component
        :is="WIDGETS[widget.name].component"
        v-for="widget in widgetStack"
        :key="widget.id"
        :widget-props="widget.props"
        @update-props="updateProps(widget.id, $event)"
      />
      <p v-if="!widgetStack.length" class="stack-empty">还没有小组件，点击齿轮添加。</p>
    </div>
  </section>
</template>

<style scoped>
.widget-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.stack-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.stack-empty {
  padding: 18px 12px;
  border: 1px dashed var(--border);
  border-radius: var(--radius-lg);
  color: var(--muted);
  font-size: 0.84rem;
  text-align: center;
}

.adder {
  display: flex;
  gap: 6px;
}

.adder-select {
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-raised);
  color: var(--text);
  font-size: 0.82rem;
  cursor: pointer;
}

.adder-select:disabled {
  color: var(--muted);
  cursor: default;
}

.edit-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.edit-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel);
}

.edit-icon {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--accent);
}

.edit-icon .icon {
  width: 15px;
  height: 15px;
}

.edit-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.edit-label {
  color: var(--text-strong);
  font-size: 0.85rem;
  font-weight: 600;
  line-height: 1.35;
}

.edit-desc {
  color: var(--muted);
  font-size: 0.72rem;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.edit-actions {
  display: flex;
  flex: none;
  gap: 2px;
}

.row-button {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

.row-button:hover:not(:disabled) {
  border-color: var(--border);
  color: var(--accent);
}

.row-button.danger:hover:not(:disabled) {
  color: var(--danger);
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

.reset {
  align-self: flex-start;
}

.manage {
  align-self: flex-start;
  color: var(--accent);
  font-size: 0.76rem;
  font-weight: 600;
}

.manage:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}
</style>
