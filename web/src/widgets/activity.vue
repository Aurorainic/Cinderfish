<!-- Ported from firefish@b425f61:packages/client/src/widgets/activity.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.activity')"
		icon="ph-bold ph-chart-bar"
		:show-header="widgetProps.showHeader === true"
		:naked="widgetProps.transparent === true"
	>
		<CfInfo v-if="me == null">登录后可以查看自己的发帖统计。</CfInfo>
		<div v-else class="stats">
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
			<CfInfo>
				逐日活动图表未提供：服务器没有 `charts/user/notes` 接口，这里显示账号的累计统计。
			</CfInfo>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish draws a heatmap / calendar / bar chart from `charts/user/notes`.
// That endpoint does not exist on Cinderfish (api/endpoints.ts), so the widget
// shows the counters the `i` response already carries instead of a fake chart
// (contract §2: no invented data).
import WidgetCard from "./WidgetCard.vue";
import { CfInfo } from "@/components/ui";
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

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.stats {
	display: flex;
	flex-direction: column;
	gap: 8px;
	padding: 12px 14px;

	> .row {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		font-size: 0.9em;

		> .label {
			opacity: 0.7;
		}

		> .value {
			font-weight: bold;
			font-variant-numeric: tabular-nums;
		}
	}
}
</style>
