// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNoteDetailed.vue
//
// Dropped versus the original, with reasons:
// - The conversation above (notes/conversation), the replies/quotes/renotes
//   tabs (notes/replies, notes/renotes) and the reacted-users tab
//   (notes/reactions): none of those endpoints exist in Cinderfish. The
//   note itself renders, the reply context comes from Note.vue's NoteSub
//   row, and the tabs degrade to a reply-count hint. TODO: add the tab
//   strip once notes/children-style listing exists.
// - Word soft-mute, plugins (noteViewInterruptors), translation and clips.
<template>
	<div ref="el" class="note-detailed">
		<NoteRow :note="note" detailed-view />
		<p v-if="note.repliesCount > 0" class="replies-hint">
			<!-- TODO: the reply tab list needs notes/replies or notes/children;
			     Cinderfish has neither, so only the count is shown. -->
			{{ wordWithCount(note.repliesCount, i18n.ts.reply, i18n.ts.replies) }}
		</p>
	</div>
</template>

<script lang="ts" setup>
import type { Note } from "../../api/types";
import NoteRow from "./Note.vue";
import { i18n } from "../../i18n";

defineProps<{
	note: Note;
	pinned?: boolean;
}>();

// The original distinguishes singular/plural only for western languages; the
// Chinese dictionary shares one form, so the count is simply prefixed.
const wordWithCount = (count: number, singular: string, _plural: string) =>
	`${count} ${singular}`;

</script>

<style scoped>
.note-detailed {
	overflow: clip;
}

.replies-hint {
	margin-block-start: 8px;
	padding-inline: 32px;
	opacity: 0.7;
	font-size: 0.9em;
}

@media (max-width: 500px) {
	.replies-hint {
		padding-inline: 16px;
	}
}
</style>
