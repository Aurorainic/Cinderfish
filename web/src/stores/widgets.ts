import { ref, watch } from "vue";
import {
  DEFAULT_WIDGET_NAMES,
  WIDGET_NAMES,
  WIDGETS,
  type WidgetName,
} from "../widgets/registry";

// The right column's stack is a client-side preference: an ordered list of
// widget instances, each holding its own props. Reading never throws — an
// unreadable, empty or invalid record falls back to the default stack.
export interface WidgetInstance {
  id: string;
  name: WidgetName;
  props: Record<string, unknown>;
}

const STORAGE_KEY = "cinderfish.ui.widgets";

function isWidgetName(value: unknown): value is WidgetName {
  return typeof value === "string" && (WIDGET_NAMES as readonly string[]).includes(value);
}

// Keeps only the keys the widget declares, and only when the stored value has
// the default's type, so a stale or hand-edited record degrades to the default
// instead of breaking the widget.
function normalizeProps(name: WidgetName, raw: unknown): Record<string, unknown> {
  const defaults = WIDGETS[name].defaultProps;
  const out: Record<string, unknown> = { ...defaults };
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return out;
  const record = raw as Record<string, unknown>;
  for (const [key, fallback] of Object.entries(defaults)) {
    const value = record[key];
    if (typeof value !== typeof fallback) continue;
    if (typeof value === "number" && !Number.isFinite(value)) continue;
    out[key] = value;
  }
  return out;
}

function defaultStack(): WidgetInstance[] {
  return DEFAULT_WIDGET_NAMES.map((name, index) => ({
    id: `widget-${name}-${index}`,
    name,
    props: normalizeProps(name, undefined),
  }));
}

let idSeq = 0;

function makeId(): string {
  idSeq += 1;
  return `widget-${Date.now().toString(36)}-${idSeq.toString(36)}`;
}

// Drops anything that is not a well-formed widget, keeps one instance per
// widget name (the add picker only offers missing ones) and repairs props, so
// a partially written record still yields a usable column.
function sanitize(value: unknown): WidgetInstance[] {
  if (!Array.isArray(value)) return [];
  const seen = new Set<string>();
  const seenNames = new Set<WidgetName>();
  const stack: WidgetInstance[] = [];
  for (const entry of value) {
    if (!entry || typeof entry !== "object") continue;
    const { id, name, props } = entry as { id?: unknown; name?: unknown; props?: unknown };
    if (!isWidgetName(name)) continue;
    if (typeof id !== "string" || !id || seen.has(id)) continue;
    if (seenNames.has(name)) continue;
    seen.add(id);
    seenNames.add(name);
    stack.push({ id, name, props: normalizeProps(name, props) });
  }
  return stack;
}

function readStack(): WidgetInstance[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultStack();
    const parsed: unknown = JSON.parse(raw);
    const stack = sanitize(parsed);
    // Partially valid records keep their valid widgets; a record with none
    // usable is treated like a missing one.
    return stack.length ? stack : defaultStack();
  } catch {
    // Unreadable storage (private mode, corrupted JSON) falls back to default.
    return defaultStack();
  }
}

export const widgetStack = ref<WidgetInstance[]>(readStack());

// Edit mode belongs to the list, not to the surface that shows it: the panel and
// the drawer each render their own header, and both edit the same ordered list.
// It is UI state only and is never persisted.
export const widgetEditing = ref(false);

export function toggleWidgetEditing(): void {
  widgetEditing.value = !widgetEditing.value;
}

// Watching the ref instead of persisting inside each mutator also covers
// direct assignment, so the stored stack cannot drift from the rendered one.
watch(
  widgetStack,
  (stack) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(stack));
    } catch {
      // Storage may be unavailable; the stack simply does not persist.
    }
  },
  { deep: true },
);

export function addWidget(name: WidgetName, index?: number): void {
  if (widgetStack.value.some((widget) => widget.name === name)) return;
  const at =
    index === undefined
      ? widgetStack.value.length
      : Math.min(Math.max(index, 0), widgetStack.value.length);
  const widget: WidgetInstance = { id: makeId(), name, props: normalizeProps(name, undefined) };
  widgetStack.value = [
    ...widgetStack.value.slice(0, at),
    widget,
    ...widgetStack.value.slice(at),
  ];
}

export function removeWidget(id: string): void {
  widgetStack.value = widgetStack.value.filter((widget) => widget.id !== id);
}

export function moveWidget(id: string, toIndex: number): void {
  const from = widgetStack.value.findIndex((widget) => widget.id === id);
  if (from < 0) return;
  const to = Math.min(Math.max(toIndex, 0), widgetStack.value.length - 1);
  if (to === from) return;
  const next = widgetStack.value.slice();
  const [widget] = next.splice(from, 1);
  next.splice(to, 0, widget);
  widgetStack.value = next;
}

export function setWidgetProps(id: string, patch: Record<string, unknown>): void {
  widgetStack.value = widgetStack.value.map((widget) =>
    widget.id === id
      ? { ...widget, props: normalizeProps(widget.name, { ...widget.props, ...patch }) }
      : widget,
  );
}

export function resetWidgets(): void {
  widgetStack.value = defaultStack();
}
