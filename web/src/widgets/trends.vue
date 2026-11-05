<!-- Ported from firefish@b425f61:packages/client/src/widgets/trends.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.trends')"
		icon="ph-bold ph-hash"
		:show-header="widgetProps.showHeader === true"
	>
		<CfLoading v-if="fetching && stats.length === 0" />
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="loadTrends">{{ t("retry") }}</CfButton>
		</div>
		<div v-else-if="stats.length === 0" class="state">{{ t("nothing") }}</div>
		<div v-else class="tags">
			<div v-for="stat in stats" :key="stat.tag" class="tag">
				<MkA
					class="a"
					:to="`/tags/${encodeURIComponent(stat.tag)}`"
					:title="stat.tag"
				>
					#{{ stat.tag }}
				</MkA>
				<p>{{ t("nUsersMentioned", { n: stat.usersCount }) }}</p>
			</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish draws `stat.chart` with MkMiniChart; chart.js is not a dependency
// (contract §2), so only the tag and the user count are shown.
import { ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { HashtagTrend } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "trends";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const stats = ref<HashtagTrend[]>([]);
const fetching = ref(true);
const failed = ref(false);

function loadTrends(): void {
	fetching.value = true;
	failed.value = false;
	api<HashtagTrend[]>("hashtags/trend")
		.then((result) => {
			stats.value = result.slice(0, 5);
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

useInterval(loadTrends, 1000 * 60, { immediate: true, afterMounted: true });

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.tags {
	> .tag {
		padding: 10px 14px;
		border-bottom: solid 0.5px var(--divider);

		&:last-child {
			border-bottom: none;
		}

		> .a {
			display: block;
			font-size: 0.9em;
			color: var(--fg);
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}

		> p {
			margin: 0;
			font-size: 75%;
			opacity: 0.7;
		}
	}
}

.state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 16px;
	font-size: 0.9em;
	color: var(--fgTransparentWeak);
	text-align: center;
}
</style>
