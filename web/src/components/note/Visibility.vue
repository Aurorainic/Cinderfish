// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkVisibility.vue
//
// Icon-only visibility mark with a tooltip label. The original shows a
// scheduled-at clock and resolves the specified-recipients tooltip through
// users/show; Cinderfish has neither scheduled notes nor that popup, so the
// specified case renders the plain envelope/eye-slash mark. TODO: recipient
// tooltip when users/show with userIds exists.
<template>
	<span class="visibility" :title="title">
		<template v-if="note.visibility === 'home'">
			<AppIcon name="home" />
		</template>
		<template v-else-if="note.visibility === 'followers'">
			<AppIcon name="lock" />
		</template>
		<template v-else-if="note.visibility === 'specified'">
			<AppIcon name="eyeSlash" />
		</template>
	</span>
	<span v-if="note.localOnly" class="localOnly" :title="i18n.ts._visibility.localOnly">
		<AppIcon name="users" />
	</span>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Note } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import { i18n } from "../../i18n";

const props = defineProps<{
	note: Note;
}>();

const title = computed(
	() => i18n.ts._visibility[props.note.visibility] ?? props.note.visibility,
);
</script>

<style scoped>
.visibility,
.localOnly {
	margin-inline-start: 0.5em;
	display: inline-flex;
	align-items: center;
	color: var(--fgTransparentWeak);
}

.visibility :deep(.icon),
.localOnly :deep(.icon) {
	width: 1em;
	height: 1em;
}
</style>
