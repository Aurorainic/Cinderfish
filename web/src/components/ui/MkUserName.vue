// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkUserName.vue
//
// The original renders the name through Mfm so inline emoji in display names
// resolve. TODO: Cinderfish's lib/mfm.ts renderer covers note text but not a
// plain/plain+nowrap name mode, so this port renders the raw string. Custom
// emoji in names degrade to their `:code:` text.
<template>
	<span class="cinderfish-user-name" :class="{ nowrap }">
		{{ user.name || user.username }}
	</span>
</template>

<script lang="ts" setup>
import type { UserLite } from "../../api/types";

withDefaults(
	defineProps<{
		user: UserLite;
		nowrap?: boolean;
	}>(),
	{
		nowrap: true,
	},
);
</script>

<style scoped>
.cinderfish-user-name {
	unicode-bidi: isolate;
}

.cinderfish-user-name.nowrap {
	white-space: nowrap;
}
</style>
