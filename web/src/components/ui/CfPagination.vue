<template>
	<div class="cxiknjgy">
		<CfLoading v-if="fetching && items.length === 0" />

		<div v-else-if="error != null && items.length === 0" class="error _fullinfo">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton primary class="retry" @click="reload">{{ t("retry") }}</CfButton>
		</div>

		<div v-else-if="empty" class="empty">
			<slot name="empty">
				<div class="_fullinfo">
					<div>{{ t("nothing") }}</div>
				</div>
			</slot>
		</div>

		<div v-else class="list">
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
	</div>
</template>

<script setup lang="ts" generic="T extends { id: string }">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import CfButton from "./CfButton.vue";
import CfLoading from "./CfLoading.vue";
import { t } from "../../i18n";
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
</style>
