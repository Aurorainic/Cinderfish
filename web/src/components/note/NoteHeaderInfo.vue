// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteHeaderInfo.vue
//
// The strip above a note in a feed: pinned marker or collapsed-reply summary.
// Firefish renders the summary through Mfm; Cinderfish prints plain text.
<template>
	<div v-if="pinned" class="info">
		<AppIcon name="pin" />{{ i18n.ts.pinnedNote }}
	</div>
	<div v-if="collapsedReply && appearNote.reply" class="reply-to">
		<MkAvatar class="avatar" :user="appearNote.reply.user" />
		<MkUserName class="username" :user="appearNote.reply.user" />
		<span class="summary">{{ getNoteSummary(appearNote.reply) }}</span>
	</div>
</template>

<script lang="ts" setup>
import type { Note } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import MkAvatar from "../ui/MkAvatar.vue";
import MkUserName from "../ui/MkUserName.vue";
import { getNoteSummary } from "./get-note-summary";
import { i18n } from "../../i18n";

defineProps<{
	note: Note;
	appearNote: Note;
	collapsedReply?: boolean;
	pinned?: boolean;
}>();
</script>

<style scoped>
.info {
	display: flex;
	align-items: center;
	font-size: 90%;
	white-space: pre;
	color: #f6c177;
}

.info :deep(.icon) {
	margin-inline-end: 4px;
}

.reply-to {
	color: var(--fgTransparentWeak);
	text-overflow: ellipsis;
	display: flex;
	align-items: center;
	gap: 0.4em;
	min-inline-size: 0;
}

.avatar {
	inline-size: 1.2em;
	block-size: 1.2em;
	border-radius: 2em;
	overflow: hidden;
	flex-shrink: 0;
}

.username {
	font-weight: 700;
	flex-shrink: 0;
	max-inline-size: 30%;
}

.username::after {
	content: ": ";
}

.summary {
	display: inline-block;
	max-inline-size: 60%;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.reply-to:hover,
.reply-to:focus-within {
	color: var(--fg);
}
</style>
