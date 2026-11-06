<!-- Ported from firefish@b425f61:packages/client/src/widgets/calendar.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="calendar">
			<div class="headline">
				<span class="year">{{ year }} 年</span>
				<span class="month">{{ month }} 月</span>
			</div>
			<div class="day">{{ day }} 日</div>
			<div class="weekday">{{ weekday }}</div>
			<div class="info">
				<div v-for="bar in bars" :key="bar.label" class="bar">
					<div class="barHead">
						<span>{{ bar.label }}</span>
						<b>{{ bar.percent.toFixed(1) }}%</b>
					</div>
					<div class="meter">
						<div class="val" :style="{ width: `${bar.percent}%` }"></div>
					</div>
				</div>
			</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// The upstream widget also shows a birthday emoji when `$i.birthday` is set.
// Cinderfish has no birthday field, and a new year / new month greeting would
// be the only other special case, so the date line stays plain.
import { computed, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { useInterval } from "@/composables/use-interval";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "calendar";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const now = ref(new Date());

const year = computed(() => now.value.getFullYear());
const month = computed(() => now.value.getMonth() + 1);
const day = computed(() => now.value.getDate());
const weekday = computed(
	() =>
		["星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"][
			now.value.getDay()
		],
);

const bars = computed(() => {
	const current = now.value.getTime();
	const y = now.value.getFullYear();
	const m = now.value.getMonth();
	const d = now.value.getDate();
	const dayStart = new Date(y, m, d).getTime();
	const monthStart = new Date(y, m, 1).getTime();
	const yearStart = new Date(y, 0, 1).getTime();
	const nextMonth = new Date(y, m + 1, 1).getTime();
	const nextYear = new Date(y + 1, 0, 1).getTime();
	const percent = (elapsed: number, total: number): number =>
		total <= 0 ? 0 : Math.min(100, Math.max(0, (elapsed / total) * 100));

	return [
		{
			label: "今天",
			percent: percent(current - dayStart, 24 * 60 * 60 * 1000),
		},
		{
			label: "本月",
			percent: percent(current - monthStart, nextMonth - monthStart),
		},
		{
			label: "今年",
			percent: percent(current - yearStart, nextYear - yearStart),
		},
	];
});

useInterval(
	() => {
		now.value = new Date();
	},
	1000 * 30,
	{ immediate: true },
);

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.calendar {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 12px 14px 14px;

	.headline {
		display: flex;
		gap: 8px;
		font-size: 0.85em;
		opacity: 0.7;
	}

	.day {
		font-size: 2.2em;
		font-weight: 700;
		line-height: 1.2;
	}

	.weekday {
		font-size: 0.85em;
		opacity: 0.65;
	}

	.info {
		width: 100%;
		margin-top: 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.barHead {
		display: flex;
		justify-content: space-between;
		font-size: 0.78em;
		opacity: 0.8;
	}

	.meter {
		height: 5px;
		border-radius: 999px;
		background: var(--bg);
		overflow: hidden;
	}

	.val {
		height: 100%;
		background: var(--accent, #86b300);
		border-radius: inherit;
	}
}
</style>
