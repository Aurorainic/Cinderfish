<!-- Ported from firefish@b425f61:packages/client/src/components/MkReactionsViewer.vue (AGPL-3.0-only) -->
<template>
	<div
		ref="reactionsEl"
		class="reactions-list swiper-no-swiping tdflqwzn"
		:class="{ isMe }"
	>
		<MkReactionsViewerReaction
			v-for="(count, reaction) in note.reactions"
			:key="reaction"
			:reaction="reaction"
			:count="count"
			:is-initial="initialReactions.has(reaction)"
			:note="note"
			@reacted="scrollToStart"
		/>
	</div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { Note } from "@/api/types";
import { me } from "@/account";
import MkReactionsViewerReaction from "@/components/MkReactionsViewer.reaction.vue";

const props = defineProps<{
	note: Note;
}>();

const reactionsEl = ref<HTMLElement | null>(null);

const initialReactions = new Set(Object.keys(props.note.reactions));

const isMe = computed(() => me.value != null && me.value.id === props.note.userId);

function scrollToStart(): void {
	reactionsEl.value?.scrollTo(0, 0);
}
</script>

<style lang="scss" scoped>
.reactions-list {
	margin-top: 0.2em;
	width: 100%;
	display: flex;
	overflow-x: auto;
	margin-inline: -24px;
	padding-inline: 22px 160px;
	mask: linear-gradient(
		to right,
		transparent,
		black 24px calc(100% - 160px),
		transparent
	);
	-webkit-mask: linear-gradient(
		to right,
		transparent,
		black 24px calc(100% - 160px),
		transparent
	);
	scrollbar-width: none;
	pointer-events: none;
	:deep(*) {
		pointer-events: all;
	}

	&::-webkit-scrollbar {
		display: none;
	}

	&:empty {
		display: none;
	}

	&.isMe {
		> span {
			cursor: default !important;
		}
	}
}
</style>
