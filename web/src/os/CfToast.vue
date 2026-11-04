<template>
	<transition name="cf-toast" appear @after-leave="emit('closed')">
		<div v-if="showing" class="cf-toast-body _acrylic" :style="{ zIndex }">
			<i
				v-if="type === 'success'"
				class="ph-bold ph-check-circle icon success"
			></i>
			<i
				v-else-if="type === 'error'"
				class="ph-bold ph-warning-circle icon error"
			></i>
			<span class="message">{{ message }}</span>
		</div>
	</transition>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { claimZIndex } from "./state";

const props = withDefaults(
	defineProps<{
		message: string;
		type?: "info" | "success" | "error";
		duration?: number;
	}>(),
	{
		type: "info",
		duration: 4000,
	},
);

const emit = defineEmits<{
	closed: [];
}>();

const zIndex = claimZIndex("high");
const showing = ref(true);

onMounted(() => {
	window.setTimeout(() => {
		showing.value = false;
	}, props.duration);
});
</script>

<style lang="scss" scoped>
.cf-toast-body {
	position: fixed;
	inset-block-end: calc(32px + env(safe-area-inset-bottom, 0px));
	inset-inline-start: 50%;
	transform: translateX(-50%);
	display: flex;
	align-items: center;
	gap: 8px;
	max-inline-size: min(480px, calc(100vw - 32px));
	padding: 12px 20px;
	background: var(--acrylicPanel);
	border-radius: 999px;
	box-shadow: 0 4px 24px var(--shadow);
	color: var(--fg);
	font-size: 0.9em;
	-webkit-backdrop-filter: blur(15px);
	backdrop-filter: blur(15px);
}

.icon {
	font-size: 1.2em;

	&.success {
		color: var(--success);
	}
	&.error {
		color: var(--error);
	}
}

.message {
	overflow-wrap: anywhere;
}

.cf-toast-enter-active,
.cf-toast-leave-active {
	transition:
		opacity 0.25s ease,
		transform 0.25s ease;
}

.cf-toast-enter-from,
.cf-toast-leave-to {
	opacity: 0;
	transform: translate(-50%, 8px);
}
</style>
