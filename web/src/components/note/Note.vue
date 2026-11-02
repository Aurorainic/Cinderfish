// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNote.vue
//
// Dropped versus the original, with reasons:
// - Word soft-mute (check-word-mute): the mutedWords store slice and the
//   matching engine are not ported (TODO).
// - noteViewInterruptors (plugins): Cinderfish has no plugin runtime.
// - Renote aggregation (the `renotes` prop and multi-avatar renote bar):
//   Cinderfish has no notes/renotes endpoint, so a renote renders its own
//   single bar. TODO: aggregate renotes when the endpoint exists.
// - v-vibrate and context menu: no vibrate runtime; the context menu items
//   all live in the footer's more-menu already.
<template>
	<div v-if="!isDeleted" ref="el" class="note-container" tabindex="-1">
		<NoteSub
			v-if="appearNote.reply && !detailedView && !collapsedReply"
			:note="appearNote.reply"
			class="reply-to"
		/>
		<div v-if="!detailedView" class="note-context" @click="noteClick">
			<NoteHeaderInfo :note="note" :appear-note="appearNote" :collapsed-reply="collapsedReply && !!appearNote.reply" :pinned="pinned" />
			<RenoteBar :note="note" :appear-note="appearNote" :is-renote="isRenote" />
		</div>
		<article class="article" :style="{ cursor: expandOnNoteClick && !detailedView ? 'pointer' : '' }" @click="noteClick">
			<div class="main">
				<div class="header-container">
					<MkAvatar class="avatar" :user="appearNote.user" />
					<NoteHeader class="header" :note="appearNote" />
				</div>
				<div class="body">
					<NoteContent
						:note="appearNote"
						:detailed="true"
						:detailed-view="detailedView"
						@push="(e) => router.push(`/notes/${e.id}`)"
					/>
				</div>
				<NoteFooter
					class="footer"
					ref="footerEl"
					:note="appearNote"
					:detailed-view="detailedView"
					@deleted="isDeleted = true"
					@event:focus="focus"
					@event:blur="blur"
					@renoted="onRenoted"
				/>
			</div>
		</article>
	</div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import type { Note } from "../../api/types";
import { focusNext, focusPrev } from "../../lib/focus-sibling";
import MkAvatar from "../ui/MkAvatar.vue";
import NoteContent from "./NoteContent.vue";
import NoteFooter from "./NoteFooter.vue";
import NoteHeader from "./NoteHeader.vue";
import NoteHeaderInfo from "./NoteHeaderInfo.vue";
import NoteSub from "./NoteSub.vue";
import RenoteBar from "./RenoteBar.vue";
import { defaultStore } from "../../store/firefish-store";
import { useNoteCapture } from "./use-note-capture";

const props = withDefaults(
	defineProps<{
		note: Note;
		pinned?: boolean;
		detailedView?: boolean;
		collapsedReply?: boolean;
	}>(),
	{
		pinned: false,
		detailedView: false,
		collapsedReply: false,
	},
);

const router = useRouter();

const keymap = {
	r: () => footerEl.value?.reply(),
	"e|a|plus": () => footerEl.value?.react(),
	"up|k": focusBefore,
	"down|j": focusAfter,
	esc: blur,
	"m|o": () => footerEl.value?.menu(),
};

const el = ref<HTMLElement | null>(null);
const footerEl = ref<InstanceType<typeof NoteFooter> | null>(null);

const expandOnNoteClick = defaultStore.reactiveState.expandOnNoteClick;

const isDeleted = ref(false);

const isRenote = computed(
	() =>
		props.note.renote != null &&
		props.note.text == null &&
		props.note.fileIds.length === 0 &&
		props.note.poll == null,
);
const appearNote = computed(() => (isRenote.value ? (props.note.renote as Note) : props.note));

// The innerNote ref mirrors the props note so the streaming capture can
// mutate reactions in place without touching the store's copy.
const innerNote = ref(props.note);
watch(
	() => props.note,
	(n) => {
		innerNote.value = n;
	},
);

useNoteCapture({
	rootEl: el,
	note: innerNote,
	isDeletedRef: isDeleted,
});

function focus() {
	el.value?.focus();
}

function blur() {
	el.value?.blur();
}

function focusBefore() {
	focusPrev(el.value);
}

function focusAfter() {
	focusNext(el.value);
}

function noteClick(e: MouseEvent) {
	if (
		document.getSelection()?.type === "Range" ||
		props.detailedView ||
		!expandOnNoteClick.value
	) {
		e.stopPropagation();
	} else {
		const target = e.target instanceof Element ? e.target : null;
		if (target?.closest("a, button, input, textarea, select, [contenteditable]")) return;
		router.push(`/notes/${appearNote.value.id}`);
	}
}

function onRenoted() {
	appearNote.value.renoteCount += 1;
}

onMounted(() => {
	el.value?.addEventListener("keydown", keymap as unknown as EventListener);
});
</script>

<style scoped>

.note-container {
	position: relative;
	inline-size: 100%;
	transition: box-shadow 0.1s ease;
	font-size: 1.05em;
	overflow: clip;
	contain: content;
	-webkit-tap-highlight-color: transparent;
}

.note-container:focus-visible {
	outline: none;
}

.note-container:focus-visible::after {
	content: "";
	pointer-events: none;
	display: block;
	position: absolute;
	z-index: 10;
	inset: 0;
	margin: auto;
	inline-size: calc(100% - 8px);
	block-size: calc(100% - 8px);
	border: solid 1px var(--focus);
	border-radius: var(--radius, 8px);
	box-sizing: border-box;
}

.note-container > .article > .main:hover :deep(.footer .button),
.note-container > .article > .main:focus-within :deep(.footer .button) {
	opacity: 1;
}

.note-context {
	position: relative;
	padding-block-start: 0;
	padding-inline: 32px;
	display: flex;
	flex-wrap: wrap;
	z-index: 1;
}

.article {
	position: relative;
	inline-size: 100%;
	box-sizing: border-box;
	overflow: clip;
	padding-block-start: 20px;
	padding-inline: 32px;
	padding-block-end: 10px;
}

.header-container {
	/* Grid instead of flex: the header column must take every pixel the avatar
	   leaves free. A flex row lets the note header collapse to its minimum
	   content width inside paginated lists, which truncates short names. */
	display: grid;
	grid-template-columns: auto minmax(0, 1fr);
	position: relative;
	z-index: 2;
}

.avatar {
	flex-shrink: 0;
	display: block;
	margin-inline-end: 14px;
	inline-size: var(--avatarSize, 48px);
	block-size: var(--avatarSize, 48px);
}

.header {
	min-inline-size: 0;
}

.main {
	flex: 1;
	min-inline-size: 0;
}

.body {
	margin-block-start: 0.7em;
}

@media (max-width: 500px) {
	.note-container {
		font-size: 0.975em;
	}

	.note-context {
		padding-inline: 16px;
	}

	.article {
		padding-block-start: 18px;
		padding-inline: 16px;
		padding-block-end: 8px;
	}

	.avatar {
		margin-inline-end: 10px;
		inline-size: var(--avatarSize, 46px);
		block-size: var(--avatarSize, 46px);
	}
}

@media (max-width: 350px) {
	.avatar {
		--avatarSize: 40px;
	}
}
</style>
