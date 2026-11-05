<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.timeline.vue (AGPL-3.0-only) -->
<template>
	<MkNotes :key="listKey" :pagination="pagination" />
</template>

<script lang="ts" setup>
// Firefish's component carries its own notes / notes+replies / with-files tab.
// The profile page owns the tab bar in Cinderfish (contract task: five tabs),
// so the selection arrives as props and this component only builds the
// `users/notes` paging object.
//
// `withReplies` is always passed explicitly: the backend default is true, the
// opposite of upstream (contract §5), and relying on a default is how the two
// directions get mixed up.
import { computed } from "vue";
import type { UserDetailed } from "@/api/types";
import type { Paging } from "@/components/ui";
import MkNotes from "@/components/MkNotes.vue";

const props = withDefaults(
	defineProps<{
		user: UserDetailed;
		withReplies: boolean;
		withFiles?: boolean;
	}>(),
	{
		withFiles: false,
	},
);

const listKey = computed<string>(
	() => `${props.user.id}:${props.withReplies}:${props.withFiles}`,
);

const pagination = computed<Paging>(() => ({
	endpoint: "users/notes",
	limit: 10,
	params: {
		userId: props.user.id,
		withReplies: props.withReplies,
		withFiles: props.withFiles,
	},
}));
</script>
