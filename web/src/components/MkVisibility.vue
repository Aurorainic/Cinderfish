<!-- Ported from firefish@b425f61:packages/client/src/components/MkVisibility.vue (AGPL-3.0-only) -->
<template>
	<span v-if="note.visibility !== 'public'" class="visibility">
		<i
			v-if="note.visibility === 'home'"
			v-tooltip="t('_visibility.home')"
			class="ph-house ph-bold ph-lg"
		></i>
		<i
			v-else-if="note.visibility === 'followers'"
			v-tooltip="t('_visibility.followers')"
			class="ph-lock ph-bold ph-lg"
		></i>
		<i
			v-else-if="note.visibility === 'specified'"
			v-tooltip="t('_visibility.specified')"
			class="ph-envelope-simple-open ph-bold ph-lg"
		></i>
	</span>
	<span v-if="note.localOnly" class="localOnly"
		><i
			v-tooltip="t('_visibility.localOnly')"
			class="ph-hand-fist ph-bold ph-lg"
		></i
	></span>
</template>

<script setup lang="ts">
import { t } from "@/i18n";

/**
 * Ported from Firefish. The specified branch used to open MkUsersTooltip with
 * users/show, whose userIds list form does not exist in Cinderfish (it accepts
 * a single userId), and MkUsersTooltip is outside this wave component set; the
 * recipient list is therefore not shown.
 */
defineProps<{
	note: {
		visibility: string;
		localOnly?: boolean;
		visibleUserIds?: string[];
	};
}>();
</script>

<style lang="scss" scoped>
.visibility,
.localOnly {
	margin-left: 0.5em;
}
</style>