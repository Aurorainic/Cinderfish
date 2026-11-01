<script setup lang="ts">
// Renders the head of os.ts's dialog queue: alert, confirm, yes/no, text input
// and single-choice select. It is modal — it owns the backdrop, keeps focus
// inside and closes on Escape — and it stops Escape from reaching the overlay
// below so one keypress never closes two layers.
import { computed, nextTick, onMounted, ref, useId } from "vue";
import { t } from "../../i18n";
import { claimZIndex } from "../../lib/z-index";
import type { DialogGroup, DialogItem, DialogOutcome, DialogRequest } from "../../os";
import AppIcon, { type AppIconName } from "../AppIcon.vue";

const props = defineProps<{ request: DialogRequest }>();
const emit = defineEmits<{ resolve: [outcome: DialogOutcome] }>();

const titleId = useId();
const zIndex = claimZIndex("middle");
const root = ref<HTMLElement | null>(null);

const isSelect = computed(() => props.request.kind === "select");
const isInput = computed(() => props.request.kind === "input");
const isParagraph = computed(
  () => props.request.kind === "input" && props.request.inputType === "paragraph",
);

const groups = computed<DialogGroup[]>(() =>
  props.request.kind === "select" ? (props.request.groupedItems ?? []) : [],
);
const items = computed<DialogItem[]>(() =>
  props.request.kind === "select" ? (props.request.items ?? []) : [],
);

function initialSelection(): string {
  const request = props.request;
  if (request.kind !== "select") return "";
  const values = request.items
    ? request.items.map((item) => item.value)
    : (request.groupedItems ?? []).flatMap((group) => group.items.map((item) => item.value));
  if (request.defaultValue != null && values.includes(request.defaultValue)) {
    return request.defaultValue;
  }
  return values[0] ?? "";
}

const text = ref(props.request.kind === "input" ? props.request.defaultValue : "");
const selected = ref(initialSelection());

const inputType = computed(() =>
  props.request.kind === "input" && props.request.inputType !== "paragraph"
    ? props.request.inputType
    : "text",
);
const placeholder = computed(() =>
  props.request.kind === "input" ? (props.request.placeholder ?? "") : "",
);
const autocomplete = computed(() =>
  props.request.kind === "input" ? (props.request.autocomplete ?? "off") : "off",
);

const tooLong = computed(
  () =>
    props.request.kind === "input" &&
    props.request.maxLength != null &&
    text.value.length > props.request.maxLength,
);
const tooShort = computed(
  () =>
    props.request.kind === "input" &&
    props.request.minLength != null &&
    text.value.length < props.request.minLength,
);

const hint = computed(() => {
  const request = props.request;
  if (request.kind !== "input") return "";
  if (tooLong.value && request.maxLength != null) {
    return t("_dialog.charactersExceeded", { current: text.value.length, max: request.maxLength });
  }
  if (tooShort.value && request.minLength != null) {
    return t("_dialog.charactersBelow", { current: text.value.length, min: request.minLength });
  }
  return "";
});

const canSubmit = computed(() => !tooLong.value && !tooShort.value);

const iconName = computed<AppIconName>(() => {
  switch (props.request.type) {
    case "error":
    case "warning":
      return "warning";
    case "success":
      return "check";
    case "question":
      return "question";
    case "waiting":
      return "clock";
    default:
      return "info";
  }
});

const okLabel = computed(() => {
  const request = props.request;
  return request.kind === "confirm" && request.okText ? request.okText : t("ok");
});
const cancelLabel = computed(() => {
  const request = props.request;
  return request.kind === "confirm" && request.cancelText ? request.cancelText : t("cancel");
});
const acceptLabel = computed(() => (props.request.kind === "yesno" ? t("yes") : okLabel.value));
const dismissLabel = computed(() => (props.request.kind === "yesno" ? t("no") : cancelLabel.value));

const showCancel = computed(() => {
  const request = props.request;
  if (request.kind === "alert") return false;
  if (request.kind === "confirm") return request.showCancelButton;
  return true;
});

let settled = false;

function settle(outcome: DialogOutcome): void {
  if (settled) return;
  settled = true;
  emit("resolve", outcome);
}

function accept(): void {
  const request = props.request;
  if (request.kind === "select") {
    settle(selected.value ? { canceled: false, result: selected.value } : { canceled: true });
    return;
  }
  if (request.kind === "input") {
    if (!canSubmit.value) return;
    settle({ canceled: false, result: text.value });
    return;
  }
  settle({ canceled: false, result: "ok" });
}

function cancel(): void {
  settle({ canceled: true });
}

function focusableElements(): HTMLElement[] {
  const element = root.value;
  if (!element) return [];
  return Array.from(
    element.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
    ),
  );
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") {
    event.preventDefault();
    event.stopPropagation();
    cancel();
    return;
  }
  if (event.key !== "Tab") return;
  const focusable = focusableElements();
  if (focusable.length === 0) {
    event.preventDefault();
    return;
  }
  const first = focusable[0];
  const last = focusable[focusable.length - 1];
  const active = document.activeElement as HTMLElement | null;
  if (event.shiftKey) {
    if (active === first || !root.value?.contains(active)) {
      event.preventDefault();
      last.focus();
    }
  } else if (active === last) {
    event.preventDefault();
    first.focus();
  }
}

onMounted(() => {
  void nextTick(() => {
    const target = root.value?.querySelector<HTMLElement>("[data-autofocus]") ?? root.value;
    target?.focus();
  });
});
</script>

<template>
  <div class="dialog-layer" :style="{ zIndex }" @keydown="onKeydown">
    <div class="dialog-backdrop" aria-hidden="true" @click="cancel"></div>
    <div
      ref="root"
      class="dialog"
      :class="`type-${request.type}`"
      :role="request.kind === 'alert' || request.kind === 'confirm' ? 'alertdialog' : 'dialog'"
      aria-modal="true"
      :aria-labelledby="request.title ? titleId : undefined"
      tabindex="-1"
    >
      <div class="dialog-head">
        <span class="dialog-icon"><AppIcon :name="iconName" /></span>
        <h2 v-if="request.title" :id="titleId" class="dialog-title">{{ request.title }}</h2>
      </div>

      <p v-if="request.text" class="dialog-text">{{ request.text }}</p>

      <input
        v-if="isInput && !isParagraph"
        v-model="text"
        class="dialog-input"
        :type="inputType"
        :placeholder="placeholder"
        :autocomplete="autocomplete"
        data-autofocus
        @keydown.enter.prevent="accept"
      />
      <textarea
        v-else-if="isInput"
        v-model="text"
        class="dialog-input dialog-textarea"
        rows="4"
        :placeholder="placeholder"
        data-autofocus
      ></textarea>
      <p v-if="isInput && hint" class="dialog-hint">{{ hint }}</p>

      <div v-if="isSelect" class="dialog-select" role="listbox" tabindex="-1" data-autofocus>
        <template v-if="groups.length">
          <div v-for="group in groups" :key="group.label" class="select-group">
            <p class="select-group-label">{{ group.label }}</p>
            <button
              v-for="item in group.items"
              :key="item.value"
              type="button"
              class="select-item"
              role="option"
              :aria-selected="selected === item.value"
              :class="{ active: selected === item.value }"
              @click="selected = item.value"
            >
              {{ item.text }}
            </button>
          </div>
        </template>
        <template v-else>
          <button
            v-for="item in items"
            :key="item.value"
            type="button"
            class="select-item"
            role="option"
            :aria-selected="selected === item.value"
            :class="{ active: selected === item.value }"
            @click="selected = item.value"
          >
            {{ item.text }}
          </button>
        </template>
      </div>

      <div class="dialog-actions">
        <button v-if="showCancel" type="button" class="dialog-button" @click="cancel">
          {{ dismissLabel }}
        </button>
        <button
          type="button"
          class="dialog-button primary"
          :disabled="!canSubmit"
          @click="accept"
        >
          {{ acceptLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dialog-layer {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 16px;
}

.dialog-backdrop {
  position: absolute;
  inset: 0;
  background: var(--overlay-bg);
}

.dialog {
  position: relative;
  width: 420px;
  max-width: 100%;
  max-height: calc(100vh - 32px);
  overflow: auto;
  padding: 20px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  color: var(--text);
}

.dialog:focus {
  outline: none;
}

.dialog-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dialog-head:has(.dialog-title) {
  margin-bottom: 10px;
}

.dialog-icon {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex: none;
  border-radius: 50%;
  color: var(--accent);
  background: var(--accent-soft);
}

.type-error .dialog-icon,
.type-warning .dialog-icon {
  color: var(--danger);
  background: var(--danger-soft);
}

.type-success .dialog-icon {
  color: var(--success);
  background: var(--success-soft);
}

.dialog-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  color: var(--text-strong);
}

.dialog-text {
  margin: 10px 0 0;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.dialog-hint {
  margin: 6px 0 0;
  font-size: 0.78rem;
  color: var(--danger);
}

.dialog-input {
  width: 100%;
  margin-top: 12px;
  padding: 8px 10px;
  font: inherit;
  color: var(--text);
  background: var(--surface-inset);
  border: 1px solid var(--input-border);
  border-radius: var(--radius-sm);
}

.dialog-input:hover {
  border-color: var(--input-border-hover);
}

.dialog-input:focus {
  outline: var(--focus-outline);
  outline-offset: 1px;
  border-color: var(--accent);
}

.dialog-textarea {
  resize: vertical;
  min-height: 92px;
}

.dialog-select {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 12px;
  max-height: min(320px, 50vh);
  overflow: auto;
}

.dialog-select:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.select-group + .select-group {
  margin-top: 8px;
}

.select-group-label {
  margin: 6px 0 2px;
  font-size: 0.74rem;
  font-weight: 700;
  color: var(--text-weak);
}

.select-item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  text-align: start;
  font: inherit;
  color: var(--text);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.select-item:hover {
  background: var(--surface-hover);
}

.select-item.active {
  color: var(--accent);
  background: var(--accent-soft);
  border-color: var(--accent-border);
}

.select-item:focus-visible {
  outline: var(--focus-outline);
  outline-offset: -1px;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

.dialog-button {
  padding: 8px 16px;
  font: inherit;
  color: var(--text);
  background: var(--panel-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.dialog-button:hover {
  border-color: var(--border-strong);
}

.dialog-button.primary {
  font-weight: 600;
  color: var(--text-on-accent);
  background: var(--accent);
  border-color: var(--accent);
}

.dialog-button.primary:hover {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
}

.dialog-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.dialog-button:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}
</style>
