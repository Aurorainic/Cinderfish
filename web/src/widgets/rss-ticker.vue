<!-- Ported from firefish@b425f61:packages/client/src/widgets/rss-ticker.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="widgetProps.showHeader === true"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<CfLoading v-if="fetching && items.length === 0" />
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="load">{{ t("retry") }}</CfButton>
		</div>
		<div v-else-if="items.length === 0" class="state">{{ t("nothing") }}</div>
		<MkMarquee
			v-else
			class="ticker"
			:duration="widgetProps.duration as number"
			:reverse="widgetProps.reverse === true"
		>
			<span v-for="(item, index) in items" :key="`${item.link}-${index}`" class="entry">
				<a
					class="link"
					:href="item.link"
					rel="nofollow noopener"
					target="_blank"
					:title="item.title"
				>{{ item.title }}</a>
				<span class="divider"></span>
			</span>
		</MkMarquee>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish scrolls the headlines with MkMarquee and reshuffles on every
// refresh; the server-side fetch and the widget options are unchanged.
import { ref, watch } from "vue";
import WidgetCard from "./WidgetCard.vue";
import MkMarquee from "@/components/MkMarquee.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { apiGet } from "@/api/client";
import type { FetchRssItem } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "rssTicker";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "url", label: "订阅地址", type: "text", default: "" },
	{ key: "shuffle", label: "随机顺序", type: "boolean", default: true },
	{ key: "refreshIntervalSec", label: "刷新间隔 (秒)", type: "number", default: 60, min: 30, max: 3600, step: 10 },
	{ key: "duration", label: "滚动时长 (秒)", type: "number", default: 70, min: 5, max: 200, step: 1 },
	{ key: "reverse", label: "反向滚动", type: "boolean", default: false },
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: false },
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
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

function shuffle(list: FetchRssItem[]): FetchRssItem[] {
	const copy = list.slice();
	for (let i = copy.length - 1; i > 0; i--) {
		const j = Math.floor(Math.random() * (i + 1));
		[copy[i], copy[j]] = [copy[j], copy[i]];
	}
	return copy;
}

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
			items.value = widgetProps.shuffle === true ? shuffle(feed.items ?? []) : feed.items ?? [];
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

watch(() => widgetProps.url, load);
useInterval(load, 1000 * Math.max(30, Number(widgetProps.refreshIntervalSec) || 60), {
	immediate: true,
	afterMounted: true,
});

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

.ticker {
	padding: 8px 0;
	font-size: 0.9em;

	.entry {
		display: inline-flex;
		align-items: center;

		> .divider {
			display: inline-block;
			width: 24px;
			height: 1px;
			margin: 0 12px;
			background: var(--divider);
		}
	}
}
</style>
