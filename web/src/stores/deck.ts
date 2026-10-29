import { ref, watch } from "vue";

// Deck columns are a client-side layout preference: an ordered list of column
// descriptors, persisted under one storage key. Reading never throws — an
// unreadable, empty or invalid record falls back to the default pair
// (local timeline + notifications).
export type ColumnType = "home" | "local" | "hybrid" | "global" | "notifications" | "explore";

export interface ColumnConfig {
  id: string;
  type: ColumnType;
}

// Names from AppIcon's hand-written set; typed here so a typo fails the build
// instead of rendering an empty <path>.
export type ColumnIcon = "home" | "globe" | "feather" | "compass" | "bell" | "search";

export interface ColumnMeta {
  label: string;
  icon: ColumnIcon;
  description: string;
}

// Every column type the menu offers, in menu order.
export const COLUMN_TYPES: readonly ColumnType[] = [
  "home",
  "local",
  "hybrid",
  "global",
  "notifications",
  "explore",
];

export const COLUMN_META: Record<ColumnType, ColumnMeta> = {
  home: { label: "首页", icon: "home", description: "你关注的时间线" },
  local: { label: "本地", icon: "globe", description: "本站公开笔记" },
  hybrid: { label: "社交", icon: "feather", description: "关注与本站的笔记" },
  global: { label: "全球", icon: "compass", description: "联邦宇宙的公开笔记" },
  notifications: { label: "通知", icon: "bell", description: "回复、反应与关注" },
  explore: { label: "探索", icon: "search", description: "按本地或全球发现内容" },
};

const STORAGE_KEY = "cinderfish.deck.columns";

function isColumnType(value: unknown): value is ColumnType {
  return typeof value === "string" && (COLUMN_TYPES as readonly string[]).includes(value);
}

function defaultColumns(): ColumnConfig[] {
  return [
    { id: "deck-local", type: "local" },
    { id: "deck-notifications", type: "notifications" },
  ];
}

let idSeq = 0;

function makeId(): string {
  idSeq += 1;
  return `deck-${Date.now().toString(36)}-${idSeq.toString(36)}`;
}

// Drops anything that is not a well-formed column and any duplicate id, so a
// hand-edited or half-written record can still yield a usable deck.
function sanitize(value: unknown): ColumnConfig[] {
  if (!Array.isArray(value)) return [];
  const seen = new Set<string>();
  const columns: ColumnConfig[] = [];
  for (const entry of value) {
    if (!entry || typeof entry !== "object") continue;
    const { id, type } = entry as { id?: unknown; type?: unknown };
    if (!isColumnType(type)) continue;
    if (typeof id !== "string" || !id || seen.has(id)) continue;
    seen.add(id);
    columns.push({ id, type });
  }
  return columns;
}

function readColumns(): ColumnConfig[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultColumns();
    const parsed: unknown = JSON.parse(raw);
    const columns = sanitize(parsed);
    // Partially valid records keep their valid columns; a record with none
    // usable is treated like a missing one.
    return columns.length ? columns : defaultColumns();
  } catch {
    // Unreadable storage (private mode, corrupted JSON) falls back to default.
    return defaultColumns();
  }
}

export const deckColumns = ref<ColumnConfig[]>(readColumns());

// Watching the ref instead of persisting inside each mutator also covers
// direct assignment, so the stored deck cannot drift from the rendered one.
watch(
  deckColumns,
  (columns) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(columns));
    } catch {
      // Storage may be unavailable; the layout simply does not persist.
    }
  },
  { deep: true },
);

export function addColumn(type: ColumnType, index?: number): void {
  const at = index === undefined ? deckColumns.value.length : Math.min(Math.max(index, 0), deckColumns.value.length);
  const column: ColumnConfig = { id: makeId(), type };
  deckColumns.value = [...deckColumns.value.slice(0, at), column, ...deckColumns.value.slice(at)];
}

export function removeColumn(id: string): void {
  deckColumns.value = deckColumns.value.filter((column) => column.id !== id);
}

export function moveColumn(id: string, toIndex: number): void {
  const from = deckColumns.value.findIndex((column) => column.id === id);
  if (from < 0) return;
  const to = Math.min(Math.max(toIndex, 0), deckColumns.value.length - 1);
  if (to === from) return;
  const next = deckColumns.value.slice();
  const [column] = next.splice(from, 1);
  next.splice(to, 0, column);
  deckColumns.value = next;
}

export function setColumnType(id: string, type: ColumnType): void {
  deckColumns.value = deckColumns.value.map((column) =>
    column.id === id ? { ...column, type } : column,
  );
}
