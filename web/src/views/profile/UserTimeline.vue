// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/index.timeline.vue
//
// Adaptations: MkTab becomes the segmented control below; the pagination's
// endpoint name becomes a fetcher over web/src/api/client.ts, and the
// include flag maps onto the backend's withReplies parameter (which the
// backend defaults to true, matching Firefish's notes-and-replies shape).
<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "../../api/client";
import type { Note, UserDetailed } from "../../api/types";
import MkPagination from "../../components/ui/MkPagination.vue";
import NoteItem from "../../components/note/Note.vue";
import { i18n } from "../../i18n";

const props = defineProps<{ user: UserDetailed }>();

type Include = null | "replies" | "files";
const include = ref<Include>(null);

// "帖子与回复" / "仅文件" are Firefish's notesAndReplies/withFiles labels;
// the shared dictionary does not carry those keys yet (it is not in this
// task's scope), so they are inlined here.
const OPTIONS: { value: Include; label: string }[] = [
	{ value: null, label: i18n.ts.notes },
	{ value: "replies", label: "帖子与回复" },
	{ value: "files", label: "仅文件" },
];

const pagination = computed(() => ({
	endpoint: (p: Record<string, unknown>) =>
		api.usersNotes({
			userId: props.user.id,
			limit: Number(p.limit) || 10,
			untilId: typeof p.untilId === "string" ? p.untilId : undefined,
			withReplies: include.value === "replies",
			withFiles: include.value === "files",
		}),
	limit: 10,
}));

function onDeleted(_id: string): void {
	// MkPagination folds removals through its own removeItem; a deleted note
	// disappears on the next reload of the list.
}
</script>

<template>
	<div class="user-timeline">
		<div class="tab" role="tablist">
			<button
				v-for="option in OPTIONS"
				:key="option.value ?? 'all'"
				type="button"
				class="tab-item"
				:class="{ active: include === option.value }"
				role="tab"
				:aria-selected="include === option.value"
				@click="include = option.value"
			>
				{{ option.label }}
			</button>
		</div>
		<MkPagination :key="include ?? 'all'" :pagination="pagination">
			<template #default="{ items }">
				<NoteItem
					v-for="note in items as Note[]"
					:key="note.id"
					:note="note"
					@deleted="onDeleted"
				/>
			</template>
		</MkPagination>
	</div>
</template>

<style scoped>
/* Firefish's MkTab default variant: one equal-width button per option,
   radius --radius, the selected one accent text on --accentedBg. */
.tab {
	display: flex;
	padding-block: 10px;
	padding-inline: 8px;
	font-size: 90%;
}

.tab-item {
	flex: 1;
	margin-inline: 8px;
	padding-block: 10px;
	padding-inline: 8px;
	border: 0;
	border-radius: var(--radius);
	background: transparent;
	color: var(--fg);
	font: inherit;
	cursor: pointer;
	transition: color 0.1s ease, background 0.1s ease;
}

.tab-item:hover {
	color: var(--fgHighlighted);
	background: var(--panelHighlight);
}

.tab-item.active {
	color: var(--accent);
	background: var(--accentedBg);
}
</style>
