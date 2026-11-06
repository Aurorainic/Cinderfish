<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/common.vue (AGPL-3.0-only) -->
<template></template>

<script setup lang="ts">
// The `main` channel part of Firefish's common.vue: a new notification shows a
// toast while the page is visible, marks that single notification read, and
// always plays the notification sound (unless the type is muted in settings).
import { onMounted, onUnmounted } from "vue";
import { me } from "@/account";
import { popup } from "@/os/state";
import { stream } from "@/stream";
import * as sound from "@/scripts/sound";
import MkNotificationToast from "@/components/MkNotificationToast.vue";
import type { Notification } from "@/api/types";

let off: (() => void) | null = null;

onMounted(() => {
	const channel = stream.useChannel("main");
	off = channel.on("notification", (notification: Notification) => {
		if (me.value == null) return;
		const muted = me.value.mutingNotificationTypes ?? [];
		if (muted.includes(notification.type)) return;

		if (document.visibilityState === "visible") {
			stream.send("readNotification", { id: notification.id });
			void popup(
				MkNotificationToast,
				{ notification },
				{ closed: () => undefined },
				"closed",
			);
		}
		sound.play("notification");
	});
});

onUnmounted(() => {
	off?.();
	off = null;
});
</script>
