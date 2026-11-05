<!-- Ported from firefish@b425f61:packages/client/src/components/MkReactionsViewer.reaction.vue (AGPL-3.0-only) -->
<template>
	<button
		v-if="count > 0"
		ref="buttonRef"
		class="hkzvhatu _button"
		:class="{
			reacted: myReaction === reaction,
			canToggle,
			newlyAdded: !isInitial,
		}"
		@click.stop="toggleReaction()"
	>
		<MkReactionIcon
			class="icon"
			:reaction="reaction"
			:custom-emojis="note.emojis"
		/>
		<span class="count">{{ count }}</span>
	</button>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { Note } from "@/api/types";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { me } from "@/account";
import { apiWithDialog, popup } from "@/os";
import { useTooltip } from "@/scripts/use-tooltip";
import MkReactionIcon from "@/components/MkReactionIcon.vue";
import MkReactionsViewerDetails from "@/components/MkReactionsViewer.details.vue";

/**
 * Ported from Firefish. Behaviour kept as-is: clicking a reaction the viewer
 * already left removes it, clicking another one switches to it, and remote
 * custom emoji reactions (`:name@host:`) cannot be toggled.
 *
 * Degradations:
 *   - `myReaction` only exists on `NoteDetailed`; a plain `Note` (e.g. a
 *     streaming event, which is packed anonymously) never renders as reacted.
 *   - The ripple popup uses `MkRipple`, which is outside this task's scope.
 *
 * The reaction *picker* belongs to the note footer (note-core's MkNote), so it
 * is deliberately not duplicated here.
 */
const props = defineProps<{
	reaction: string;
	count: number;
	isInitial: boolean;
	note: Note;
}>();

const emit = defineEmits<{
	reacted: [];
}>();

const buttonRef = ref<HTMLElement | null>(null);

const myReaction = computed<string | null>(() => {
	const value = (props.note as { myReaction?: string | null }).myReaction;
	return value ?? null;
});

const canToggle = computed(
	() => !props.reaction.match(/@\w/) && me.value != null,
);

const toggleReaction = (): void => {
	if (!canToggle.value) return;

	const oldReaction = myReaction.value;
	if (oldReaction != null) {
		void apiWithDialog(ENDPOINTS.notesReactionsDelete, {
			noteId: props.note.id,
		}).then(() => {
			if (oldReaction !== props.reaction) {
				void apiWithDialog(ENDPOINTS.notesReactionsCreate, {
					noteId: props.note.id,
					reaction: props.reaction,
				});
			}
		});
	} else {
		void apiWithDialog(ENDPOINTS.notesReactionsCreate, {
			noteId: props.note.id,
			reaction: props.reaction,
		});
		emit("reacted");
	}
};

useTooltip(
	buttonRef,
	async (showing) => {
		const reactions = await api(ENDPOINTS.notesReactions, {
			noteId: props.note.id,
			type: props.reaction,
			limit: 11,
		});

		const users = reactions.map((x) => x.user);

		popup(
			MkReactionsViewerDetails,
			{
				showing,
				reaction: props.reaction,
				emojis: props.note.emojis,
				users,
				count: props.count,
				targetElement: buttonRef.value,
			},
			{},
			"closed",
		);
	},
	100,
);
</script>

<style lang="scss" scoped>
.hkzvhatu {
	position: relative;
	display: inline-block;
	height: 32px;
	margin-block: 2px;
	padding: 0 8px;
	pointer-events: all;
	min-width: max-content;
	&::before {
		content: "";
		position: absolute;
		inset: 0 2px;
		border-radius: 4px;
		z-index: -1;
	}
	&.newlyAdded {
		animation: scaleInSmall 0.3s cubic-bezier(0, 0, 0, 1.2);
		:deep(.mk-emoji) {
			animation: scaleIn 0.4s cubic-bezier(0.7, 0, 0, 1.5);
		}
	}
	:deep(.mk-emoji) {
		transition: transform 0.4s cubic-bezier(0, 0, 0, 6);
	}
	&.reacted :deep(.mk-emoji) {
		transition: transform 0.4s cubic-bezier(0, 0, 0, 1);
	}
	&:active {
		:deep(.mk-emoji) {
			transition: transform 0.4s cubic-bezier(0, 0, 0, 1);
			transform: scale(0.85);
		}
	}
	&.canToggle {
		&::before {
			background: rgba(0, 0, 0, 0.05);
		}
		&:hover:not(.reacted)::before {
			background: rgba(0, 0, 0, 0.1);
		}
	}

	&:not(.canToggle) {
		cursor: default;
	}

	&.reacted {
		order: -1;
		&::before {
			background: var(--accent);
		}

		> .count {
			color: var(--fgOnAccent);
			font-weight: 600;
		}

		> .icon {
			filter: drop-shadow(0 0 2px rgba(0, 0, 0, 0.5));
		}
	}

	> .count {
		font-size: 0.9em;
		line-height: 32px;
		margin: 0 0 0 4px;
	}
}
</style>
