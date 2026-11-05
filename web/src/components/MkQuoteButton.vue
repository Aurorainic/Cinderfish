<!-- Ported from firefish@b425f61:packages/client/src/components/MkQuoteButton.vue (AGPL-3.0-only) -->
<template>
	<button
		v-if="canRenote && defaultStore.state.seperateRenoteQuote"
		v-tooltip.noDelay.bottom="t('quote')"
		class="eddddedb _button"
		@click.stop="quote()"
	>
		<i class="ph-quotes ph-bold ph-lg"></i>
	</button>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Note, NoteDetailed } from "@/api/types";
import { me } from "@/account";
import { post } from "@/os/post";
import { pleaseLogin } from "@/scripts/please-login";
import { defaultStore } from "@/store";
import { t } from "@/i18n";

/**
 * Ported from Firefish. Quoting opens the composer with `renote`, which is the
 * same path Firefish's `os.post({ renote })` takes.
 */
const props = defineProps<{
	note: Note;
}>();

const canRenote = computed(
	() =>
		["public", "home"].includes(props.note.visibility) ||
		props.note.userId === me.value?.id,
);

function quote(): void {
	pleaseLogin();
	void post({ renote: props.note as unknown as NoteDetailed });
}
</script>

<style lang="scss" scoped>
.eddddedb {
	display: inline-block;
	height: 32px;
	margin: 2px;
	padding: 0 6px;
	border-radius: 4px;

	&.renoted {
		background: var(--accent);
	}

	> .count {
		display: inline;
		margin-left: 8px;
		opacity: 0.7;
	}
}
</style>
