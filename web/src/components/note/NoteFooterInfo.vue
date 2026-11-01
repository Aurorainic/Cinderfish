// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteFooterInfo.vue
//
// Dropped: the channel link (Cinderfish has no channels). The absolute
// timestamp only shows in the detailed view, as in the original.
<template>
	<div v-if="detailedView" class="footer-info">
		<MkA class="created-at" :to="`/notes/${note.id}`">
			<MkTime :time="note.createdAt" mode="absolute" />
		</MkA>
	</div>
</template>

<script lang="ts" setup>
import type { Note } from "../../api/types";
import MkA from "../ui/MkA.vue";
import MkTime from "../ui/MkTime.vue";

defineProps<{
	note: Note;
	detailedView?: boolean;
}>();
</script>

<style scoped>
.footer-info {
	display: flex;
	justify-content: space-between;
	flex-wrap: wrap;
	gap: 0.7em;
	margin-block-start: 16px;
	opacity: 0.7;
	font-size: 0.9em;
}
</style>
