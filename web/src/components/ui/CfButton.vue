<template>
	<button
		v-if="!link"
		ref="el"
		class="bghgjjyj _button"
		:class="{ inline, primary, gradate, danger, rounded, full, mini, chip }"
		:type="type"
		:disabled="disabled"
		@click="onClick"
		@mousedown="onMousedown"
	>
		<div ref="ripples" class="ripples"></div>
		<div class="content"><slot></slot></div>
	</button>
	<CfA
		v-else
		class="bghgjjyj _button"
		:class="{ inline, primary, gradate, danger, rounded, full, mini, chip }"
		:to="to ?? '#'"
	>
		<div class="content"><slot></slot></div>
	</CfA>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import CfA from "./CfA.vue";

const props = withDefaults(
	defineProps<{
		type?: "button" | "submit" | "reset";
		primary?: boolean;
		gradate?: boolean;
		rounded?: boolean;
		inline?: boolean;
		link?: boolean;
		to?: string;
		autofocus?: boolean;
		wait?: boolean;
		danger?: boolean;
		full?: boolean;
		mini?: boolean;
		chip?: boolean;
		disabled?: boolean;
	}>(),
	{
		type: "button",
		primary: false,
		gradate: false,
		rounded: false,
		inline: false,
		link: false,
		to: undefined,
		autofocus: false,
		wait: false,
		danger: false,
		full: false,
		mini: false,
		chip: false,
		disabled: false,
	},
);

const emit = defineEmits<{
	click: [event: MouseEvent];
}>();

const el = ref<HTMLElement | null>(null);
const ripples = ref<HTMLElement | null>(null);

onMounted(() => {
	if (props.autofocus) el.value?.focus();
});

function onClick(event: MouseEvent): void {
	if (props.disabled) return;
	emit("click", event);
}

function onMousedown(event: MouseEvent): void {
	if (props.disabled) return;
	const target = event.currentTarget as HTMLElement | null;
	const host = ripples.value;
	if (host == null || target == null) return;

	const rect = target.getBoundingClientRect();
	const ripple = document.createElement("div");
	ripple.style.top = `${event.clientY - rect.top - 1}px`;
	ripple.style.left = `${event.clientX - rect.left - 1}px`;
	host.appendChild(ripple);

	const scale = Math.max(
		Math.hypot(0 - (event.clientX - rect.left), 0 - (event.clientY - rect.top)),
		Math.hypot(rect.width - (event.clientX - rect.left), 0 - (event.clientY - rect.top)),
		Math.hypot(0 - (event.clientX - rect.left), rect.height - (event.clientY - rect.top)),
		Math.hypot(
			rect.width - (event.clientX - rect.left),
			rect.height - (event.clientY - rect.top),
		),
	);

	window.setTimeout(() => {
		ripple.style.transform = `scale(${scale / 2})`;
	}, 1);
	window.setTimeout(() => {
		ripple.style.transition = "all 1s ease";
		ripple.style.opacity = "0";
	}, 1000);
	window.setTimeout(() => {
		ripple.remove();
	}, 2000);
}
</script>

<style lang="scss" scoped>
.bghgjjyj {
	position: relative;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	min-block-size: 38px;
	padding: 0 16px;
	overflow: hidden;
	border-radius: 6px;
	background: var(--buttonBg);
	color: var(--fg);
	font-weight: bold;
	transition:
		background 0.2s,
		color 0.2s;

	&:hover {
		background: var(--buttonHoverBg);
	}

	&.primary {
		background: var(--accent);
		color: var(--fgOnAccent);

		&:hover {
			background: var(--X8);
		}
	}

	&.gradate {
		background: linear-gradient(
			var(--gradient-to-inline-end),
			var(--buttonGradateA),
			var(--buttonGradateB)
		);
		color: var(--fgOnAccent);
	}

	&.danger {
		background: var(--error);
		color: #fff;
	}

	&.rounded {
		border-radius: 999px;
	}

	&.inline {
		display: inline-flex;
		min-block-size: 30px;
		padding: 0 12px;
		font-size: 0.9em;
	}

	&.mini {
		min-block-size: 28px;
		padding: 0 10px;
		font-size: 0.8em;
	}

	&.chip {
		border-radius: 999px;
		min-block-size: 26px;
		padding: 0 10px;
		font-size: 0.8em;
	}

	&.full {
		display: flex;
		inline-size: 100%;
	}

	&:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.ripples {
		position: absolute;
		inset: 0;
		pointer-events: none;

		:deep(div) {
			position: absolute;
			inline-size: 4px;
			block-size: 4px;
			border-radius: 50%;
			background: currentColor;
			opacity: 0.15;
			transform: scale(0);
		}
	}

	.content {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}
}
</style>
