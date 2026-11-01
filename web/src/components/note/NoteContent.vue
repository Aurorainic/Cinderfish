// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteContent.vue
//
// Dropped versus the original, with reasons:
// - MFM animation toggle: Cinderfish's lib/mfm.ts is a static renderer, so
//   there is nothing to play or stop (TODO: animated MFM when the renderer
//   supports it).
// - URL previews: no MkUrlPreview equivalent exists (TODO).
// - The jump-to-previous-reply icon: Firefish links to `#replyId` anchors
//   that the Cinderfish note page does not render.
// - Note deletion state comes from the streaming capture in the parent.
<template>
	<p v-if="note.cw != null" class="cw">
		<span v-if="note.cw !== ''" class="text">{{ note.cw }}</span>
	</p>
	<div class="wrapper">
		<div class="content" :class="{ collapsed: isLong && collapsed, showContent: note.cw != null && !showContent, manyImages: (note.files?.length ?? 0) > 4 }">
			<CwButton
				v-if="note.cw != null && !showContent"
				v-model="showContent"
				:note="note"
			/>
			<div
				class="body"
				:aria-hidden="note.cw != null && !showContent ? 'true' : undefined"
				:tabindex="note.cw != null && !showContent ? -1 : undefined"
			>
				<span v-if="note.cw == null && isLong && collapsed" style="opacity: 0.5"></span>
				<!-- eslint-disable-next-line vue/no-v-html -->
				<div v-if="note.text" class="text" v-html="renderedText"></div>
				<MkA v-if="!detailed && note.renoteId" class="rp" :to="`/notes/${note.renoteId}`">
					{{ i18n.ts.quote }}: ...
				</MkA>
				<NoteMediaList v-if="(note.files?.length ?? 0) > 0" :media-list="note.files ?? []" />
				<!-- TODO: Cinderfish has no notes/poll vote endpoints yet; the poll
				     renders read-only once the note type carries one. -->
				<div v-if="note.poll" class="poll">
					<div v-for="(choice, i) in note.poll.choices" :key="i" class="poll-choice">
						<span class="poll-text">{{ choice.text }}</span>
						<span class="poll-votes">{{ choice.votes }}</span>
					</div>
					<p class="poll-total">{{ note.poll.votesCount }} 票</p>
				</div>
				<template v-if="detailed && note.renote">
					<div class="renote" @click.stop="emit('push', note.renote)">
						<NoteSimple :note="note.renote" />
					</div>
				</template>
			</div>
			<CwButton v-if="note.cw != null && showContent" v-model="showContent" :note="note" />
		</div>
	</div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import type { Note } from "../../api/types";
import CwButton from "./CwButton.vue";
import NoteMediaList from "./NoteMediaList.vue";
import NoteSimple from "./NoteSimple.vue";
import MkA from "../ui/MkA.vue";
import { renderMFM } from "../../lib/mfm";
import { i18n } from "../../i18n";

const props = withDefaults(
	defineProps<{
		note: Note;
		detailed?: boolean;
	}>(),
	{
		detailed: false,
	},
);

const emit = defineEmits<{
	(ev: "push", v: Note): void;
}>();

// Mirrors the original's default isLongJudger: over 10 lines, over 800 chars
// or more than 4 files.
const isLong = computed(() => {
	const text = props.note.text;
	if (text == null) return false;
	return text.split("\n").length > 10 || text.length > 800 || (props.note.files?.length ?? 0) > 4;
});

const renderedText = computed(() => renderMFM(props.note.text));

const showContent = ref(false);
const collapsed = ref(props.note.cw == null && isLong.value);

watch(
	() => props.note.id,
	() => {
		showContent.value = false;
		collapsed.value = props.note.cw == null && isLong.value;
	},
);
</script>

<style scoped>
.cw {
	cursor: default;
	display: block;
	margin: 0;
	padding: 0;
	margin-block-end: 10px;
	overflow-wrap: break-word;
}

.cw .text {
	margin-inline-end: 8px;
}

.wrapper .content {
	overflow-wrap: break-word;
}

.body .text {
	white-space: pre-wrap;
	word-break: break-word;
}

.body .rp {
	margin-inline-start: 4px;
	font-style: oblique;
	color: var(--renote);
}

.poll {
	margin-block-start: 0.4em;
	font-size: 80%;
}

.poll-choice {
	display: flex;
	justify-content: space-between;
	gap: 10px;
	padding: 6px 10px;
	border: 1px solid var(--divider);
	border-radius: var(--radius, 6px);
}

.poll-total {
	margin: 6px 0 0;
	color: var(--fgTransparentWeak);
}

.renote {
	padding-block-start: 8px;
}

.renote > * {
	padding: 16px;
	border: solid 1px var(--renote);
	border-radius: 8px;
	transition: background 0.2s;
	cursor: pointer;
}

.renote > *:hover,
.renote > *:focus-within {
	background-color: var(--panelHighlight);
}

.content.collapsed,
.content.showContent {
	position: relative;
	max-block-size: calc(15em + 100px);
}

.content.manyImages.collapsed {
	max-block-size: calc(15em + 250px);
}

.content.collapsed .body,
.content.showContent .body {
	max-block-size: inherit;
	mask: linear-gradient(black calc(100% - 64px), transparent);
	-webkit-mask: linear-gradient(black calc(100% - 64px), transparent);
	overflow: hidden;
	user-select: none;
}

.content.showContent .body {
	min-block-size: 2em;
	max-block-size: 5em;
	filter: blur(5px);
}

.content.showContent .body :deep(img) {
	filter: blur(12px);
}

.content.collapsed .body,
.content.showContent .body {
	box-sizing: border-box;
}
</style>
