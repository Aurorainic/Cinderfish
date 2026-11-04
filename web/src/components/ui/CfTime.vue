<template>
	<time :title="absolute">
		<template v-if="invalid">{{ t("invalidValue") }}</template>
		<template v-else-if="mode === 'relative'">{{ relative }}</template>
		<template v-else-if="mode === 'absolute'">{{ absolute }}</template>
		<template v-else>{{ absolute }} ({{ relative }})</template>
	</time>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { t } from "../../i18n";

const props = withDefaults(
	defineProps<{
		time: Date | string | number | null;
		origin?: Date | null;
		mode?: "relative" | "absolute" | "detail";
	}>(),
	{
		origin: null,
		mode: "relative",
	},
);

const dateTimeFormat = new Intl.DateTimeFormat(undefined, {
	dateStyle: "medium",
	timeStyle: "short",
});

function getTime(value: Date | string | number): number {
	if (value instanceof Date) return value.getTime();
	return new Date(value).getTime();
}

const timestamp = computed(() =>
	props.time == null ? Number.NaN : getTime(props.time),
);
const invalid = computed(() => Number.isNaN(timestamp.value));
const absolute = computed(() =>
	invalid.value ? t("invalidValue") : dateTimeFormat.format(timestamp.value),
);

const now = ref(props.origin?.getTime() ?? Date.now());

const relative = computed<string>(() => {
	if (props.mode === "absolute") return "";
	if (invalid.value) return t("invalidValue");

	const seconds = Math.abs(now.value - timestamp.value) / 1000;
	const suffix = now.value > timestamp.value ? "_ago" : "_later";

	if (seconds >= 31_536_000) {
		return t(`${suffix}.yearsAgo`, { n: Math.floor(seconds / 31_536_000) });
	}
	if (seconds >= 2_592_000) {
		return t(`${suffix}.monthsAgo`, { n: Math.floor(seconds / 2_592_000) });
	}
	if (seconds >= 604_800) {
		return t(`${suffix}.weeksAgo`, { n: Math.floor(seconds / 604_800) });
	}
	if (seconds >= 86_400) {
		return t(`${suffix}.daysAgo`, { n: Math.floor(seconds / 86_400) });
	}
	if (seconds >= 3600) {
		return t(`${suffix}.hoursAgo`, { n: Math.floor(seconds / 3600) });
	}
	if (seconds >= 60) {
		return t(`${suffix}.minutesAgo`, { n: Math.floor(seconds / 60) });
	}
	if (seconds >= 10) {
		return t(`${suffix}.secondsAgo`, { n: Math.floor(seconds % 60) });
	}
	if (seconds >= -1) {
		return t(`${suffix}.justNow`);
	}
	return t(`${suffix}.future`);
});

let tickId: number | null = null;

function tick(force = false): void {
	if (
		invalid.value ||
		props.origin != null ||
		(props.mode !== "relative" && props.mode !== "detail")
	) {
		if (tickId != null) window.clearInterval(tickId);
		tickId = null;
		return;
	}

	const currentInterval = (now.value - timestamp.value) / 1000;
	now.value = Date.now();
	const nextInterval = (now.value - timestamp.value) / 1000;
	const previousPeriod =
		currentInterval < 60 ? 10_000 : currentInterval < 3600 ? 60_000 : 180_000;
	const nextPeriod =
		nextInterval < 60 ? 10_000 : nextInterval < 3600 ? 60_000 : 180_000;

	if (tickId == null) {
		tickId = window.setInterval(tick, nextPeriod);
	} else if (previousPeriod < nextPeriod || force) {
		window.clearInterval(tickId);
		tickId = window.setInterval(tick, nextPeriod);
	}
}

watch(() => props.time, () => tick(true));
onMounted(() => tick());
onUnmounted(() => {
	if (tickId != null) window.clearInterval(tickId);
});
</script>
