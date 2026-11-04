// Shared prop/type definitions for the UI atoms. They live in a .ts module
// because <script setup> cannot export types.

export interface Paging {
  endpoint: string;
  limit: number;
  params?: Record<string, unknown> | import("vue").Ref<Record<string, unknown>>;
  /** Endpoints that cannot paginate (search); fetch a single page. */
  noPaging?: boolean;
  /** Items are stored oldest-first (newest at the bottom). */
  reversed?: boolean;
  offsetMode?: boolean;
}

export interface PaginationItem {
  id: string;
  [key: string]: unknown;
}

export interface RadioOption {
  value: string | number;
  label: string;
  disabled?: boolean;
}

export interface DateSeparatedItem {
  id: string;
  createdAt: string;
}

export interface WindowButton {
  title: string;
  icon: string;
  highlighted?: boolean;
  onClick: () => void;
}
