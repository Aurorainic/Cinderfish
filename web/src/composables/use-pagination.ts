import { computed, ref, unref, type ComputedRef, type Ref } from "vue";
import {
	DEFAULT_MORE_LIMIT,
	fetchPage,
	mergeUnique,
	removeById,
	type Cursor,
} from "../api/pagination";

export interface UsePaginationOptions {
	endpoint: string;
	params?: Record<string, unknown> | Ref<Record<string, unknown>> | (() => Record<string, unknown>);
	limit?: number;
	/** Automatically fetch the first page on creation (default true). */
	immediate?: boolean;
	/** When true, newest items stay at the end of `items`. */
	reversed?: boolean;
}

export interface PaginationState<T extends { id: string }> {
	items: Ref<T[]>;
	fetching: Ref<boolean>;
	moreFetching: Ref<boolean>;
	hasMore: Ref<boolean>;
	error: Ref<unknown>;
	empty: Ref<boolean>;
	reversed: ComputedRef<boolean>;
	reload: () => Promise<T[]>;
	refresh: () => Promise<void>;
	loadMore: () => Promise<void>;
	prepend: (...items: T[]) => void;
	append: (...items: T[]) => void;
	removeById: (id: string) => boolean;
	updateById: (id: string, updater: (item: T) => T) => boolean;
}

function resolveParams(
	params: UsePaginationOptions["params"],
): Record<string, unknown> {
	if (params == null) return {};
	if (typeof params === "function") return params();
	return unref(params) as Record<string, unknown>;
}

/**
 * Reactive cursor pagination, wired to `api/pagination.ts`.
 *
 * Replaces Firefish's `MkPagination` internals for pages that do not need the
 * full component: `items` is updated in place so `CfDateSeparatedList` and
 * stream capture can mutate entries without remounting the list.
 */
export function usePagination<T extends { id: string }>(
	options: UsePaginationOptions,
): PaginationState<T> {
	const items = ref<T[]>([]) as Ref<T[]>;
	const fetching = ref(true);
	const moreFetching = ref(false);
	const hasMore = ref(false);
	const error = ref<unknown>(null);
	const limit = options.limit ?? DEFAULT_MORE_LIMIT;
	let cursor: Cursor = {};

	const empty = computed(() => items.value.length === 0 && !fetching.value);

	async function reload(): Promise<T[]> {
		fetching.value = true;
		error.value = null;
		cursor = {};
		try {
			const page = await fetchPage<T>(
				options.endpoint,
				resolveParams(options.params),
				limit,
				cursor,
			);
			items.value = page.items;
			cursor = page.cursor;
			hasMore.value = page.hasMore;
			return page.items;
		} catch (cause) {
			error.value = cause;
			items.value = [];
			hasMore.value = false;
			return [];
		} finally {
			fetching.value = false;
		}
	}

	async function refresh(): Promise<void> {
		const previous = items.value.length || limit;
		fetching.value = true;
		try {
			const page = await fetchPage<T>(
				options.endpoint,
				resolveParams(options.params),
				previous + 1,
				{},
			);
			items.value = page.items;
			cursor = page.cursor;
			hasMore.value = page.hasMore;
			error.value = null;
		} catch (cause) {
			error.value = cause;
		} finally {
			fetching.value = false;
		}
	}

	async function loadMore(): Promise<void> {
		if (!hasMore.value || fetching.value || moreFetching.value) return;
		moreFetching.value = true;
		try {
			const page = await fetchPage<T>(
				options.endpoint,
				resolveParams(options.params),
				limit,
				cursor,
			);
			items.value = mergeUnique(items.value, page.items);
			cursor = page.cursor;
			hasMore.value = page.hasMore;
			error.value = null;
		} catch (cause) {
			error.value = cause;
		} finally {
			moreFetching.value = false;
		}
	}

	function prepend(...incoming: T[]): void {
		if (options.reversed) {
			items.value = mergeUnique(items.value, incoming);
		} else {
			items.value = mergeUnique(incoming, items.value);
		}
	}

	function append(...incoming: T[]): void {
		items.value = mergeUnique(items.value, incoming);
	}

	function updateItem(id: string, updater: (item: T) => T): boolean {
		const index = items.value.findIndex((item) => item.id === id);
		if (index === -1) return false;
		items.value[index] = updater(items.value[index]);
		return true;
	}

	if (options.immediate !== false) {
		void reload();
	}

	return {
		items,
		fetching,
		moreFetching,
		hasMore,
		error,
		empty,
		reversed: computed(() => options.reversed === true),
		reload,
		refresh,
		loadMore,
		prepend,
		append,
		removeById: (id: string) => removeById(items.value, id),
		updateById: updateItem,
	};
}
