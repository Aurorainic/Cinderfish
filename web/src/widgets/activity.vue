<!-- Ported from firefish@b425f61:packages/client/src/widgets/activity.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.activity')"
		icon="ph-bold ph-chart-bar"
		:show-header="widgetProps.showHeader === true"
		:naked="widgetProps.transparent === true"
	>
		<CfInfo v-if="me == null">登录后可以查看自己的发帖统计。</CfInfo>
		<div v-else class="body">
			<CfLoading v-if="fetching && chart == null" />
			<CfInfo v-else-if="failed" warn>
				统计加载失败，稍后会自动重试。
			</CfInfo>
			<NotesChart v-else :chart="chart" />
			<div class="stats">
				<div class="row">
					<span class="label">{{ t("notesCount") }}</span>
					<span class="value">{{ me.notesCount }}</span>
				</div>
				<div class="row">
					<span class="label">{{ t("following") }}</span>
					<span class="value">{{ me.followingCount }}</span>
				</div>
				<div class="row">
					<span class="label">{{ t("followers") }}</span>
					<span class="value">{{ me.followersCount }}</span>
				</div>
			</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish draws a heatmap or a calendar from `charts/user/notes`. The shared
// NotesChart component renders the same data as a 30-day stacked bar chart.
import { onMounted, ref, watch } from "vue";
import WidgetCard from "./WidgetCard.vue";
import NotesChart from "@/components/NotesChart.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { UserNotesChart } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "activity";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const chart = ref<UserNotesChart | null>(null);
const fetching = ref(false);
const failed = ref(false);

function load(): void {
	if (me.value == null) {
		chart.value = null;
		return;
	}
	fetching.value = true;
	failed.value = false;
	api<UserNotesChart>("charts/user/notes", {
		span: "day",
		limit: 30,
		userId: me.value.id,
	})
		.then((result) => {
			chart.value = result;
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

onMounted(load);
watch(() => me.value?.id, load);

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.body {
	display: flex;
	flex-direction: column;
	gap: 10px;
	padding: 12px 14px;
}

.stats {
	display: flex;
	flex-direction: column;
	gap: 8px;

	> .row {
		display: flex;
		justify-content: space-between;
		font-size: 0.9em;

		> .label {
			opacity: 0.7;
		}
	}
}
</style>
