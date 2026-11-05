<!-- Ported from firefish@b425f61:packages/client/src/components/MkUserOnlineIndicator.vue (AGPL-3.0-only) -->
<template>
	<div v-tooltip="text" class="fzgwjkgc" :class="status"></div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { UserLite } from "@/api/types";
import { t } from "@/i18n";

// Cinderfish's backend emits onlineStatus: "unknown" for every user
// (docs/research/api-contract/auth-users.md), so the indicator is a stable
// neutral dot until the presence stream lands. The full upstream mapping is
// kept so the component needs no change when it does.
const props = defineProps<{
	user: Pick<UserLite, "onlineStatus">;
}>();

const status = computed<string>(() => {
	switch (props.user.onlineStatus) {
		case "online":
		case "active":
		case "offline":
			return props.user.onlineStatus;
		default:
			return "unknown";
	}
});

const text = computed<string>(() => {
	switch (status.value) {
		case "online":
			return t("online");
		case "active":
			return t("active");
		case "offline":
			return t("offline");
		default:
			return t("unknown");
	}
});
</script>

<style lang="scss" scoped>
.fzgwjkgc {
	box-shadow: 0 0 0 3px var(--panel);
	border-radius: 120%;

	&.online {
		background: #9ccfd8;
	}

	&.active {
		background: #f6c177;
	}

	&.offline {
		background: #eb6f92;
	}

	&.unknown {
		background: #6e6a86;
	}
}
</style>
