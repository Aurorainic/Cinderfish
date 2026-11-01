// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkPagination.vue
//
// Adaptations:
// - Firefish calls os.api(endpoint) by endpoint name; the Cinderfish api
//   client exposes typed methods instead of a generic dispatcher, so the
//   paging config takes `endpoint` as either a string (used only for the
//   `endpointKey` in errors) or a fetcher function. Callers pass e.g.
//   `(p) => api.usersNotes(p)`. TODO: a generic endpoint dispatch on the
//   api shim would let call sites pass plain endpoint names again.
// - The ad-insertion hook (`_shouldInsertAd_`) and item folding (`folder`
//   prop) are dropped: TODO Cinderfish has no ads backend and no fold pipeline.
// - Firefish's `v-appear` directive becomes an IntersectionObserver based
//   infinite-scroll sentinel; the "load more" button remains as the fallback
//   and always works.
<template>
	<transition name="fade" mode="out-in">
		<MkLoading v-if="fetching" />

		<MkError v-else-if="error" @retry="reload()" />

		<div v-else-if="empty" class="empty">
			<slot name="empty">
				<div class="fullinfo">
					<div class="ghost">🫥</div>
					<div>{{ strings.nothing }}</div>
				</div>
			</slot>
		</div>

		<div v-else ref="rootEl" class="list">
			<div v-show="pagination.reversed && more" class="more">
				<MkButton
					v-if="!moreFetching"
					class="button"
					:disabled="moreFetching"
					:style="{ cursor: moreFetching ? 'wait' : 'pointer' }"
					primary
					@click="fetchMoreAhead"
				>
					{{ strings.loadMore }}
				</MkButton>
				<MkLoading v-else class="loading" />
			</div>
			<slot :items="items"></slot>
			<div v-show="!pagination.reversed && more" ref="sentinelEl" class="more">
				<MkButton
					v-if="!moreFetching"
					class="button"
					:disabled="moreFetching"
					:style="{ cursor: moreFetching ? 'wait' : 'pointer' }"
					primary
					@click="fetchMore"
				>
					{{ strings.loadMore }}
				</MkButton>
				<MkLoading v-else class="loading" />
			</div>
		</div>
	</transition>
</template>

<script lang="ts" setup generic="E extends { id: string }">
import {
	computed,
	isRef,
	onBeforeUnmount,
	onMounted,
	ref,
	shallowRef,
	unref,
	watch,
	type Ref,
} from "vue";
import MkButton from "@/components/ui/MkButton.vue";
import MkError from "@/components/ui/MkError.vue";
import MkLoading from "@/components/ui/MkLoading.vue";

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	loadMore: "加载更多",
	nothing: "没有内容",
};

// Firefish exports the Paging type from the same file; script setup does not
// allow generic interface exports here, so callers inline their own shape.
interface Paging<E> {
	/** Fetcher for the paged endpoint, e.g. `(p) => api.usersNotes(p)`. */
	endpoint: string | ((params: Record<string, unknown>) => Promise<E[]>);
	limit: number;
	secondFetchLimit?: number;
	params?: Record<string, unknown> | Ref<Record<string, unknown>>;
	/** Endpoints without paging support (search APIs). */
	noPaging?: boolean;
	/** Reverse item order in `items` (newest last). */
	reversed?: boolean;
	/** Sort by id ascending instead of descending. */
	ascending?: boolean;
	offsetMode?: boolean;
}

type Item = E;
type Param = Record<string, unknown>;

const SECOND_FETCH_LIMIT_DEFAULT = 30;
const FIRST_FETCH_LIMIT_DEFAULT = 10;

const props = withDefaults(
	defineProps<{
		pagination: Paging<E>;
		disableAutoLoad?: boolean;
		displayLimit?: number;
	}>(),
	{
		disableAutoLoad: false,
		displayLimit: 30,
	},
);

const emit = defineEmits<{
	(ev: "queue", count: number): void;
	(ev: "status", hasError: boolean): void;
}>();

const rootEl = ref<HTMLElement>();
const sentinelEl = ref<HTMLElement>();
// shallowRef: these arrays are replaced wholesale and must not be deeply
// unwrapped, which would break the generic E typing (UnwrapRefSimple).
const items = shallowRef<Item[]>([]);

function toReversed<T>(arr: T[]): T[] {
	return [...arr].reverse();
}

// Not a computed for the same reason Firefish gives: performance.
function calculateItems() {
	const res = [
		toReversed(prepended.value),
		...arrItems.value.map((arr) => arr),
		appended.value,
	].flat(1);
	if (props.pagination.reversed) {
		res.reverse();
	}
	items.value = res;
}

const queue = shallowRef<Item[]>([]);

/** Elements inserted at the front by `prepend`. */
const prepended = shallowRef<Item[]>([]);
/** Frozen item pages. */
const arrItems = shallowRef<Item[][]>([]);
/** Elements inserted at the back by `append`. */
const appended = shallowRef<Item[]>([]);

const idMap = new Map<string, boolean>();

const offset = ref(0);

type PagingByParam =
	| { offset: number }
	| { sinceId: string }
	| { untilId: string }
	| Record<string, never>;
let nextPagingBy: PagingByParam = {};

const fetching = ref(true);
const moreFetching = ref(false);
const more = ref(false);
const backed = ref(false); // scrolling back through history
const isBackTop = ref(false);
const empty = computed(() => items.value.length === 0);
const error = ref(false);

async function callEndpoint(params: Param): Promise<Item[]> {
	const endpoint = props.pagination.endpoint;
	if (typeof endpoint === "function") {
		return endpoint(params);
	}
	throw new Error(
		`MkPagination: endpoint "${endpoint}" needs a fetcher function; ` +
			"the Cinderfish api client has no generic endpoint dispatch",
	);
}

const init = async (): Promise<void> => {
	queue.value = [];
	fetching.value = true;
	await fetch(true);
};

const reload = (): Promise<void> => {
	arrItems.value = [];
	appended.value = [];
	prepended.value = [];
	idMap.clear();
	offset.value = 0;
	nextPagingBy = {};
	return init();
};

async function fetch(firstFetching?: boolean) {
	let limit: number;

	if (firstFetching) {
		limit =
			(props.pagination.limit || FIRST_FETCH_LIMIT_DEFAULT) +
			(props.pagination.noPaging ? 0 : 1);

		if (props.pagination.ascending) {
			// An initial value smaller than all possible ids.
			nextPagingBy = { sinceId: "0" };
		}
	} else {
		if (
			!more.value ||
			fetching.value ||
			moreFetching.value ||
			items.value.length === 0
		)
			return;
		moreFetching.value = true;
		backed.value = true;

		limit =
			(props.pagination.secondFetchLimit ?? SECOND_FETCH_LIMIT_DEFAULT) +
			(props.pagination.noPaging ? 0 : 1);
	}

	const params = props.pagination.params
		? unref(props.pagination.params)
		: {};

	try {
		let res = await callEndpoint({
			...params,
			limit,
			...nextPagingBy,
		});

		if (!props.pagination.noPaging && res.length > limit - 1) {
			res = res.slice(0, limit - 1);
			more.value = true;
		} else {
			more.value = false;
		}

		offset.value += res.length;
		error.value = false;
		fetching.value = false;
		moreFetching.value = false;

		const lastRes = res[res.length - 1];

		if (props.pagination.offsetMode) {
			nextPagingBy = { offset: offset.value };
		} else if (props.pagination.ascending) {
			nextPagingBy = { sinceId: lastRes?.id };
		} else {
			nextPagingBy = { untilId: lastRes?.id };
		}

		// For ascending and offset modes, append and prepend may cause item
		// duplication, so filter the fetched page against what is present.
		if (props.pagination.offsetMode || props.pagination.ascending) {
			for (const item of appended.value) {
				idMap.set(item.id, true);
			}
			res = res.filter((it) => {
				if (idMap.has(it.id)) return false;
				idMap.set(it.id, true);
				return true;
			});
		}

		// `appended` is frozen into arrItems to keep element positions stable.
		arrItems.value.push(appended.value);
		arrItems.value.push(res);
		appended.value = [];

		calculateItems();
	} catch {
		error.value = true;
		fetching.value = false;
		moreFetching.value = false;
	}
}

const fetchMore = async (): Promise<void> => {
	await fetch();
};

const fetchMoreAhead = async (): Promise<void> => {
	await fetch();
};

const prepend = (...item: Item[]): void => {
	// Merge too many prepended items into arrItems to keep the array small.
	if (
		prepended.value.length >
		(props.pagination.secondFetchLimit || SECOND_FETCH_LIMIT_DEFAULT)
	) {
		arrItems.value.unshift(toReversed(prepended.value));
		prepended.value = [];
	}

	if (props.pagination.reversed) {
		prepended.value.push(...item);
		calculateItems();
	} else {
		// Before first display, no scroll position to preserve.
		if (!rootEl.value) {
			prepended.value.push(...item);
			calculateItems();
			return;
		}

		const isTop =
			isBackTop.value ||
			(document.body.contains(rootEl.value) && isTopVisible(rootEl.value));

		if (isTop) {
			prepended.value.push(...item);
			calculateItems();
		} else {
			// Queue items and flush when the user scrolls back to the top.
			queue.value.push(...item);
			onScrollTop(rootEl.value, () => {
				prepend(...queue.value);
				queue.value = [];
			});
		}
	}
};

const append = (...it: Item[]): void => {
	if (
		appended.value.length >
		(props.pagination.secondFetchLimit || SECOND_FETCH_LIMIT_DEFAULT)
	) {
		for (const item of appended.value) {
			idMap.set(item.id, true);
		}
		arrItems.value.push(appended.value);
		appended.value = [];
	}
	appended.value.push(...it);
	calculateItems();
};

function _removeItem(arr: Item[], finder: (item: Item) => boolean): boolean {
	const i = arr.findIndex(finder);
	if (i === -1) return false;
	arr.splice(i, 1);
	return true;
}

function _updateItem(
	arr: Item[],
	id: string,
	replacer: (old: Item) => Item,
): boolean {
	const i = arr.findIndex((item) => item.id === id);
	if (i === -1) return false;
	arr[i] = replacer(arr[i]);
	return true;
}

const removeItem = (finder: (item: Item) => boolean): boolean => {
	const res =
		_removeItem(prepended.value, finder) ||
		_removeItem(appended.value, finder) ||
		arrItems.value.filter((arr) => _removeItem(arr, finder)).length > 0;
	calculateItems();
	return res;
};

const updateItem = (id: string, replacer: (old: Item) => Item): boolean => {
	const res =
		_updateItem(prepended.value, id, replacer) ||
		_updateItem(appended.value, id, replacer) ||
		arrItems.value.filter((arr) => _updateItem(arr, id, replacer)).length > 0;
	calculateItems();
	return res;
};

// Ported from firefish@02c31e0:packages/client/src/scripts/scroll.ts (the
// two helpers MkPagination uses; Cinderfish has no shared scroll module).
function isTopVisible(el: HTMLElement): boolean {
	const rect = el.getBoundingClientRect();
	return rect.top >= 0 && rect.top <= window.innerHeight;
}

function onScrollTop(el: HTMLElement, callback: () => void): void {
	const handler = () => {
		if (el.getBoundingClientRect().top >= 0) {
			window.removeEventListener("scroll", handler);
			callback();
		}
	};
	window.addEventListener("scroll", handler, { passive: true });
}

if (props.pagination.params && isRef<Param>(props.pagination.params)) {
	watch(props.pagination.params, reload, { deep: true });
}

watch(
	queue,
	(a, b) => {
		if (a.length === 0 && b.length === 0) return;
		emit("queue", queue.value.length);
	},
	{ deep: true },
);

watch(error, (n, o) => {
	if (n === o) return;
	emit("status", n);
});

// Infinite scroll: replace Firefish's v-appear directive with an observer on
// the bottom sentinel. The "load more" button stays as the manual fallback.
let observer: IntersectionObserver | undefined;

onMounted(() => {
	observer = new IntersectionObserver(
		(entries) => {
			if (
				entries.some((e) => e.isIntersecting) &&
				!props.disableAutoLoad &&
				more.value &&
				!moreFetching.value &&
				!fetching.value
			) {
				void fetchMore();
			}
		},
		{ rootMargin: "256px" },
	);
	if (sentinelEl.value) observer.observe(sentinelEl.value);
});

watch(sentinelEl, (el, prev) => {
	if (prev && observer) observer.unobserve(prev);
	if (el && observer) observer.observe(el);
});

onBeforeUnmount(() => {
	observer?.disconnect();
});

init();

defineExpose({
	items,
	queue,
	backed,
	reload,
	prepend,
	append,
	removeItem,
	updateItem,
});
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.125s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}

.empty > .fullinfo {
	padding: 32px;
	text-align: center;
}

.empty > .fullinfo > .ghost {
	font-size: 48px;
	margin-block-end: 8px;
}

.list > .more > .button {
	margin-inline: auto;
	margin-block-end: 16px;
}
</style>
