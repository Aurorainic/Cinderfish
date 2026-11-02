// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNoteDetailed.vue
//
// Kept: the note itself through Note.vue's detailed rendering and the
// underline tab strip that lists replies, renotes and quotes through
// MkPagination.
//
// Adaptations versus the original:
// - The conversation above the note (notes/conversation) and the reacted-users
//   tab (notes/reactions) stay out: neither endpoint exists in Cinderfish. The
//   reply context comes from Note.vue's NoteSub row instead.
// - The renote and quote tabs share notes/renotes because the backend has no
//   `filter` parameter; quotes are the returned notes carrying text and pure
//   renotes are the rest, the same distinction Firefish gets from the server.
//   note.quoteCount is not packed either, so the quote tab has no count and is
//   offered whenever the note has renotes.
// - Reply and quote rows render through Note.vue with detailed-view (the fork
//   passes detailed-view to MkNoteSub as well), which keeps the row's own
//   context strip out; pure renotes keep the normal row so their renote bar
//   names the account that renoted. The fork instead lists MkUserCardMini
//   cards, which Cinderfish has no equivalent for.
// - The `clips` tab, word soft-mute, plugins (noteViewInterruptors) and
//   translation are dropped with the features they need.
<template>
	<div ref="el" class="note-detailed">
		<NoteRow :note="note" detailed-view />

		<nav class="tabs" role="tablist" aria-label="笔记分页">
			<button
				v-for="option in tabs"
				:key="option.key"
				type="button"
				class="tab"
				role="tab"
				:aria-selected="tab === option.key"
				@click="tab = option.key"
			>
				{{ option.label }}
			</button>
		</nav>

		<MkPagination
			v-if="tab === 'replies' && note.repliesCount > 0"
			:pagination="repliesPagination"
		>
			<template #empty>
				<p class="empty">没有回复。</p>
			</template>
			<template #default="{ items }">
				<NoteRow
					v-for="reply in items as Note[]"
					:key="reply.id"
					class="row"
					:note="reply"
					detailed-view
				/>
			</template>
		</MkPagination>

		<MkPagination
			v-else-if="tab === 'renotes' && note.renoteCount > 0"
			:pagination="renotesPagination"
		>
			<template #empty>
				<p class="empty">没有被转发。</p>
			</template>
			<template #default="{ items }">
				<NoteRow
					v-for="renote in pureRenotes(items as Note[])"
					:key="renote.id"
					class="row"
					:note="renote"
				/>
			</template>
		</MkPagination>

		<MkPagination
			v-else-if="tab === 'quotes' && note.renoteCount > 0"
			:pagination="quotesPagination"
		>
			<template #empty>
				<p class="empty">没有被引用。</p>
			</template>
			<template #default="{ items }">
				<NoteRow
					v-for="quote in quotesIn(items as Note[])"
					:key="quote.id"
					class="row"
					:note="quote"
					detailed-view
				/>
			</template>
		</MkPagination>
	</div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import MkPagination from "../ui/MkPagination.vue";
import NoteRow from "./Note.vue";
import { i18n } from "../../i18n";

const props = defineProps<{
	note: Note;
	pinned?: boolean;
}>();

type Tab = "replies" | "renotes" | "quotes";

const tab = ref<Tab>("replies");

// The original distinguishes singular/plural only for western languages; the
// Chinese dictionary shares one form, so the count is simply prefixed.
const wordWithCount = (count: number, singular: string, _plural: string) =>
	count === 0 ? singular : `${count} ${singular}`;

const tabs = computed<{ key: Tab; label: string }[]>(() => {
	const list: { key: Tab; label: string }[] = [
		{
			key: "replies",
			label: wordWithCount(props.note.repliesCount, i18n.ts.reply, i18n.ts.replies),
		},
	];
	if (props.note.renoteCount > 0) {
		list.push({
			key: "renotes",
			label: wordWithCount(props.note.renoteCount, i18n.ts.renote, i18n.ts.renotes),
		});
		// notes/renotes does not report a quote count, so the label stays bare.
		list.push({ key: "quotes", label: i18n.ts.quotes });
	}
	return list;
});

const REPLIES_LIMIT = 10;
const RENOTES_LIMIT = 30;

// notes/replies pages oldest-first in the fork (ascending: true); the merged
// endpoint honours sinceId for that direction.
const repliesPagination = {
	endpoint: (p: Record<string, unknown>) =>
		api.notesReplies({
			noteId: props.note.id,
			limit: Number(p.limit) || REPLIES_LIMIT,
			sinceId: typeof p.sinceId === "string" ? p.sinceId : undefined,
			untilId: typeof p.untilId === "string" ? p.untilId : undefined,
		}),
	limit: REPLIES_LIMIT,
	ascending: true,
};

// Both the renote and quote lists page the same endpoint newest-first; each
// tab filters the fetched page to its own kind.
const renotesPagination = {
	endpoint: (p: Record<string, unknown>) =>
		api.notesRenotes({
			noteId: props.note.id,
			limit: Number(p.limit) || RENOTES_LIMIT,
			sinceId: typeof p.sinceId === "string" ? p.sinceId : undefined,
			untilId: typeof p.untilId === "string" ? p.untilId : undefined,
		}),
	limit: RENOTES_LIMIT,
};

const quotesPagination = renotesPagination;

// A pure renote has no text of its own; a quote is a renote that carries the
// author's comment, which is the split notes/renotes' `filter` would make
// server-side.
function pureRenotes(items: Note[]): Note[] {
	return items.filter((item) => item.text == null);
}

function quotesIn(items: Note[]): Note[] {
	return items.filter((item) => item.text != null);
}
</script>

<style scoped>
.note-detailed {
	overflow: clip;
}

/* Firefish's MkTab underline variant: a row of borderless labels whose active
   one takes the accent colour and a bottom rule. */
.tabs {
	display: flex;
	padding-block: 0;
	padding-inline: 32px;
	overflow-x: auto;
	overflow-y: hidden;
	scrollbar-width: none;
}

.tabs::-webkit-scrollbar {
	display: none;
}

.tab {
	flex: none;
	padding-block: 10px;
	padding-inline: 16px;
	border: 0;
	border-block-end: 2px solid transparent;
	border-radius: 0;
	background: none;
	color: var(--fgTransparentWeak);
	font: inherit;
	font-size: 0.85em;
	white-space: nowrap;
	cursor: pointer;
	transition: color 0.1s ease, border-color 0.1s ease;
}

.tab:hover {
	color: var(--fgHighlighted);
}

.tab:focus-visible {
	outline: 2px solid var(--accent);
	outline-offset: -4px;
}

.tab[aria-selected="true"] {
	border-block-end-color: var(--accent);
	color: var(--accent);
	font-weight: 700;
}

.row {
	border-block-start: 1px solid var(--divider);
}

.empty {
	margin: 0;
	padding: 24px 32px;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
	text-align: center;
}

@media (max-width: 500px) {
	.tabs {
		padding-inline: 16px;
	}

	.tab {
		padding-inline: 12px;
	}

	.empty {
		padding-inline: 16px;
	}
}
</style>
