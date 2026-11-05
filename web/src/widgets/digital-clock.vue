<!-- Ported from firefish@b425f61:packages/client/src/widgets/digital-clock.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="digital _monospace" :style="{ fontSize: `${fontSize}em` }">
			<div v-if="showLabel" class="label">{{ parts.zoneLabel }}</div>
			<div class="time">{{ timeText }}</div>
			<div v-if="showLabel" class="label">{{ parts.offsetLabel }}</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish renders MkDigitalClock with an `offset` in minutes. That component
// does not exist here, so the string is formatted from `Intl` time parts, which
// supports any IANA zone instead of a fixed offset table.
import { computed, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { useInterval } from "@/composables/use-interval";
import { clockParts, timezoneOptions, useWidgetPropsManager } from "./index";
import type { WidgetComponentExpose, WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "digitalClock";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
	{ key: "fontSize", label: "字号 (em)", type: "number", default: 1.5, min: 0.8, max: 4, step: 0.1 },
	{ key: "showMs", label: "显示毫秒", type: "boolean", default: false },
	{ key: "showLabel", label: "显示时区", type: "boolean", default: true },
	{ key: "timezone", label: "时区", type: "enum", default: null, options: timezoneOptions() },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const now = ref(new Date());
useInterval(() => {
	now.value = new Date();
}, 50, { immediate: true });

const timezone = computed<string | null>(() =>
	typeof widgetProps.timezone === "string" ? widgetProps.timezone : null,
);

const parts = computed(() => clockParts(now.value, timezone.value));

const fontSize = computed(() => {
	const value = widgetProps.fontSize;
	return typeof value === "number" && Number.isFinite(value) ? value : 1.5;
});

const showMs = computed(() => widgetProps.showMs === true);
const showLabel = computed(() => widgetProps.showLabel === true);

const timeText = computed(() => {
	const base = `${String(parts.value.hour).padStart(2, "0")}:${String(parts.value.minute).padStart(2, "0")}:${String(parts.value.second).padStart(2, "0")}`;
	return showMs.value
		? `${base}.${String(parts.value.millisecond).padStart(3, "0")}`
		: base;
});

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.digital {
	padding: 16px 0;
	text-align: center;

	> .label {
		font-size: 65%;
		opacity: 0.7;
	}

	> .time {
		font-variant-numeric: tabular-nums;
	}
}
</style>
