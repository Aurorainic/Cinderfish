// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteHeader.vue
//
// The header carries no avatar: Note.vue draws it in .header-container, exactly
// like MkNote.vue does, so this component only owns the name, handle, time,
// visibility and instance ticker.
//
// Adaptations:
// - MkAcct is inlined as an @handle string (Cinderfish has no MkAcct runtime).
// - The instance ticker shows only when the account is remote, because
//   UserLite carries no software name or theme colour.
// - `avatar` / `radius` / `detail` props are accepted for call-site
//   compatibility with older components and ignored.
<template>
	<header class="cinderfish-note-header" :class="{ mini }">
		<div class="user-info">
			<div class="who">
				<MkA v-if="note.user" class="name" :to="userPage">
					<MkUserName :user="note.user" />
					<span v-if="note.user.isBot" class="is-bot">bot</span>
				</MkA>
				<div class="username">{{ acct }}</div>
			</div>
			<div class="meta">
				<div class="info">
					<MkA class="created-at" :to="`/notes/${note.id}`">
						<MkTime :time="note.createdAt" />
					</MkA>
					<Visibility :note="note" />
				</div>
				<InstanceTicker v-if="showTicker" class="ticker" :instance="note.user.instance" />
			</div>
		</div>
	</header>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Note } from "../../api/types";
import MkA from "../ui/MkA.vue";
import MkTime from "../ui/MkTime.vue";
import MkUserName from "../ui/MkUserName.vue";
import InstanceTicker from "./InstanceTicker.vue";
import Visibility from "./Visibility.vue";
import { defaultStore } from "../../store/firefish-store";

const props = withDefaults(
	defineProps<{
		note: Note;
		mini?: boolean;
		pinned?: boolean;
		/** Deprecated: the avatar lives in the caller. */
		avatar?: boolean;
		/** Deprecated: the avatar lives in the caller. */
		radius?: "50%" | "8px";
		/** Deprecated: NoteContent decides how much detail to show. */
		detail?: boolean;
	}>(),
	{ mini: false, pinned: false, avatar: true, radius: "50%", detail: false },
);

const userPage = computed(
	() => `/@${props.note.user.username}${props.note.user.host ? `@${props.note.user.host}` : ""}`,
);

const acct = computed(
	() => `@${props.note.user.username}${props.note.user.host ? `@${props.note.user.host}` : ""}`,
);

const showTicker = computed(() => {
	const mode = defaultStore.state.instanceTicker;
	if (mode === "always") return true;
	if (mode === "remote") return Boolean(props.note.user.host);
	return false;
});
</script>

<style scoped>
.cinderfish-note-header {
	position: relative;
	z-index: 2;
	display: flex;
	align-items: center;
	white-space: nowrap;
	justify-self: flex-end;
	width: 100%;
	font-size: 0.8em;
}

.cinderfish-note-header > .user-info {
	inline-size: 0;
	flex-grow: 1;
	line-height: 1.5;
	display: flex;
	font-size: 1.2em;
}

.user-info > .who {
	flex-grow: 1;
	inline-size: 0;
	overflow: hidden;
	text-overflow: ellipsis;
}

.user-info > .meta {
	max-inline-size: 50%;
	display: flex;
	flex-direction: column;
	align-items: flex-end;
	gap: 0.3em 0.5em;
}

.name {
	font-weight: bold;
	color: var(--fgHighlighted);
}

.is-bot {
	margin-inline-start: 0.5em;
	padding: 0 0.4em;
	font-size: 0.7em;
	border: solid 1px var(--infoWarnFg);
	border-radius: 999px;
	color: var(--infoWarnFg);
}

.username {
	opacity: 0.7;
	font-size: 0.9em;
}

.info {
	display: flex;
	align-items: center;
	gap: 0.5em;
	opacity: 0.7;
}

.created-at {
	color: inherit;
}

.ticker {
	max-inline-size: 100%;
}

.mini {
	font-size: 0.75em;
}
</style>
