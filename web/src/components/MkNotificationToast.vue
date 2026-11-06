<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotificationToast.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-notification-toast">
		<transition name="notification-toast" appear @after-leave="emit('closed')">
			<MkNotification v-if="showing" :notification="notification" class="notification _panel" />
		</transition>
	</div>
</template>

<script setup lang="ts">
// Firefish shows the toast for six seconds and lets the popup manager remove
// it; the component only reports when the leave transition finished.
import { onMounted, ref } from "vue";
import MkNotification from "@/components/MkNotification.vue";
import type { Notification } from "@/api/types";

const props = defineProps<{ notification: Notification }>();
const emit = defineEmits<{ closed: [] }>();

const showing = ref(true);

onMounted(() => {
	window.setTimeout(() => {
		showing.value = false;
	}, 6000);
});
</script>

<style lang="scss" scoped>
.notification-toast-enter-active,
.notification-toast-leave-active {
	transition:
		opacity 0.3s,
		transform 0.3s;
}

.notification-toast-enter-from,
.notification-toast-leave-to {
	opacity: 0;
	transform: translateX(-250px);
}

.mk-notification-toast {
	position: fixed;
	left: 0;
	top: 32px;
	width: 250px;
	padding: 0 32px;
	pointer-events: none;
	z-index: 2000;

	@media (max-width: 700px) {
		top: initial;
		bottom: 112px;
		padding: 0 16px;
	}

	@media (max-width: 500px) {
		bottom: calc(env(safe-area-inset-bottom, 0px) + 92px);
		padding: 0 8px;
	}

	> .notification {
		height: 100%;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
		border-radius: 8px;
		overflow: hidden;
	}
}
</style>
