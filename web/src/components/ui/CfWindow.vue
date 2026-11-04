<template>
	<div
		v-if="showing"
		ref="rootEl"
		class="ebkgocck"
		:class="{ maximized }"
		:style="{ width: `${width}px`, height: height === null ? undefined : `${height}px` }"
		@mousedown="top"
		@keydown="onKeydown"
	>
		<div class="body _shadow _narrow_">
			<div class="header" :class="{ mini }" @contextmenu.prevent.stop="onContextmenu">
				<span class="left">
					<button
						v-for="(button, index) in buttonsLeft"
						:key="index"
						:title="button.title"
						class="button _button"
						:class="{ highlighted: button.highlighted }"
						@click="button.onClick"
					>
						<i :class="button.icon"></i>
					</button>
				</span>
				<span
					class="title"
					@mousedown.prevent="onHeaderMousedown"
					@touchstart.prevent="onHeaderMousedown"
				>
					<slot name="header"></slot>
				</span>
				<span class="right">
					<button
						v-for="(button, index) in buttonsRight"
						:key="index"
						:title="button.title"
						class="button _button"
						:class="{ highlighted: button.highlighted }"
						@click="button.onClick"
					>
						<i :class="button.icon"></i>
					</button>
					<button v-if="canResize" class="button _button" @click="toggleMaximize">
						<i :class="maximized ? 'ph-bold ph-copy' : 'ph-bold ph-browser'"></i>
					</button>
					<button v-if="closeButton" class="button _button" @click="close">
						<i class="ph-bold ph-x"></i>
					</button>
				</span>
			</div>
			<div class="window-body"><slot></slot></div>
		</div>
		<template v-if="canResize">
			<div
				v-for="handle in handles"
				:key="handle"
				class="handle"
				:class="handle"
				@mousedown.prevent="onResizeStart(handle, $event)"
			></div>
		</template>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { claimZIndex, contextMenu } from "../../os";
import type { MenuItem } from "../../os/types";

interface WindowButton {
	icon: string;
	title?: string;
	highlighted?: boolean;
	onClick: () => void;
}

const props = withDefaults(
	defineProps<{
		initialWidth?: number;
		initialHeight?: number | null;
		canResize?: boolean;
		closeButton?: boolean;
		mini?: boolean;
		front?: boolean;
		contextmenu?: MenuItem[] | null;
		buttonsLeft?: WindowButton[];
		buttonsRight?: WindowButton[];
	}>(),
	{
		initialWidth: 400,
		initialHeight: null,
		canResize: false,
		closeButton: true,
		mini: false,
		front: false,
		contextmenu: null,
		buttonsLeft: () => [],
		buttonsRight: () => [],
	},
);

const emit = defineEmits<{
	closed: [];
}>();

const handles = [
	"top",
	"right",
	"bottom",
	"left",
	"top-left",
	"top-right",
	"bottom-right",
	"bottom-left",
] as const;

type Handle = (typeof handles)[number];

const rootEl = ref<HTMLElement | null>(null);
const showing = ref(true);
const maximized = ref(false);
const width = ref(props.initialWidth);
const height = ref<number | null>(props.initialHeight);
const zIndex = claimZIndex(props.front ? "middle" : "low");

let saved: { width: number; height: number | null; top: string; left: string } | null = null;

function top(): void {
	if (rootEl.value != null) rootEl.value.style.zIndex = String(claimZIndex(props.front ? "middle" : "low"));
}

function close(): void {
	showing.value = false;
	emit("closed");
}

function onKeydown(event: KeyboardEvent): void {
	if (event.key === "Escape") {
		event.preventDefault();
		event.stopPropagation();
		close();
	}
}

function onContextmenu(event: MouseEvent): void {
	if (props.contextmenu != null) void contextMenu(props.contextmenu, event);
}

function toggleMaximize(): void {
	const element = rootEl.value;
	if (element == null) return;
	if (!maximized.value) {
		saved = {
			width: width.value,
			height: height.value,
			top: element.style.top,
			left: element.style.left,
		};
		maximized.value = true;
		element.style.top = "0";
		element.style.left = "0";
		element.style.width = "100%";
		element.style.height = "100%";
	} else {
		maximized.value = false;
		width.value = saved?.width ?? props.initialWidth;
		height.value = saved?.height ?? props.initialHeight;
		requestAnimationFrame(() => {
			if (element == null) return;
			element.style.top = saved?.top ?? "100px";
			element.style.left = saved?.left ?? "";
			element.style.width = `${width.value}px`;
			element.style.height = height.value == null ? "" : `${height.value}px`;
		});
	}
}

function pointerMove(
	startX: number,
	startY: number,
	startWidth: number,
	startHeight: number,
	startLeft: number,
	startTop: number,
	direction: Handle,
): void {
	const move = (event: PointerEvent): void => {
		const deltaX = event.clientX - startX;
		const deltaY = event.clientY - startY;
		let nextWidth = startWidth;
		let nextHeight = startHeight;
		let nextLeft = startLeft;
		let nextTop = startTop;

		if (direction.includes("right")) nextWidth = Math.max(250, startWidth + deltaX);
		if (direction.includes("left")) {
			nextWidth = Math.max(250, startWidth - deltaX);
			nextLeft = startLeft + (startWidth - nextWidth);
		}
		if (direction.includes("bottom")) nextHeight = Math.max(50, startHeight + deltaY);
		if (direction.includes("top")) {
			nextHeight = Math.max(50, startHeight - deltaY);
			nextTop = startTop + (startHeight - nextHeight);
		}

		width.value = nextWidth;
		height.value = nextHeight;
		const element = rootEl.value;
		if (element != null) {
			element.style.left = `${nextLeft}px`;
			element.style.top = `${nextTop}px`;
			element.style.width = `${nextWidth}px`;
			element.style.height = `${nextHeight}px`;
		}
	};

	const up = (): void => {
		window.removeEventListener("pointermove", move);
		window.removeEventListener("pointerup", up);
	};

	window.addEventListener("pointermove", move);
	window.addEventListener("pointerup", up);
}

function onResizeStart(handle: Handle, event: MouseEvent): void {
	const element = rootEl.value;
	if (element == null) return;
	const rect = element.getBoundingClientRect();
	pointerMove(
		event.clientX,
		event.clientY,
		rect.width,
		rect.height,
		rect.left,
		rect.top,
		handle,
	);
}

function onHeaderMousedown(event: MouseEvent | TouchEvent): void {
	if (event instanceof MouseEvent && event.button === 2) return;

	const element = rootEl.value;
	if (element == null) return;

	if (maximized.value) toggleMaximize();

	const startX = event instanceof MouseEvent ? event.clientX : event.touches[0].clientX;
	const startY = event instanceof MouseEvent ? event.clientY : event.touches[0].clientY;
	const rect = element.getBoundingClientRect();
	const offsetX = startX - rect.left;
	const offsetY = startY - rect.top;

	const move = (moveEvent: MouseEvent | TouchEvent): void => {
		const clientX =
			moveEvent instanceof MouseEvent ? moveEvent.clientX : moveEvent.touches[0].clientX;
		const clientY =
			moveEvent instanceof MouseEvent ? moveEvent.clientY : moveEvent.touches[0].clientY;
		let nextLeft = clientX - offsetX;
		let nextTop = clientY - offsetY;
		nextLeft = Math.max(0, Math.min(nextLeft, window.innerWidth - element.offsetWidth));
		nextTop = Math.max(0, Math.min(nextTop, window.innerHeight - element.offsetHeight));
		element.style.left = `${nextLeft}px`;
		element.style.top = `${nextTop}px`;
	};

	const up = (): void => {
		window.removeEventListener("mousemove", move);
		window.removeEventListener("touchmove", move);
		window.removeEventListener("mouseup", up);
		window.removeEventListener("touchend", up);
	};

	window.addEventListener("mousemove", move);
	window.addEventListener("touchmove", move, { passive: true });
	window.addEventListener("mouseup", up);
	window.addEventListener("touchend", up);
}

onMounted(() => {
	const element = rootEl.value;
	if (element == null) return;
	element.style.zIndex = String(zIndex);
	element.style.left = `${Math.max(0, (window.innerWidth - width.value) / 2)}px`;
	element.style.top = "100px";
});

defineExpose({ close, maximize: toggleMaximize, unMaximize: toggleMaximize });
</script>

<style lang="scss" scoped>
.ebkgocck {
	position: fixed;
	display: flex;
	flex-direction: column;
	max-inline-size: 100vw;
	max-block-size: 100dvb;
	min-inline-size: 250px;
	min-block-size: 50px;
	outline: none;

	&.maximized {
		border-radius: 0;
	}

	> .body {
		display: flex;
		flex-direction: column;
		inline-size: 100%;
		block-size: 100%;
		min-block-size: 0;
		background: var(--panel);
		border-radius: var(--radius);
		overflow: clip;
	}

	.header {
		display: flex;
		align-items: center;
		flex-shrink: 0;
		block-size: 38px;
		padding-inline: 8px;
		background: var(--windowHeader);
		border-block-end: solid 1px var(--divider);
		cursor: move;

		&.mini {
			block-size: 30px;
		}

		.left,
		.right {
			display: flex;
			align-items: center;
			gap: 2px;
		}

		.title {
			flex: 1;
			min-inline-size: 0;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
			text-align: center;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}

		.button {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			inline-size: 28px;
			block-size: 28px;
			border-radius: 5px;

			&:hover {
				background: var(--buttonBg);
			}

			&.highlighted {
				color: var(--accent);
			}
		}
	}

	.window-body {
		flex: 1;
		min-block-size: 0;
		overflow: auto;
	}
}

.handle {
	position: absolute;

	&.top {
		inset-block-start: -3px;
		inset-inline: 8px;
		block-size: 6px;
		cursor: ns-resize;
	}
	&.bottom {
		inset-block-end: -3px;
		inset-inline: 8px;
		block-size: 6px;
		cursor: ns-resize;
	}
	&.left {
		inset-block: 8px;
		inset-inline-start: -3px;
		inline-size: 6px;
		cursor: ew-resize;
	}
	&.right {
		inset-block: 8px;
		inset-inline-end: -3px;
		inline-size: 6px;
		cursor: ew-resize;
	}
	&.top-left,
	&.top-right,
	&.bottom-left,
	&.bottom-right {
		inline-size: 12px;
		block-size: 12px;
	}
	&.top-left {
		inset-block-start: -4px;
		inset-inline-start: -4px;
		cursor: nwse-resize;
	}
	&.top-right {
		inset-block-start: -4px;
		inset-inline-end: -4px;
		cursor: nesw-resize;
	}
	&.bottom-left {
		inset-block-end: -4px;
		inset-inline-start: -4px;
		cursor: nesw-resize;
	}
	&.bottom-right {
		inset-block-end: -4px;
		inset-inline-end: -4px;
		cursor: nwse-resize;
	}
}
</style>
