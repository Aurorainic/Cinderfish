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
			<template v-else>
				<div class="chart" role="img" :aria-label="`近 ${days.length} 天的发帖量`">
					<div
						v-for="(day, index) in days"
						:key="index"
						class="bar"
						:title="`${day.label}：${day.total} 帖`"
					>
						<div
							class="stack"
							:style="{ height: `${Math.max(day.total === 0 ? 0 : 6, (day.total / maxDay) * 100)}%` }"
						>
							<span
								v-if="day.normal > 0"
								class="seg normal"
								:style="{ flexGrow: day.normal }"
							></span>
							<span
								v-if="day.reply > 0"
								class="seg reply"
								:style="{ flexGrow: day.reply }"
							></span>
							<span
								v-if="day.renote > 0"
								class="seg renote"
								:style="{ flexGrow: day.renote }"
							></span>
						</div>
					</div>
				</div>
				<div class="legend">
					<span class="entry normal">{{ t("note") }}</span>
					<span class="entry reply">{{ t("reply") }}</span>
					<span class="entry renote">{{ t("renote") }}</span>
				</div>
			</template>
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
// Firefish draws a heatmap or a calendar from `charts/user/notes`. The
// calendar/heatmap components are not part of this port, so the same data is
// drawn as a 30-day stacked bar chart: normal / reply / renote per UTC day,
// with the account counters underneath.
import { computed, onMounted, ref, watch } from "vue";
import WidgetCard from "./WidgetCard.vue";
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
const days = computed(() => {
	const data = chart.value;
	if (data == null) return [];
	const now = new Date();
	return data.diffs.normal.map((normal, index) => {
		const reply = data.diffs.reply[index] ?? 0;
		const renote = data.diffs.renote[index] ?? 0;
		const date = new Date(
			Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()),
		);
		date.setUTCDate(date.getUTCDate() - (data.diffs.normal.length - 1 - index));
		return {
			normal,
			reply,
			renote,
			total: normal + reply + renote,
			label: `${date.getUTCMonth() + 1}/${date.getUTCDate()}`,
		};
	});
});
const maxDay = computed(() =>
	Math.max(1, ...days.value.map((day) => day.total)),
);

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

.chart {
	display: flex;
	align-items: flex-end;
	gap: 2px;
	height: 80px;

	> .bar {
		flex: 1;
		height: 100%;
		display: flex;
		align-items: flex-end;

		> .stack {
			width: 100%;
			display: flex;
			flex-direction: column;
			border-radius: 1px;
			overflow: hidden;

			> .seg {
				display: block;
				min-height: 1px;

				&.normal {
					background: var(--accent);
				}

				&.reply {
					background: var(--accentedBg);
				}

				&.renote {
					background: var(--divider);
				}
			}
		}
	}
}

.legend {
	display: flex;
	gap: 10px;
	font-size: 0.75em;
	opacity: 0.75;

	> .entry::before {
		content: "";
		display: inline-block;
		width: 8px;
		height: 8px;
		margin-right: 4px;
		border-radius: 2px;
	}

	> .normal::before {
		background: var(--accent);
	}

	> .reply::before {
		background: var(--accentedBg);
	}

	> .renote::before {
		background: var(--divider);
	}
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
