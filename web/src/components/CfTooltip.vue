<!-- Cinderfish: replaces firefish@b425f61:packages/client/src/components/MkTooltip.vue (AGPL-3.0-only) -->
<template>
	<transition name="cf-tooltip" appear @after-leave="emit('closed')">
		<div
			v-if="showing"
			ref="tooltipEl"
			class="cf-tooltip"
			:class="direction"
			:style="{ top: `${top}px`, left: `${left}px`, zIndex }"
			role="tooltip"
		>
			{{ text }}
		</div>
	</transition>
</template>

<script setup lang="ts">
// Firefish positions its Mfm tooltip through os.popup and places it above or
// below the source. This version does the same with plain text: fixed
// positioning, clamped to the viewport, pointer-events none so the bubble can
// never capture the hover that keeps it open.
import { nextTick, onMounted, ref } from "vue";
import { claimZIndex } from "@/os/state";

const props = withDefaults(
	defineProps<{
		showing: boolean;
		text: string;
		source: HTMLElement;
		direction?: "top" | "bottom";
	}>(),
	{ direction: "top" },
);

const emit = defineEmits<{ closed: [] }>();

const zIndex = claimZIndex("high");
const top = ref(0);
const left = ref(0);
const tooltipEl = ref<HTMLElement | null>(null);

onMounted(() => {
	const sourceRect = props.source.getBoundingClientRect();
	const half = 130;
	left.value = Math.min(
		Math.max(sourceRect.left + sourceRect.width / 2, half + 8),
		window.innerWidth - half - 8,
	);
	top.value = props.direction === "bottom" ? sourceRect.bottom + 6 : sourceRect.top - 6;

	void nextTick(() => {
		const tooltip = tooltipEl.value;
		if (tooltip == null) return;
		const rect = tooltip.getBoundingClientRect();
		const shiftX = Math.max(8 - rect.left, Math.min(0, window.innerWidth - rect.right - 8));
		const shiftY = Math.max(8 - rect.top, Math.min(0, window.innerHeight - rect.bottom - 8));
		left.value += shiftX;
		top.value += shiftY;
	});
});
</script>

<style lang="scss" scoped>
.cf-tooltip {
	position: fixed;
	max-width: min(250px, calc(100dvw - 16px));
	padding: 4px 10px;
	border-radius: 6px;
	background: rgb(20 20 24 / 92%);
	color: #fff;
	font-size: 0.78em;
	line-height: 1.4;
	text-align: center;
	pointer-events: none;
	box-shadow: 0 2px 8px rgb(0 0 0 / 30%);

	&.top {
		transform: translate(-50%, -100%);
	}

	&.bottom {
		transform: translate(-50%, 0);
	}
}

.cf-tooltip-enter-active,
.cf-tooltip-leave-active {
	transition: opacity 0.12s ease;
}

.cf-tooltip-enter-from,
.cf-tooltip-leave-to {
	opacity: 0;
}
</style>
