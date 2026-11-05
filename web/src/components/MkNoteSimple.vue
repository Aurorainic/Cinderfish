<!-- Ported from firefish@b425f61:packages/client/src/components/MkNoteSimple.vue (AGPL-3.0-only) -->
<template>
	<div v-size="{ min: [350, 500] }" class="yohlumlk">
		<MkAvatar class="avatar" :user="note.user" />
		<div class="main">
			<MkNoteHeader class="header" :note="note" :mini="true" />
			<div class="body">
				<MkSubNoteContent class="text" :note="note" />
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
import type { Note, NoteDetailed } from "@/api/types";
import MkNoteHeader from "./MkNoteHeader.vue";
import MkSubNoteContent from "./MkSubNoteContent.vue";

type SimpleNote = Note &
	Partial<
		Pick<
			NoteDetailed,
			"renote" | "reply" | "poll" | "myReaction" | "hasPoll"
		>
	>;

defineProps<{
	note: SimpleNote;
	pinned?: boolean;
}>();
</script>

<style lang="scss" scoped>
.yohlumlk {
	display: flex;
	margin: 0;
	padding: 0;
	overflow: clip;
	font-size: 0.95em;

	&.min-width_350px {
		> .avatar {
			margin: 0 10px 0 0;
			width: 44px;
			height: 44px;
		}
	}

	&.min-width_500px {
		> .avatar {
			margin: 0 12px 0 0;
			width: 48px;
			height: 48px;
		}
	}

	> .avatar {
		flex-shrink: 0;
		display: block;
		margin: 0 10px 0 0;
		width: 40px;
		height: 40px;
		border-radius: 8px;
	}

	> .main {
		flex: 1;
		min-width: 0;

		> .header {
			margin-bottom: 2px;
		}
	}
}
</style>
