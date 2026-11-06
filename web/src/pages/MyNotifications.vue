<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<Notifications />
</template>

<script lang="ts" setup>
// Alias for /my/notifications (wave-4 contract §3.6): renders the real
// Notifications page so there is exactly one implementation, and no route or
// logic duplicate.
import Notifications from "./Notifications.vue";

defineOptions({ name: "CfPageMyNotifications" });
</script>
