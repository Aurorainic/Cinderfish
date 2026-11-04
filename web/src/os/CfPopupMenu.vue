<template>
	<div
		ref="rootEl"
		class="cf-popup-menu _shadow"
		:style="{ top: `${positionY}px`, left: `${positionX}px`, zIndex }"
		@click.stop
		@contextmenu.prevent.stop
	>
		<template v-for="(item, index) in items" :key="index">
			<div v-if="item.type === 'label'" class="label">{{ item.text }}</div>
			<div v-else-if="item.separator" class="separator"></div>
			<button
				v-else
				class="item _button"
				:class="{ active: item.active }"
				@click="select(item)"
			>
				<i v-if="item.icon" :class="item.icon"></i>
				<span class="text">{{ item.text }}</span>
			</button>
		</template>
	</div>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from "vue";
import { claimZIndex } from "./state";
import type { MenuItem } from "./types";

const props = withDefaults(
	defineProps<{
		items: MenuItem[];
		x: number;
		y: number;
	}>(),
	{
		x: 0,
		y: 0,
	},
);

const emit = defineEmits<{
	closed: [];
	select: [item: MenuItem];
}>();

const rootEl = ref<HTMLElement | null>(null);
const positionX = ref(props.x);
const positionY = ref(props.y);
const zIndex = claimZIndex("middle");

function clampPosition(): void {
	const element = rootEl.value;
	if (element == null) return;
	const rect = element.getBoundingClientRect();
	const margin = 8;
	positionX.value = Math.max(
		margin,
		Math.min(props.x, window.innerWidth - rect.width - margin),
	);
	positionY.value = Math.max(
		margin,
		Math.min(props.y, window.innerHeight - rect.height - margin),
	);
}

function select(item: MenuItem): void {
	close();
	item.action?.();
}

function close(): void {
	emit("closed");
}

function onPointerDown(event: PointerEvent): void {
	if (rootEl.value != null && !rootEl.value.contains(event.target as Node)) {
		close();
	}
}

function onKeydown(event: KeyboardEvent): void {
	if (event.key === "Escape") close();
}

onMounted(() => {
	clampPosition();
	window.addEventListener("pointerdown", onPointerDown, true);
	window.addEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
	window.removeEventListener("pointerdown", onPointerDown, true);
	window.removeEventListener("keydown", onKeydown);
});
</script>

<style lang="scss" scoped>
.cf-popup-menu {
	position: fixed;
	min-inline-size: 180px;
	max-inline-size: 320px;
	max-block-size: min(70dvb, 480px);
	overflow: auto;
	padding: 6px;
	background: var(--popup);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	color: var(--fg);
	animation: cf-menu-in 0.12s ease;
}

@keyframes cf-menu-in {
	from {
		opacity: 0;
		transform: scale(0.96);
	}
	to {
		opacity: 1;
		transform: scale(1);
	}
}

.label {
	padding: 8px 12px 4px;
	font-size: 0.75em;
	color: var(--fgTransparentWeak);
	user-select: none;
}

.separator {
	block-size: 1px;
	margin: 4px 8px;
	background: var(--divider);
}

.item {
	display: flex;
	align-items: center;
	gap: 10px;
	inline-size: 100%;
	padding: 9px 12px;
	border-radius: 6px;
	color: var(--fg);
	text-align: start;
	font-size: 0.95em;

	&:hover {
		background: var(--listItemHoverBg);
	}

	&.active {
		color: var(--accent);
	}

	.text {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
}
</style>
