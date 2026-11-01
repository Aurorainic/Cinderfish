// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkReactionsViewer.vue
// and MkReactionsViewer.reaction.vue (merged: the per-reaction toggle, its
// newly-added animation and the strip container live in one file here).
//
// Reaction chips with counts; clicking toggles the viewer's own reaction via
// notes/reactions/create + delete. The hover tooltip listing reactors needs
// notes/reactions and os.popup, which the capability matrix defers: TODO add
// the reactor list tooltip when those endpoints land.
<template>
	<div v-if="entries.length" ref="reactionsEl" class="reactions-list">
		<button
			v-for="[reaction, count] in entries"
			:key="reaction"
			type="button"
			class="reaction"
			:class="{
				reacted: myReaction === reaction,
				canToggle,
				newlyAdded: !initialReactions.has(reaction),
			}"
			:title="`${emojiName(reaction)} ${count}`"
			@click.stop="toggleReaction(reaction)"
		>
			<span class="icon">{{ reaction }}</span>
			<span class="count">{{ count }}</span>
		</button>
	</div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import { emojiName } from "../../lib/emoji-catalog";
import { isSignedIn, me } from "../../me";

const props = defineProps<{
	note: Note;
	// The optimistic reaction state owned by useNote(); the original reads
	// note.myReaction directly, but Cinderfish keeps mutations in the row.
	myReaction?: string;
}>();

const emit = defineEmits<{
	// Raised after a successful toggle so the row can refresh its counters.
	(ev: "reacted"): void;
}>();

const reactionsEl = ref<HTMLElement | null>(null);

const initialReactions = new Set(Object.keys(props.note.reactions));

const entries = computed(() =>
	Object.entries(props.note.reactions).filter(([, count]) => count > 0),
);

const canToggle = computed(() => isSignedIn(me) && !/@/.test(Object.keys(props.note.reactions)[0] ?? ""));

async function toggleReaction(reaction: string) {
	if (!isSignedIn(me) || /@/.test(reaction)) return;
	const oldReaction = props.myReaction;
	try {
		if (oldReaction) {
			await api.unreact(props.note.id);
			if (oldReaction !== reaction) {
				await api.react(props.note.id, reaction);
			}
		} else {
			await api.react(props.note.id, reaction);
			reactionsEl.value?.scrollTo(0, 0);
		}
		emit("reacted");
	} catch {
		// The backend rejected the toggle (visibility, missing note, …); the
		// next note re-render restores the authoritative counters.
	}
}
</script>

<style scoped>
.reactions-list {
	margin-block-start: 0.2em;
	inline-size: 100%;
	display: flex;
	flex-wrap: wrap;
	gap: 2px;
	overflow-x: auto;
	scrollbar-width: none;
}

.reactions-list::-webkit-scrollbar {
	display: none;
}

.reactions-list:empty {
	display: none;
}

.reaction {
	position: relative;
	display: inline-block;
	block-size: 32px;
	margin-block: 2px;
	padding-block: 0;
	padding-inline: 8px;
	min-inline-size: max-content;
	border: 0;
	background: transparent;
	cursor: pointer;
	font-size: inherit;
}

.reaction::before {
	content: "";
	position: absolute;
	inset: 0 2px;
	border-radius: 4px;
	z-index: -1;
	background: color-mix(in srgb, currentColor 5%, transparent);
}

.reaction.canToggle:hover:not(.reacted)::before {
	background: color-mix(in srgb, currentColor 10%, transparent);
}

.reaction:not(.canToggle) {
	cursor: default;
}

.reaction .icon {
	font-size: 1.5em;
	line-height: 32px;
	transition: transform 0.4s cubic-bezier(0, 0, 0, 6);
}

.reaction:active .icon {
	transform: scale(0.85);
}

.reaction.newlyAdded {
	animation: scaleInSmall 0.3s cubic-bezier(0, 0, 0, 1.2);
}

.reaction.newlyAdded .icon {
	animation: scaleIn 0.4s cubic-bezier(0.7, 0, 0, 1.5);
}

.reaction.reacted {
	order: -1;
}

.reaction.reacted::before {
	background: var(--accent);
}

.reaction.reacted .count {
	color: var(--fgOnAccent);
	font-weight: 600;
}

.reaction.reacted .icon {
	filter: drop-shadow(0 0 2px rgba(0, 0, 0, 0.5));
}

.reaction .count {
	font-size: 0.9em;
	line-height: 32px;
	margin-inline-start: 4px;
}

@keyframes scaleInSmall {
	from {
		transform: scale(0.6);
	}
	to {
		transform: scale(1);
	}
}

@keyframes scaleIn {
	from {
		transform: scale(0.4);
	}
	to {
		transform: scale(1);
	}
}
</style>
