// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkRenoteButton.vue
//
// The renote (boost) footer action. Clicking opens the same visibility menu
// as the original (public / home / followers / localOnly), because Cinderfish
// supports notes/create with visibility; the followers-renote option is only
// offered to the author, matching canRenote. The renoters tooltip needs
// notes/renotes, which Cinderfish lacks (TODO). Ripple animation and
// vibration are dropped (no MkRipple / vibrate runtime).
<template>
	<button
		v-if="canRenote"
		ref="buttonRef"
		type="button"
		class="button"
		:title="i18n.ts.renote"
		@click.stop="renote()"
	>
		<AppIcon name="renote" />
		<p v-if="count > 0 && !detailedView" class="count">{{ count }}</p>
	</button>
	<button v-else type="button" class="button" :title="i18n.ts.cantRenote" disabled>
		<AppIcon name="renote" />
	</button>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import { i18n } from "../../i18n";
import { me } from "../../me";
import { popupMenu } from "../../os";
import type { MenuItem } from "../../types/menu";

const props = defineProps<{
	note: Note;
	count: number;
	detailedView?: boolean;
}>();

const buttonRef = ref<HTMLButtonElement | null>(null);

const canRenote = computed(
	() =>
		(["public", "home"].includes(props.note.visibility) ||
			props.note.userId === me?.id),
);

const emit = defineEmits<{
	renoted: [];
}>();

function renoteAction(visibility: Note["visibility"], localOnly = false): MenuItem {
	return {
		text:
			visibility === "public"
				? i18n.ts.renote
				: `${i18n.ts.renote} (${i18n.ts._visibility[visibility]})`,
		icon: visibility === "public" ? "ph-rocket-launch" : visibility === "home" ? "ph-house" : "ph-lock",
		action: async () => {
			await api.createNote({ renoteId: props.note.id, visibility, localOnly });
			emit("renoted");
		},
	};
}

async function renote() {
	if (!me) return;
	const items: MenuItem[] = [];
	if (props.note.visibility === "public") {
		items.push(renoteAction("public"));
		items.push(renoteAction("home"));
	} else {
		items.push(renoteAction(props.note.visibility));
	}
	if (canRenote.value) {
		items.push({
			text: `${i18n.ts.renote} (${i18n.ts.local})`,
			icon: "ph-users",
			action: async () => {
				await api.createNote({
					renoteId: props.note.id,
					visibility: props.note.visibility,
					localOnly: true,
				});
				emit("renoted");
			},
		});
	}
	// The original appends the quote action here when
	// seperateRenoteQuote is off; Cinderfish keeps quote as its own footer
	// button (seperateRenoteQuote default), so the menu stays renote-only.
	await popupMenu(items, buttonRef.value, { align: "center", anchor: { x: "center", y: "bottom" } });
}
</script>

<style scoped>
.button {
	border: 0;
	background: transparent;
	padding: 8px;
	cursor: pointer;
	color: inherit;
	display: inline-flex;
	align-items: center;
}

.button:disabled {
	opacity: 0.5;
	cursor: default;
}

.count {
	display: inline;
	margin: 0;
	margin-inline-start: 8px;
	opacity: 0.7;
}
</style>
