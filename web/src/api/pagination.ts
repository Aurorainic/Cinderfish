import { api } from "./client";
import type { PaginationParams } from "./endpoints";

export interface Cursor {
	sinceId?: string;
	untilId?: string;
}

export const DEFAULT_FIRST_LIMIT = 10;
export const DEFAULT_MORE_LIMIT = 30;

/**
 * Firefish/Misskey cursor rules, as implemented by the backend:
 *   - default order is `id DESC` (newest first);
 *   - `sinceId` alone flips the order to `id ASC` (walking upwards);
 *   - `untilId` walks downwards and never flips.
 */
export function isAscendingCursor(params: PaginationParams): boolean {
	return params.sinceId != null && params.untilId == null;
}

export function olderCursor(lastItemId: string, ascending = false): Cursor {
	return ascending ? { sinceId: lastItemId } : { untilId: lastItemId };
}

export function newerCursor(firstItemId: string): Cursor {
	return { untilId: firstItemId };
}

export interface CursorPage<T> {
	items: T[];
	hasMore: boolean;
	cursor: Cursor;
}

interface Identifiable {
	id: string;
}

/**
 * Fetch one cursor page. Requests `limit + 1` rows so `hasMore` needs no
 * second round-trip (the same trick Firefish's MkPagination uses).
 */
export async function fetchPage<T extends Identifiable>(
	endpoint: string,
	params: Record<string, unknown>,
	limit: number,
	cursor: Cursor,
	token?: string | null,
): Promise<CursorPage<T>> {
	const requestLimit = Math.max(1, limit) + 1;
	const response = await api<T[]>(
		endpoint,
		{ ...params, limit: requestLimit, ...cursor },
		token,
	);

	const items = Array.isArray(response) ? [...response] : [];
	const hasMore = items.length > limit;
	if (hasMore) items.pop();

	const last = items[items.length - 1];
	const next: Cursor =
		last == null
			? cursor
			: olderCursor(last.id, isAscendingCursor({ ...params, ...cursor }));

	return { items, hasMore, cursor: next };
}

/** Merge pages while preserving order and dropping duplicates by `id`. */
export function mergeUnique<T extends Identifiable>(
	existing: readonly T[],
	incoming: readonly T[],
): T[] {
	const seen = new Set(existing.map((item) => item.id));
	const result = [...existing];
	for (const item of incoming) {
		if (seen.has(item.id)) continue;
		seen.add(item.id);
		result.push(item);
	}
	return result;
}

/** Remove an item and keep the array reference stable for Vue reactivity. */
export function removeById<T extends Identifiable>(items: T[], id: string): boolean {
	const index = items.findIndex((item) => item.id === id);
	if (index === -1) return false;
	items.splice(index, 1);
	return true;
}

/**
 * Minimal paginator for non-component callers (stores, prefetch). Components
 * should use `usePagination`, which adds reactive state and queueing.
 */
export class CursorPaginator<T extends Identifiable> {
	private cursor: Cursor = {};
	private items: T[] = [];

	constructor(
		private readonly endpoint: string,
		private readonly params: Record<string, unknown> = {},
		private readonly limit: number = DEFAULT_MORE_LIMIT,
		private readonly token?: string | null,
	) {}

	async reload(): Promise<T[]> {
		this.cursor = {};
		const page = await fetchPage<T>(this.endpoint, this.params, this.limit, {}, this.token);
		this.items = page.items;
		this.cursor = page.cursor;
		this.hasMore = page.hasMore;
		return this.items;
	}

	hasMore = false;

	async loadMore(): Promise<T[]> {
		if (!this.hasMore) return [];
		const page = await fetchPage<T>(this.endpoint, this.params, this.limit, this.cursor, this.token);
		this.items = mergeUnique(this.items, page.items);
		this.cursor = page.cursor;
		this.hasMore = page.hasMore;
		return page.items;
	}

	get value(): readonly T[] {
		return this.items;
	}
}
