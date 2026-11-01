// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNoteSimple.vue
//
// The compact list row used by search results and mention lists: avatar, mini
// header and content only — no footer, no reactions. Firefish deep-clones the
// note and attaches a note-capture websocket subscription; Cinderfish keeps
// rows stateless, so live updates are not wired here (TODO: subscribe through
// the streaming store when list-level capture exists).
<template>
	<div class="note-simple">
		<MkAvatar class="avatar" :user="note.user" />
		<div class="main">
			<NoteHeader class="header" :note="note" mini />
			<div class="body">
				<NoteContent :note="note" />
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
import type { Note } from "../../api/types";
import MkAvatar from "../ui/MkAvatar.vue";
import NoteContent from "./NoteContent.vue";
import NoteHeader from "./NoteHeader.vue";

defineProps<{
	note: Note;
}>();
</script>

<style scoped>
.note-simple {
	display: flex;
	padding: 16px;
	font-size: 1em;
	border-bottom: solid 0.5px var(--divider);
	overflow: clip;
}

.avatar {
	flex-shrink: 0;
	display: block;
	margin-block: 0;
	margin-inline: 0 10px;
	inline-size: 42px;
	block-size: 42px;
}

.main {
	flex: 1;
	min-inline-size: 0;
}

.header {
	font-size: 0.9em;
}

.body {
	margin-block-start: 0.2em;
}

@media (max-width: 500px) {
	.note-simple {
		padding: 12px;
		font-size: 0.95em;
	}
}
</style>
