// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkTime.vue
//
// Relative strings come from the shared lib/time.ts helper instead of the
// Firefish i18n `_ago` section, with simple Chinese wording here.
<template>
	<time :title="absolute">
		<template v-if="invalid">{{ strings.invalid }}</template>
		<template v-else-if="mode === 'relative'">{{ relative }}</template>
		<template v-else-if="mode === 'absolute'">{{ absolute }}</template>
		<template v-else-if="mode === 'detail'">{{ absolute }} ({{ relative }})</template>
	</time>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { absoluteTime, relativeTime } from "../../lib/time";

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	invalid: "无效日期",
};

const props = withDefaults(
	defineProps<{
		time: Date | string | number | null;
		origin?: Date | null;
		mode?: "relative" | "absolute" | "detail";
	}>(),
	{
		mode: "relative",
	},
);

const _time = computed(() => {
	if (props.time == null) return Number.NaN;
	try {
		const d = props.time instanceof Date ? props.time : new Date(props.time);
		return d.getTime();
	} catch {
		return Number.NaN;
	}
});
const invalid = computed(() => Number.isNaN(_time.value));
const absolute = computed(() =>
	!invalid.value
		? absoluteTime(new Date(_time.value).toISOString())
		: strings.invalid,
);

const now = ref(props.origin?.getTime() ?? Date.now());

const relative = computed<string>(() => {
	if (props.mode === "absolute") return ""; // not needed in absolute mode
	if (invalid.value) return strings.invalid;
	return relativeTime(new Date(_time.value).toISOString());
});

let tickId: number | undefined;

function tick(forceUpdateTicker = false) {
	if (
		invalid.value ||
		props.origin ||
		(props.mode !== "relative" && props.mode !== "detail")
	) {
		if (tickId) window.clearInterval(tickId);
		tickId = undefined;
		return;
	}

	const _now = Date.now();
	const currentInterval = (now.value - _time.value) / 1000; /* ms */

	now.value = _now;

	const newInterval = (now.value - _time.value) / 1000; /* ms */
	const prev =
		currentInterval < 60 ? 10000 : currentInterval < 3600 ? 60000 : 180000;
	const next = newInterval < 60 ? 10000 : newInterval < 3600 ? 60000 : 180000;

	if (!tickId) {
		tickId = window.setInterval(tick, next);
	} else if (prev < next || forceUpdateTicker) {
		window.clearInterval(tickId);
		tickId = window.setInterval(tick, next);
	}
}

watch(
	() => props.time,
	() => tick(true),
);

onMounted(() => {
	tick();
});

onUnmounted(() => {
	if (tickId) window.clearInterval(tickId);
});
</script>
