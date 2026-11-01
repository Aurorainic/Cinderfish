// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNoteSub.vue
//
// A single reply row in the conversation. Dropped versus the original, with
// reasons:
// - The conversation / auto-add-replies tree: Cinderfish has no
//   notes/children endpoint, so only the single row is rendered (TODO).
// - Word soft-mute and the context menu (see Note.vue).
<template>
	<article class="note-sub" tabindex="-1">
		<div class="main" :style="{ cursor: expandOnNoteClick ? 'pointer' : '' }" @click="noteClick">
			<div class="avatar-container">
				<MkAvatar class="avatar" :user="appearNote.user" />
				<div class="line"></div>
			</div>
			<div class="body">
				<NoteHeader class="header" :note="note" mini />
				<div class="body-content">
					<NoteContent :note="note" />
				</div>
				<NoteFooter class="footer" :note="appearNote" />
			</div>
		</div>
	</article>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { Note } from "../../api/types";
import MkAvatar from "../ui/MkAvatar.vue";
import NoteContent from "./NoteContent.vue";
import NoteFooter from "./NoteFooter.vue";
import NoteHeader from "./NoteHeader.vue";
import { defaultStore } from "../../store/firefish-store";

const props = defineProps<{
	note: Note;
	// Present in the original for the conversation tree; accepted for call
	// compatibility and unused without a notes/children endpoint.
	detailedView?: boolean;
}>();

const router = useRouter();

const expandOnNoteClick = defaultStore.reactiveState.expandOnNoteClick;

// Like the original: a reply whose only content is a renote renders the
// renoted note.
const appearNote = computed(() =>
	props.note.renote && props.note.text == null && props.note.fileIds.length === 0 && props.note.poll == null
		? (props.note.renote as Note)
		: props.note,
);

function noteClick(e: MouseEvent) {
	if (!expandOnNoteClick.value) return;
	const target = e.target instanceof Element ? e.target : null;
	if (target?.closest("a, button, input, textarea, select, [contenteditable]")) return;
	router.push(`/notes/${appearNote.value.id}`);
}
</script>

<style scoped>
.note-sub {
	display: flex;
	font-size: 1em;
	position: relative;
}

.main {
	display: flex;
	flex: 1;
	min-inline-size: 0;
}

.avatar-container {
	position: relative;
	flex-shrink: 0;
}

.avatar {
	inline-size: 38px;
	block-size: 38px;
}

.line {
	position: absolute;
	inline-size: 2px;
	inset-block-start: calc(38px + 4px);
	inset-block-end: -8px;
	inset-inline-start: calc(38px / 2 - 1px);
	background: currentColor;
	opacity: 0.25;
}

.body {
	flex: 1;
	min-inline-size: 0;
	margin-inline-start: 8px;
}

.header {
	font-size: 0.9em;
}

.body-content {
	margin-block-start: 0.2em;
}

.footer {
	font-size: 0.9em;
}

@media (max-width: 450px) {
	.avatar {
		inline-size: 32px;
		block-size: 32px;
	}
}
</style>
