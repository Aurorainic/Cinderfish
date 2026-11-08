<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkAcct.vue (AGPL-3.0-only) -->
<template>
	<span class="mk-acct">
		<span class="name">@{{ user.username }}</span>
		<span v-if="user.host || detail || showFullAcct" class="host"
			>@{{ user.host || host }}</span
		>
	</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { UserLite } from "../../api/types";
import { host as hostRaw } from "../../config";
import { defaultStore } from "../../store";

// Firefish decodes the instance host with punycode; that dependency is not part
// of the Cinderfish dependency set, so the host is rendered as configured.
const host = hostRaw;

withDefaults(
	defineProps<{
		user: Pick<UserLite, "username" | "host">;
		detail?: boolean;
	}>(),
	{
		detail: false,
	},
);

const showFullAcct = computed<boolean>(() => defaultStore.state.showFullAcct);
</script>

<style lang="scss" scoped>
.mk-acct {
	> .host {
		color: var(--fgTransparentWeak);
	}
}
</style>
