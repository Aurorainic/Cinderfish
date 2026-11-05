<!-- Ported from firefish@b425f61:packages/client/src/widgets/clock.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="face-wrap" :class="size">
			<div v-if="labelsVisible" class="label zone">{{ parts.zoneLabel }}</div>
			<svg class="face" viewBox="0 0 100 100" role="img" :aria-label="timeText">
				<circle class="rim" cx="50" cy="50" r="48" />
				<g v-if="showNumbers" class="numbers">
					<text
						v-for="number in 12"
						:key="number"
						:x="numberX(number)"
						:y="numberY(number)"
						text-anchor="middle"
					>
						{{ number }}
					</text>
				</g>
				<line
					v-for="tick in 60"
					:key="tick"
					class="tick"
					:class="{ major: tick % 5 === 0 }"
					x1="50"
					:y1="tickStart(tick)"
					x2="50"
					y2="47"
					:transform="`rotate(${tick * 6} 50 50)`"
				/>
				<line
					class="hand hour"
					x1="50"
					y1="50"
					x2="50"
					y2="28"
					:style="{ transform: `rotate(${hourAngle}deg)` }"
				/>
				<line
					class="hand minute"
					x1="50"
					y1="50"
					x2="50"
					y2="16"
					:style="{ transform: `rotate(${minuteAngle}deg)` }"
				/>
				<line
					v-if="showSeconds"
					class="hand second"
					x1="50"
					y1="54"
					x2="50"
					y2="10"
					:style="{ transform: `rotate(${secondAngle}deg)` }"
				/>
				<circle class="pivot" cx="50" cy="50" r="2" />
			</svg>
			<div v-if="labelsVisible" class="label time _monospace">{{ timeText }}</div>
			<div v-if="labelsVisible" class="label offset">{{ parts.offsetLabel }}</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish composes MkAnalogClock + MkDigitalClock, both of which read an
// `offset` in minutes. Neither component exists here, so the face is drawn with
// plain SVG and the hand angles are computed from `Intl` time parts, which also
// makes an arbitrary IANA zone work without a fixed-offset table.
import { computed, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { useInterval } from "@/composables/use-interval";
import { clockParts, timezoneOptions, useWidgetPropsManager } from "./index";
import type { WidgetComponentExpose, WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "clock";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
	{
		key: "size",
		label: "尺寸",
		type: "enum",
		default: "medium",
		options: [
			{ value: "small", label: "小" },
			{ value: "medium", label: "中" },
			{ value: "large", label: "大" },
		],
	},
	{ key: "showNumbers", label: "显示数字刻度", type: "boolean", default: true },
	{ key: "showSeconds", label: "显示秒针", type: "boolean", default: true },
	{
		key: "label",
		label: "标签",
		type: "enum",
		default: "none",
		options: [
			{ value: "none", label: "无" },
			{ value: "time", label: "时间" },
			{ value: "tz", label: "时区" },
			{ value: "timeAndTz", label: "时间与时区" },
		],
	},
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
}, 1000, { immediate: true });

const timezone = computed<string | null>(() =>
	typeof widgetProps.timezone === "string" ? widgetProps.timezone : null,
);

const parts = computed(() => clockParts(now.value, timezone.value));

const size = computed(() =>
	["small", "medium", "large"].includes(String(widgetProps.size))
		? String(widgetProps.size)
		: "medium",
);

const showNumbers = computed(() => widgetProps.showNumbers !== false);
const showSeconds = computed(() => widgetProps.showSeconds !== false);

const labelMode = computed(() => String(widgetProps.label ?? "none"));
const labelsVisible = computed(() => labelMode.value !== "none");

const hourAngle = computed(
	() => ((parts.value.hour % 12) + parts.value.minute / 60) * 30,
);
const minuteAngle = computed(
	() => parts.value.minute * 6 + parts.value.second * 0.1,
);
const secondAngle = computed(() => parts.value.second * 6);

const timeText = computed(
	() =>
		`${String(parts.value.hour).padStart(2, "0")}:${String(parts.value.minute).padStart(2, "0")}`,
);

function tickStart(tick: number): number {
	return tick % 5 === 0 ? 42 : 45;
}

function numberX(number: number): number {
	return 50 + 36 * Math.sin((number * 30 * Math.PI) / 180);
}

function numberY(number: number): number {
	return 50 - 36 * Math.cos((number * 30 * Math.PI) / 180) + 3.2;
}

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.face-wrap {
	position: relative;
	display: flex;
	align-items: center;
	justify-content: center;

	&.small {
		padding: 12px;

		> .face {
			height: 100px;
		}
	}

	&.medium {
		padding: 14px;

		> .face {
			height: 150px;
		}
	}

	&.large {
		padding: 16px;

		> .face {
			height: 200px;
		}
	}

	> .face {
		width: auto;
		overflow: visible;

		> .rim {
			fill: none;
			stroke: var(--divider);
			stroke-width: 1.5;
		}

		> .numbers > text {
			fill: var(--fg);
			font-size: 7px;
		}

		> .tick {
			stroke: var(--divider);
			stroke-width: 0.8;

			&.major {
				stroke: var(--fgTransparentWeak);
				stroke-width: 1.4;
			}
		}

		> .hand {
			stroke: var(--fg);
			stroke-linecap: round;
			transform-origin: 50px 50px;

			&.hour {
				stroke-width: 3;
			}

			&.minute {
				stroke-width: 2;
			}

			&.second {
				stroke: var(--accent);
				stroke-width: 0.8;
			}
		}

		> .pivot {
			fill: var(--accent);
		}
	}

	> .label {
		position: absolute;
		font-size: 0.75em;
		opacity: 0.7;

		&.zone {
			top: 12px;
			left: 12px;
		}

		&.time {
			bottom: 12px;
			left: 12px;
		}

		&.offset {
			bottom: 12px;
			right: 12px;
		}
	}
}
</style>
