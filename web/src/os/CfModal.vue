<template>
	<div
		class="cf-modal _modalBg"
		:style="{ zIndex }"
		@click.self="onBackdropClick"
	>
		<slot></slot>
	</div>
</template>

<script setup lang="ts">
import { claimZIndex } from "./state";

const props = withDefaults(
	defineProps<{
		zIndex?: number;
		/** Clicking the dimmed area closes the modal (dialogs opt out). */
		closeOnBackdrop?: boolean;
	}>(),
	{
		zIndex: 0,
		closeOnBackdrop: true,
	},
);

const emit = defineEmits<{
	click: [event: MouseEvent];
	closed: [];
}>();

const zIndex = props.zIndex === 0 ? claimZIndex("high") : props.zIndex;

function onBackdropClick(event: MouseEvent): void {
	emit("click", event);
	if (props.closeOnBackdrop) close();
}

function close(): void {
	emit("closed");
}

defineExpose({ close });
</script>

<style lang="scss" scoped>
.cf-modal {
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 32px;

	@media (max-inline-size: 500px) {
		padding: 16px;
	}
}
</style>
