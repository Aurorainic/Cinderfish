// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteFooter.vue
//
// Dropped versus the original, with reasons:
// - Translation button (MkNoteTranslation): no translation endpoint.
// - Star-button split (enableEmojiReactions): Cinderfish always has emoji
//   reactions (⭐ default), so the star-only variant is unnecessary; the
//   reaction button opens the shared reaction picker.
// - The per-note menu is trimmed to the endpoints Cinderfish has
//   (get-note-menu.ts port below); favorites, clips, watch, mute, translate,
//   edit and abuse report stay out until their endpoints exist.
<template>
	<footer ref="el" class="footer" tabindex="-1">
		<ReactionsViewer :note="note" :my-reaction="note.myReaction" @reacted="onReacted" />
		<button type="button" class="button" :title="i18n.ts.reply" @click.stop="reply()">
			<AppIcon name="reply" />
			<p v-if="note.repliesCount > 0 && !detailedView" class="count">{{ note.repliesCount }}</p>
		</button>
		<RenoteButton class="button" :note="note" :count="note.renoteCount" :detailed-view="detailedView" @renoted="emit('renoted')" />
		<button
			ref="reactButton"
			type="button"
			class="button"
			:class="{ reacted: note.myReaction != null }"
			:title="note.myReaction != null ? i18n.ts.unrenote : i18n.ts.reaction"
			@click.stop="react()"
		>
			<AppIcon :name="note.myReaction != null ? 'close' : 'smiley'" />
			<p v-if="reactionCount > 0" class="count">{{ reactionCount }}</p>
		</button>
		<button
			ref="quoteButton"
			type="button"
			class="button"
			:title="i18n.ts.quote"
			@click.stop="quote()"
		>
			<AppIcon name="quote" />
		</button>
		<button ref="menuButton" type="button" class="button" :title="i18n.ts.more" @click.stop="menu()">
			<AppIcon name="more" />
		</button>
	</footer>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import ReactionsViewer from "./ReactionsViewer.vue";
import RenoteButton from "./RenoteButton.vue";
import { i18n } from "../../i18n";
import { openPostForm, openReactionPicker, popupMenu } from "../../os";
import { pleaseLogin } from "./please-login";
import { getNoteMenu } from "./get-note-menu";

const props = defineProps<{
	note: Note;
	detailedView?: boolean;
}>();

const emit = defineEmits<{
	"event:focus": [];
	"event:blur": [];
	deleted: [];
	renoted: [];
}>();

const el = ref<HTMLElement | null>(null);
const reactButton = ref<HTMLButtonElement | null>(null);
const menuButton = ref<HTMLButtonElement | null>(null);

const reactionCount = computed(() =>
	Object.values(props.note.reactions).reduce((sum, val) => sum + val, 0),
);

function reply(): void {
	pleaseLogin();
	openPostForm({ reply: props.note });
}

function react(): void {
	pleaseLogin();
	emit("event:blur");
	if (props.note.myReaction != null) {
		// Undo the existing reaction, as the original's undoReact does.
		api.unreact(props.note.id).catch(() => undefined);
		return;
	}
	openReactionPicker((reaction) => {
		api.react(props.note.id, reaction).catch(() => undefined);
	}, reactButton.value);
}

function quote(): void {
	pleaseLogin();
	openPostForm({ renote: props.note });
}

async function menu(): Promise<void> {
	const isDeleted = ref(false);
	await popupMenu(getNoteMenu({ note: props.note, isDeleted }), menuButton.value, {
		align: "right",
	});
	if (isDeleted.value) emit("deleted");
}

function onReacted() {
	// The ReactionsViewer keeps its own optimistic copy; re-render the count.
	el.value?.scrollTo?.(0, 0);
}

defineExpose({
	reply,
	react,
	menu,
	focus: () => el.value?.focus(),
});
</script>

<style scoped>
.footer {
	position: relative;
	z-index: 2;
	display: flex;
	flex-wrap: wrap;
	margin-block-start: 0.4em;
}

.footer > :deep(.button) {
	position: relative;
	margin: 0;
	padding: 8px;
	opacity: 0.7;

	flex-grow: 1;
	max-inline-size: 3.5em;
	inline-size: max-content;
	min-inline-size: max-content;
	block-size: auto;
	transition: opacity 0.2s;
	border: 0;
	background: transparent;
	color: inherit;
	cursor: pointer;
	font-size: 0.9em;
	display: inline-flex;
	align-items: center;
}

.footer > :deep(.button)::before {
	content: "";
	position: absolute;
	inset: 0;
	inset-block-end: 2px;
	background: var(--panel);
	z-index: -1;
	transition: background 0.2s;
}

.footer > :deep(.button:first-of-type) {
	margin-inline-start: -0.5em;
}

.footer > :deep(.button:first-of-type)::before {
	border-radius: 100px 0 0 100px;
}

.footer > :deep(.button:last-of-type)::before {
	border-radius: 0 100px 100px 0;
}

.footer > :deep(.button):hover {
	color: var(--fgHighlighted);
}

.footer > :deep(.button):disabled {
	opacity: 0.3;
}

.footer .count {
	display: inline;
	margin: 0;
	margin-inline-start: 8px;
	opacity: 0.7;
}

.footer .reacted {
	color: var(--accent);
}
</style>
