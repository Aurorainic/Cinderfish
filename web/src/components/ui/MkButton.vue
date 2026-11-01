// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkButton.vue
<template>
	<button
		v-if="!link"
		class="cinderfish-button"
		:class="{ inline, primary, gradate, danger, rounded, full, mini, chip }"
		:type="type ?? 'button'"
		:autofocus="autofocus"
		@click="emit('click', $event)"
		@mousedown="onMousedown"
	>
		<div ref="ripples" class="ripples"></div>
		<div class="content">
			<slot></slot>
		</div>
	</button>
	<RouterLink
		v-else
		class="cinderfish-button"
		:class="{ inline, primary, gradate, danger, rounded, full, mini }"
		:to="to!"
		@mousedown="onMousedown"
	>
		<div ref="ripples" class="ripples"></div>
		<div class="content">
			<slot></slot>
		</div>
	</RouterLink>
</template>

<script lang="ts" setup>
import { nextTick, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";

const props = defineProps<{
	type?: "button" | "submit" | "reset";
	primary?: boolean;
	gradate?: boolean;
	rounded?: boolean;
	inline?: boolean;
	// FIXME: if `link`, `to` is necessary
	link?: boolean;
	to?: string;
	autofocus?: boolean;
	wait?: boolean;
	danger?: boolean;
	full?: boolean;
	mini?: boolean;
	chip?: boolean;
}>();

const emit = defineEmits<{
	click: [payload: MouseEvent];
}>();

const el = ref<HTMLElement | null>(null);
const ripples = ref<HTMLElement | null>(null);

onMounted(() => {
	if (props.autofocus) {
		nextTick(() => {
			el.value?.focus();
		});
	}
});

function distance(
	p: { x: number; y: number },
	q: { x: number; y: number },
): number {
	return Math.hypot(p.x - q.x, p.y - q.y);
}

function calcCircleScale(
	boxW: number,
	boxH: number,
	circleCenterX: number,
	circleCenterY: number,
): number {
	const origin = { x: circleCenterX, y: circleCenterY };
	const dist1 = distance({ x: 0, y: 0 }, origin);
	const dist2 = distance({ x: boxW, y: 0 }, origin);
	const dist3 = distance({ x: 0, y: boxH }, origin);
	const dist4 = distance({ x: boxW, y: boxH }, origin);
	return Math.max(dist1, dist2, dist3, dist4) * 2;
}

// Firefish calls its `vibrate` script here; the direct Vibration API call
// covers the same behaviour without the extra module.
function onMousedown(evt: MouseEvent): void {
	if (typeof navigator.vibrate === "function") navigator.vibrate(10);
	const target = evt.target as HTMLElement;
	const rect = target.getBoundingClientRect();

	const ripple = document.createElement("div");
	ripple.style.top = `${evt.clientY - rect.top - 1}px`;
	ripple.style.left = `${evt.clientX - rect.left - 1}px`;

	ripples.value!.appendChild(ripple);

	const circleCenterX = evt.clientX - rect.left;
	const circleCenterY = evt.clientY - rect.top;

	const scale = calcCircleScale(
		target.clientWidth,
		target.clientHeight,
		circleCenterX,
		circleCenterY,
	);

	window.setTimeout(() => {
		ripple.style.transform = `scale(${scale / 2})`;
	}, 1);
	window.setTimeout(() => {
		ripple.style.transition = "all 1s ease";
		ripple.style.opacity = "0";
	}, 1000);
	window.setTimeout(() => {
		if (ripples.value) ripples.value.removeChild(ripple);
	}, 2000);
}
</script>

<style scoped>
.cinderfish-button {
	position: relative;
	z-index: 1; /* keep above other components' box-shadow */
	display: block;
	min-inline-size: 100px;
	min-block-size: 35px;
	inline-size: max-content;
	padding-block: 8px;
	padding-inline: 16px;
	text-align: center;
	font-weight: normal;
	font-size: max(12px, 1em);
	box-shadow: none;
	text-decoration: none;
	color: inherit;
	background: var(--panel-2, var(--panel));
	border-radius: 5px;
	overflow: clip;
	box-sizing: border-box;
	transition: background 0.1s ease;
	margin-inline: 0.2rem;
	cursor: pointer;
	border: none;
	font-family: inherit;
}

.cinderfish-button:not(:disabled):hover {
	background: var(--accent-soft);
}

.cinderfish-button:not(:disabled):active {
	background: var(--accent-soft);
}

.cinderfish-button.full {
	inline-size: 100%;
}

.cinderfish-button.rounded {
	border-radius: 999px;
}

.cinderfish-button.primary {
	font-weight: bold;
	color: #fff;
	background: var(--accent);
}

.cinderfish-button.primary:not(:disabled):hover,
.cinderfish-button.primary:not(:disabled):active {
	background: var(--accent-strong, var(--accent));
}

.cinderfish-button.gradate {
	font-weight: bold;
	color: #fff;
	background: linear-gradient(
		90deg,
		var(--accent),
		var(--accent-strong, var(--accent))
	);
}

.cinderfish-button.gradate:not(:disabled):hover,
.cinderfish-button.gradate:not(:disabled):active {
	background: var(--accent-strong, var(--accent));
}

.cinderfish-button.danger {
	color: #eb6f92;
}

.cinderfish-button.danger.primary {
	color: #e0def4;
	background: #eb6f92;
}

.cinderfish-button.danger.primary:not(:disabled):hover,
.cinderfish-button.danger.primary:not(:disabled):active {
	background: #eb6f92;
}

.cinderfish-button.mini {
	padding-block: 4px;
	padding-inline: 8px;
	font-size: max(12px, 0.9em);
	border-radius: 100px;
}

.cinderfish-button.chip {
	padding-block: 4px;
	padding-inline: 12px;
	font-size: max(12px, 0.9em);
	min-inline-size: unset;
	border-radius: 100px;
}

.cinderfish-button:disabled {
	opacity: 0.7;
}

.cinderfish-button:focus-visible {
	outline: auto;
}

.cinderfish-button.inline {
	display: inline-block;
	inline-size: auto;
	min-inline-size: 100px;
}

.cinderfish-button > .ripples {
	position: absolute;
	z-index: 0;
	inset: 0;
	inline-size: 100%;
	block-size: 100%;
	border-radius: 6px;
	overflow: hidden;
}

.cinderfish-button > .ripples :deep(div) {
	position: absolute;
	inline-size: 2px;
	block-size: 2px;
	border-radius: 100%;
	background: rgba(0, 0, 0, 0.1);
	opacity: 1;
	transform: scale(1);
	transition: all 0.5s cubic-bezier(0, 0.5, 0, 1);
}

.cinderfish-button.primary > .ripples :deep(div) {
	background: rgba(0, 0, 0, 0.15);
}

.cinderfish-button > .content {
	position: relative;
	z-index: 1;
}
</style>
