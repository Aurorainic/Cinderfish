<!-- Ported from firefish@b425f61:packages/client/src/widgets/rss.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		title="RSS"
		icon="ph-bold ph-rss"
		:show-header="widgetProps.showHeader === true"
	>
		<CfLoading v-if="fetching && items.length === 0" />
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="load">{{ t("retry") }}</CfButton>
		</div>
		<div v-else-if="items.length === 0" class="state">{{ t("nothing") }}</div>
		<div
			v-else
			class="feed"
			:style="{ maxHeight: `${widgetProps.height}px` }"
		>
			<a
				v-for="(item, index) in items"
				:key="`${item.link}-${index}`"
				class="item"
				:href="item.link"
				rel="nofollow noopener"
				target="_blank"
				:title="item.title"
			>{{ item.title }}</a>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish fetches the feed through the server (`/api/fetch-rss`) so the
// browser never makes the cross-origin request, and refreshes once a minute.
import { ref, watch } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { apiGet } from "@/api/client";
import type { FetchRssItem } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "rss";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "url", label: "订阅地址", type: "text", default: "" },
	{ key: "height", label: "高度 (px)", type: "number", default: 300, min: 100, max: 1000, step: 10 },
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const items = ref<FetchRssItem[]>([]);
const fetching = ref(true);
const failed = ref(false);

function load(): void {
	const url = String(widgetProps.url ?? "").trim();
	if (url === "") {
		items.value = [];
		failed.value = false;
		fetching.value = false;
		return;
	}
	fetching.value = true;
	failed.value = false;
	apiGet<{ items: FetchRssItem[] }>("fetch-rss", { url })
		.then((feed) => {
			items.value = feed.items ?? [];
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

watch(() => widgetProps.url, load);
useInterval(load, 1000 * 60, { immediate: true, afterMounted: true });

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 16px 14px;
	opacity: 0.8;
}

.feed {
	overflow-y: auto;
	font-size: 0.9em;

	> .item {
		display: block;
		padding: 8px 16px;
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;

		&:nth-child(even) {
			background: rgba(0, 0, 0, 0.04);
		}
	}
}
</style>
