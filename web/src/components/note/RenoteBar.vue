// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkRenoteBar.vue
//
// Dropped: the multi-renote avatar list (needs notes/renotes; TODO) and the
// renote-menu on the timestamp for pure renotes by me (deletion goes through
// the footer's more-menu on the renote itself, since Cinderfish renders each
// renote as its own row).
<template>
	<div class="renote">
		<AppIcon name="renote" class="mark" />
		<span class="text">
			<MkAvatar class="avatar" :user="note.user" />
			<MkA class="name" :to="userPage" @click.stop>
				<MkUserName :user="note.user" />
			</MkA>
			{{ i18n.ts.renotedBy }}
		</span>
		<div class="info">
			<span class="time">
				<MkTime :time="note.createdAt" />
			</span>
			<Visibility :note="note" />
		</div>
	</div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Note } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import MkA from "../ui/MkA.vue";
import MkAvatar from "../ui/MkAvatar.vue";
import MkTime from "../ui/MkTime.vue";
import MkUserName from "../ui/MkUserName.vue";
import Visibility from "./Visibility.vue";
import { i18n } from "../../i18n";

const props = defineProps<{
	note: Note;
	appearNote: Note;
}>();

const userPage = computed(
	() => `/@${props.note.user.username}${props.note.user.host ? `@${props.note.user.host}` : ""}`,
);

</script>

<style scoped>
.renote {
	display: flex;
	align-items: center;
	white-space: pre;
	color: var(--renote);
	cursor: pointer;
}

.mark {
	margin-inline-end: 4px;
	flex-shrink: 0;
}

.avatar {
	inline-size: 1.2em;
	block-size: 1.2em;
	border-radius: 2em;
	overflow: hidden;
	margin-inline-end: 0.4em;
	flex-shrink: 0;
}

.text {
	overflow: hidden;
	flex-shrink: 1;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.name {
	font-weight: bold;
	color: inherit;
}

.info {
	margin-inline-start: auto;
	font-size: 0.9em;
	display: flex;
	align-items: center;
	flex-shrink: 0;
}

.time {
	flex-shrink: 0;
	color: inherit;
	display: inline-flex;
	align-items: center;
}
</style>
