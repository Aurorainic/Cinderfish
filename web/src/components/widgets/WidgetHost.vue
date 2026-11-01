// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/ui/universal.widgets.vue and
// firefish@02c31e0:packages/client/src/components/MkWidgets.vue.
//
// The widget column: an edit toggle that reveals the add picker, then the
// widget stack read from defaultStore.widgets. Adaptations:
//  - no VueDraggable: reordering uses the up/down buttons the ported widget
//    store already offers (no new npm dependencies).
//  - add/remove/update write back to defaultStore.widgets (Firefish-shaped
//    records: name/id/place/data) via the pizzax Storage `set`.
<script setup lang="ts">
import { computed, ref } from "vue";
import { defaultStore } from "../../store/firefish-store";
import { WIDGET_NAMES, WIDGETS, type WidgetName } from "../../widgets/registry";
import { getUniqueId } from "../../lib/os";

// TODO: Cinderfish has no MkAd / ads capability, so Firefish's MkAd slot is
// not ported.
interface WidgetRecord {
  name: WidgetName;
  id: string;
  place: string | null;
  data: Record<string, unknown>;
}

const editMode = ref(false);
const selected = ref<WidgetName | "">("");

const widgets = computed<WidgetRecord[]>(() =>
  (defaultStore.state.widgets as WidgetRecord[]).filter(
    (w): w is WidgetRecord => typeof w?.name === "string" && w.name in WIDGETS,
  ),
);

// Only widget types not already in the stack can be added; one instance each.
const available = computed(() =>
  WIDGET_NAMES.filter((name) => !widgets.value.some((w) => w.name === name)),
);

function setWidgets(next: WidgetRecord[]): void {
  defaultStore.set("widgets", next);
}

function addWidget(): void {
  if (!selected.value) return;
  const name = selected.value;
  selected.value = "";
  setWidgets([
    { name, id: getUniqueId(), place: null, data: { ...WIDGETS[name].defaultProps } },
    ...defaultStore.state.widgets as WidgetRecord[],
  ]);
}

function removeWidget(id: string): void {
  setWidgets((defaultStore.state.widgets as WidgetRecord[]).filter((w) => w.id !== id));
}

function moveWidget(index: number, delta: number): void {
  const next = widgets.value.slice();
  const target = index + delta;
  if (target < 0 || target >= next.length) return;
  const [item] = next.splice(index, 1);
  next.splice(target, 0, item);
  setWidgets(next);
}

function updateWidget(id: string, data: Record<string, unknown>): void {
  setWidgets(
    (defaultStore.state.widgets as WidgetRecord[]).map((w) =>
      w.id === id ? { ...w, data } : w,
    ),
  );
}
</script>

<template>
  <aside class="widgets" aria-label="小组件">
    <template v-if="editMode">
      <header class="editor">
        <select
          v-model="selected"
          class="editor-select"
          aria-label="选择要添加的小组件"
        >
          <option value="">{{ available.length ? "选择小组件…" : "已全部添加" }}</option>
          <option v-for="name in available" :key="name" :value="name">
            {{ WIDGETS[name].label }}
          </option>
        </select>
        <div class="editor-buttons">
          <button class="editor-button primary" type="button" :disabled="!selected" @click="addWidget">
            添加
          </button>
          <button class="editor-button" type="button" @click="editMode = false">关闭</button>
        </div>
      </header>
      <ul class="edit-list">
        <li v-for="(widget, index) in widgets" :key="widget.id" class="edit-item">
          <div class="edit-widget">
            <component
              :is="WIDGETS[widget.name].component"
              class="edit-preview"
              :widget-props="widget.data"
              @update-props="updateWidget(widget.id, $event)"
            />
          </div>
          <div class="edit-actions">
            <button
              class="edit-button"
              type="button"
              title="上移"
              aria-label="上移"
              :disabled="index === 0"
              @click="moveWidget(index, -1)"
            >
              ↑
            </button>
            <button
              class="edit-button"
              type="button"
              title="下移"
              aria-label="下移"
              :disabled="index >= widgets.length - 1"
              @click="moveWidget(index, 1)"
            >
              ↓
            </button>
            <button
              class="edit-button remove"
              type="button"
              title="删除"
              aria-label="删除"
              @click="removeWidget(widget.id)"
            >
              ✕
            </button>
          </div>
        </li>
      </ul>
      <button v-if="editMode" class="done" type="button" @click="editMode = false">✓ 完成编辑</button>
    </template>
    <template v-else>
      <component
        :is="WIDGETS[widget.name].component"
        v-for="widget in widgets"
        :key="widget.id"
        class="widget"
        :widget-props="widget.data"
        @update-props="updateWidget(widget.id, $event)"
      />
      <button class="done" type="button" @click="editMode = true">✏ 编辑小组件</button>
    </template>
  </aside>
</template>

<style scoped>
/* Ported from firefish@02c31e0:packages/client/src/ui/universal.widgets.vue. */
.widgets {
  height: min-content;
  min-height: 100vh;
  padding: var(--margin) 0;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.widgets > :deep(*) {
  margin-block-start: var(--margin);
  width: 300px;
}

.widgets > :deep(*:first-child) {
  margin-block-start: 0;
}

.editor {
  margin: 16px 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.editor-select {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm, 6px);
  background: var(--panel);
  color: var(--fg);
  font: inherit;
  font-size: 0.9em;
}

.editor-buttons {
  display: flex;
  gap: 8px;
}

.editor-button {
  flex: 1;
  padding: 7px 10px;
  font: inherit;
  font-size: 0.9em;
  color: var(--fg);
  background: var(--panel);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm, 6px);
  cursor: pointer;
}

.editor-button.primary {
  color: var(--fgOnAccent);
  background: var(--accent);
  border-color: transparent;
}

.editor-button:disabled {
  opacity: 0.5;
  cursor: default;
}

.edit-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.edit-item {
  position: relative;
  cursor: move;
}

.edit-preview {
  pointer-events: none;
}

.edit-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 8px;
  z-index: 10;
}

.edit-button {
  width: 32px;
  height: 32px;
  color: #fff;
  background: rgba(0, 0, 0, 0.7);
  border: 0;
  border-radius: 4px;
  cursor: pointer;
  font: inherit;
}

.done {
  margin: 0 auto;
  padding: 6px 10px;
  font: inherit;
  font-size: 0.9em;
  color: var(--accent);
  background: transparent;
  border: 0;
  cursor: pointer;
}
</style>
