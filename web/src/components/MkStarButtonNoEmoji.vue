<!-- Ported from firefish@b425f61:packages/client/src/components/MkStarButtonNoEmoji.vue (AGPL-3.0-only) -->
<template>
	<button
		v-tooltip.noDelay.bottom="t('_gallery.like')"
		class="button _button"
		:class="classes.root"
		@click.stop="toggleStar()"
	>
		<span v-if="!reacted">
			<i
				v-if="defaultReaction === '👍'"
				class="ph-thumbs-up ph-bold ph-lg"
			></i>
			<i
				v-else-if="defaultReaction === '❤️'"
				class="ph-heart ph-bold ph-lg"
			></i>
			<i v-else class="ph-star ph-bold ph-lg"></i>
		</span>
		<span v-else>
			<i
				v-if="defaultReaction === '👍'"
				class="ph-thumbs-up ph-bold ph-lg ph-fill"
				:class="classes.yellow"
			></i>
			<i
				v-else-if="defaultReaction === '❤️'"
				class="ph-heart ph-bold ph-lg ph-fill"
				:class="classes.red"
			></i>
			<i
				v-else
				class="ph-star ph-bold ph-lg ph-fill"
				:class="classes.yellow"
			></i>
		</span>
		<template v-if="count > 0"
			><p class="count">{{ count }}</p></template
		>
	</button>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Note } from "@/api/types";
import { ENDPOINTS } from "@/api/endpoints";
import { apiWithDialog } from "@/os";
import { pleaseLogin } from "@/scripts/please-login";
import { instance } from "@/instance";
import { t } from "@/i18n";

/**
 * Ported from Firefish. The hover tooltip listed the reactors through
 * `MkUsersTooltip`, which is not part of this wave's component set; the count
 * already carries the same information, so only the tooltip is dropped.
 *
 * `reacted` is passed in by the caller (MkNote), because a plain `Note` does
 * not carry `myReaction` — see `MkReactionsViewer.reaction.vue`.
 */
const props = defineProps<{
	note: Note;
	count: number;
	reacted: boolean;
}>();

/** Synthesised by @/instance: Cinderfish's /api/meta has no defaultReaction. */
const defaultReaction = computed(() => instance.defaultReaction);

/** CSS modules are not used elsewhere in the port; a plain class map is. */
const classes = {
	root: "mk-star-button-no-emoji",
	yellow: "yellow",
	red: "red",
};

function toggleStar(): void {
	pleaseLogin();

	if (!props.reacted) {
		void apiWithDialog(ENDPOINTS.notesReactionsCreate, {
			noteId: props.note.id,
			reaction: defaultReaction.value,
		});
	} else {
		void apiWithDialog(ENDPOINTS.notesReactionsDelete, {
			noteId: props.note.id,
		});
	}
}
</script>

<style lang="scss" scoped>
.yellow {
	color: var(--warn);
}

.red {
	color: var(--error);
}
</style>
