<template>
	<div class="cxiknjgy">
		<Transition
			:enter-active-class="animation ? 'fade-enter-active' : ''"
			:leave-active-class="animation ? 'fade-leave-active' : ''"
			:enter-from-class="animation ? 'fade-enter-from' : ''"
			:leave-to-class="animation ? 'fade-leave-to' : ''"
			:mode="animation ? 'out-in' : undefined"
		>
			<CfLoading v-if="fetching && items.length === 0" />

			<div v-else-if="error != null && items.length === 0" key="error" class="error _fullinfo">
				<div>{{ errorMessage }}</div>
				<CfButton primary class="retry" @click="reload">{{ t("retry") }}</CfButton>
			</div>

			<div v-else-if="empty" key="empty" class="empty">
				<slot name="empty">
					<div class="_fullinfo">
						<div>{{ t("nothing") }}</div>
					</div>
				</slot>
			</div>

			<div v-else key="list" class="list">
				<div v-if="reversed && hasMore" class="more">
					<CfButton v-if="!moreFetching" primary @click="loadMore">
						{{ t("loadMore") }}
					</CfButton>
					<CfLoading v-else mini />
				</div>

				<slot :items="items"></slot>

				<div v-if="!reversed && hasMore" ref="sentinelEl" class="more">
					<CfButton v-if="!moreFetching" primary @click="loadMore">
						{{ t("loadMore") }}
					</CfButton>
					<CfLoading v-else mini />
				</div>
			</div>
		</Transition>
	</div>
</template>

<script setup lang="ts" generic="T extends { id: string }">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import CfButton from "./CfButton.vue";
import CfLoading from "./CfLoading.vue";
import { t } from "../../i18n";
import { isApiError } from "../../api/errors";
import { usePagination } from "../../composables/use-pagination";
import { defaultStore } from "../../store/store";
import type { Paging } from "./pagination";

const props = withDefaults(
	defineProps<{
		pagination: Paging;
		disableAutoLoad?: boolean;
	}>(),
	{
		disableAutoLoad: false,
	},
);

const emit = defineEmits<{
	queue: [count: number];
	status: [hasError: boolean];
}>();

const {
	items,
	fetching,
	moreFetching,
	hasMore,
	error,
	empty,
	reversed,
	reload,
	refresh,
	loadMore,
	prepend,
	append,
	removeById,
	updateById,
} = usePagination<T>({
	endpoint: props.pagination.endpoint,
	params: () => props.pagination.params ?? {},
	limit: props.pagination.limit,
	reversed: props.pagination.reversed,
});


// The loading/error/empty/list branches fade through `<Transition>`. The
// user's animation preference (device-scoped) disables both the fade and its
// out-in mode, matching how the rest of the client gates motion.
const animation = computed(() => defaultStore.reactiveState.animation.value);

// `usePagination` keeps the raw rejection around. `fetchPage` throws `ApiError`,
// which carries the backend's `{ message }`; fall back to the generic string
// when the cause is not a structured error (or has no message).
const errorMessage = computed(() =>
	isApiError(error.value) && error.value.message !== ""
		? error.value.message
		: t("somethingHappened"),
);

const sentinelEl = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function setupObserver(): void {
	if (observer != null) observer.disconnect();
	if (props.disableAutoLoad) return;
	if (!defaultStore.state.enableInfiniteScroll) return;
	if (typeof IntersectionObserver === "undefined") return;
	observer = new IntersectionObserver(
		(entries) => {
			for (const entry of entries) {
				if (entry.isIntersecting && hasMore.value && !moreFetching.value) {
					void loadMore();
				}
			}
		},
		{ rootMargin: "400px" },
	);
	if (sentinelEl.value != null) observer.observe(sentinelEl.value);
}

onMounted(setupObserver);
onBeforeUnmount(() => observer?.disconnect());

watch(
	() => sentinelEl.value,
	() => setupObserver(),
);

watch(error, (value) => emit("status", value != null));

defineExpose({
	items,
	reload,
	refresh,
	prepend,
	append,
	removeItem: (finder: (item: T) => boolean) => {
		const found = items.value.find(finder);
		return found == null ? false : removeById(found.id);
	},
	updateItem: (id: string, updater: (old: T) => T) => updateById(id, updater),
});
</script>

<style lang="scss" scoped>
.cxiknjgy {
	> .error,
	> .empty {
		color: var(--fgTransparentWeak);
	}

	.retry {
		margin-block-start: 12px;
	}

	> .list > .more {
		display: flex;
		justify-content: center;
		padding-block: var(--margin);
	}
}

.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.125s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}
</style>
