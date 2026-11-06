<!-- Ported from firefish@b425f61:packages/client/src/widgets/unix-clock.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="unix-clock _monospace" :style="{ fontSize: `${fontSize}em` }">
			<div v-if="widgetProps.showLabel !== false" class="label">UNIX Epoch</div>
			<div class="time">
				<span>{{ seconds }}</span>
				<span
					v-if="widgetProps.showMs !== false"
					class="colon"
					:class="{ show: colonVisible }"
					>:</span
				>
				<span v-if="widgetProps.showMs !== false">{{ milliseconds }}</span>
			</div>
			<div v-if="widgetProps.showLabel !== false" class="label">UTC</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Straight port: the epoch clock ticks on `requestAnimationFrame`, and the
// colon between seconds and milliseconds blinks at 500ms like upstream.
import { computed, onMounted, onUnmounted, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "unixClock";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
	{
		key: "fontSize",
		label: "字号（em）",
		type: "number",
		default: 1.5,
		min: 0.5,
		max: 4,
		step: 0.1,
	},
	{ key: "showMs", label: "显示毫秒", type: "boolean", default: true },
	{ key: "showLabel", label: "显示标签", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const now = ref(Date.now());
const fontSize = computed(() =>
	Number(widgetProps.fontSize ?? 1.5),
);
const seconds = computed(() => Math.floor(now.value / 1000));
const milliseconds = computed(() =>
	String(now.value % 1000).padStart(3, "0"),
);
const colonVisible = computed(() => millisecondsBlink(now.value));

function millisecondsBlink(ms: number): boolean {
	return Math.floor(ms / 500) % 2 === 0;
}

let frame = 0;

function tick(): void {
	now.value = Date.now();
	frame = requestAnimationFrame(tick);
}

onMounted(() => {
	frame = requestAnimationFrame(tick);
});

onUnmounted(() => {
	cancelAnimationFrame(frame);
});

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.unix-clock {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 2px;
	padding: 12px 8px;

	.label {
		font-size: 0.5em;
		opacity: 0.55;
		letter-spacing: 0.08em;
	}

	.time {
		display: flex;
		align-items: baseline;
		line-height: 1;
	}

	.colon {
		opacity: 0.3;

		&.show {
			opacity: 1;
		}
	}
}
</style>
