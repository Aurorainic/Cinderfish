<!-- Ported from firefish@b425f61:packages/client/src/components/MkNoteDetailed.vue (AGPL-3.0-only) -->
<template>
	<div
		v-if="!muted.muted"
		v-show="!isDeleted"
		v-hotkey="keymap"
		v-size="{ max: [500, 350, 300] }"
		class="lxwezrsl _block"
		:tabindex="!isDeleted ? '-1' : undefined"
		:class="{ renote: isRenote }"
	>
		<template v-if="conversation">
			<MkNoteSub
				v-for="conversationNote in conversation ?? []"
				:key="conversationNote.id"
				class="reply-to"
				:note="conversationNote"
				:detailedView="true"
			/>
		</template>
		<MkLoading v-else-if="note.reply" mini />
		<MkNoteSub
			v-if="note.reply"
			:note="note.reply"
			class="reply-to"
			:detailedView="true"
		/>

		<MkNote
			ref="noteEl"
			@contextmenu.stop="onContextmenu"
			tabindex="-1"
			:note="note"
			detailedView
		></MkNote>

		<CfTab
			v-model="tab"
			:tabs="tabOptions"
			style="underline"
			@update:modelValue="loadTab"
		/>

		<CfInfo v-if="tabError" class="tab-message" warn>{{ tabError }}</CfInfo>
		<div v-else-if="tabEmpty" class="tab-empty">{{ t("noNotes") }}</div>

		<template v-if="directReplies && tab === 'replies'">
			<MkNoteSub
				v-for="replyNote in directReplies"
				:key="replyNote.id"
				:note="replyNote"
				class="reply"
				:conversation="replies"
				:detailedView="true"
				:parentId="replyNote.id"
			/>
		</template>
		<MkLoading v-else-if="tab === 'replies' && directReplies == null" />

		<template v-if="directQuotes && tab === 'quotes'">
			<MkNoteSub
				v-for="quoteNote in directQuotes"
				:key="quoteNote.id"
				:note="quoteNote"
				class="reply"
				:conversation="replies"
				:detailedView="true"
				:parentId="quoteNote.id"
			/>
		</template>
		<MkLoading v-else-if="tab === 'quotes' && directQuotes == null" />

		<template v-if="tab === 'renotes' && renotes">
			<MkA
				v-for="item in renotes"
				:key="item.user.id"
				class="user-card-mini _panel"
				:to="userPage(item.user)"
			>
				<MkAvatar :user="item.user" class="avatar" />
				<MkUserName :user="item.user" class="name" />
				<MkAcct :user="item.user" class="acct" />
			</MkA>
		</template>
		<MkLoading v-else-if="tab === 'renotes' && renotes == null" />

		<template v-if="tab === 'reactions' && reactedUsers">
			<MkA
				v-for="item in reactedUsers"
				:key="item.id"
				class="reacted-user _panel"
				:to="userPage(item.user)"
			>
				<MkAvatar :user="item.user" class="avatar" />
				<MkUserName :user="item.user" class="name" />
				<MkAcct :user="item.user" class="acct" />
				<MkEmoji :emoji="item.type" :is-reaction="true" class="reaction" />
			</MkA>
		</template>
		<MkLoading v-else-if="tab === 'reactions' && reactedUsers == null" />
	</div>
	<div v-else class="_panel muted" @click="muted.muted = false">
		<small>{{ softMuteText }}</small>
	</div>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * clips tab dropped (`notes/clips` has no endpoint);
//   * `MkUserCardMini` / `MkReactedUsers` are not part of this wave, so both
//     lists are rendered inline with the same class names (the surrounding SCSS
//     hooks are kept);
//   * `stream.on("noteUpdated")` is dropped: the backend streams `subNote` events
//     only, and `useNoteCapture` inside MkNote already applies them;
//   * translation is dropped (no endpoint).
import { computed, onMounted, onUnmounted, onUpdated, ref, type Ref } from "vue";
import { api } from "@/api/client";
import type { Note, NoteDetailed, NoteReaction } from "@/api/types";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfTab from "@/components/ui/CfTab.vue";
import MkNote from "@/components/MkNote.vue";
import MkNoteSub from "@/components/MkNoteSub.vue";
import { pleaseLogin } from "@/scripts/please-login";
import { getWordSoftMute, type Muted } from "@/scripts/check-word-mute";
import { reactionPicker } from "@/scripts/reaction-picker";
import { buildNoteMenu } from "@/scripts/get-note-menu";
import { useNoteCapture } from "@/composables/use-note-capture";
import { deepClone } from "@/scripts/clone";
import { userPage } from "@/filters/user";
import { defaultStore } from "@/store/base";
import { me } from "@/account";
import { t } from "@/i18n";
import * as os from "@/os";

type DetailNote = Note &
	Partial<
		Pick<
			NoteDetailed,
			"reply" | "renote" | "poll" | "myReaction" | "hasPoll"
		>
	>;

const props = defineProps<{
	note: DetailNote;
	pinned?: boolean;
}>();

const tab = ref<string | number>("replies");

const note = ref(deepClone(props.note)) as Ref<DetailNote>;
const muted = ref<Muted>(
	getWordSoftMute(note.value, me.value, defaultStore.state.mutedWords),
);

const softMuteText = computed(() => {
	const what = muted.value.what;
	const key =
		what === "note"
			? "userSaysSomethingReason"
			: what === "reply"
			? "userSaysSomethingReasonReply"
			: what === "renote"
			? "userSaysSomethingReasonRenote"
			: what === "quote"
			? "userSaysSomethingReasonQuote"
			: "userSaysSomething";
	return t(key, {
		name: note.value.user.name ?? note.value.user.username,
		reason: muted.value.matched.join(", "),
	});
});

const isRenote =
	note.value.renote != null &&
	note.value.text == null &&
	note.value.fileIds.length === 0 &&
	note.value.poll == null;

const noteEl = ref<{
	focus: () => void;
	blur: () => void;
	scrollIntoView: () => void;
} | null>(null);
const reactButton = ref<HTMLElement | null>(null);
const isDeleted = ref(false);
const conversation = ref<DetailNote[] | null>([]);
const replies = ref<DetailNote[]>([]);
const directReplies = ref<DetailNote[] | null>(null);
const directQuotes = ref<DetailNote[] | null>(null);
const renotes = ref<DetailNote[] | null>(null);
const reactedUsers = ref<NoteReaction[] | null>(null);
let isScrolling = false;

const reactionsCount = Object.values(props.note.reactions).reduce(
	(x, y) => x + y,
	0,
);

// Tabs, in Firefish's order. A tab is offered when it has something to show,
// exactly as upstream; the renote/quote counts only arrive once
// `notes/children` resolves, so the list has to stay reactive instead of being
// decided once during setup.
const tabOptions = computed(() => {
	const tabs = [
		{
			value: "replies",
			label: t("_notification._types.reply"),
			count: note.value.repliesCount > 0 ? note.value.repliesCount : undefined,
		},
	];
	if (note.value.renoteCount > 0) {
		tabs.push({
			value: "renotes",
			label: t("_notification._types.renote"),
			count: note.value.renoteCount,
		});
	}
	if (reactionsCount > 0) {
		tabs.push({ value: "reactions", label: t("reaction"), count: reactionsCount });
	}
	if (directQuotes.value != null && directQuotes.value.length > 0) {
		tabs.push({
			value: "quotes",
			label: t("_notification._types.quote"),
			count: directQuotes.value.length,
		});
	}
	return tabs;
});

const tabError = ref<string | null>(null);

// Only report "empty" once the list has actually arrived; `null` is loading.
const tabEmpty = computed<boolean>(() => {
	switch (tab.value) {
		case "replies":
			return directReplies.value != null && directReplies.value.length === 0;
		case "quotes":
			return directQuotes.value != null && directQuotes.value.length === 0;
		case "renotes":
			return renotes.value != null && renotes.value.length === 0;
		case "reactions":
			return reactedUsers.value != null && reactedUsers.value.length === 0;
		default:
			return false;
	}
});

const keymap = {
	r: () => reply(true),
	"e|a|plus": () => react(true),
	esc: blur,
	"m|o": () => menu(true),
};

useNoteCapture({
	note: note as unknown as Ref<NoteDetailed>,
	isDeletedRef: isDeleted,
});

function reply(_viaKeyboard = false): void {
	pleaseLogin();
	// Firefish passes `animation: !viaKeyboard` too; the dialog owns its own
	// transition in this port.
	void os.post({ reply: note.value as unknown as NoteDetailed });
}

function react(_viaKeyboard = false): void {
	pleaseLogin();
	blur();
	reactionPicker.show(
		reactButton.value,
		(reaction) => {
			void api("notes/reactions/create", {
				noteId: note.value.id,
				reaction,
			});
		},
		() => {
			focus();
		},
	);
}

function onContextmenu(ev: MouseEvent): void {
	const isLink = (target: HTMLElement | null): boolean => {
		if (target == null) return false;
		if (target.tagName === "A") return true;
		return isLink(target.parentElement);
	};
	if (isLink(ev.target as HTMLElement | null)) return;
	if (window.getSelection()?.toString() !== "") return;

	if (defaultStore.state.useReactionPickerForContextMenu) {
		ev.preventDefault();
		react();
	} else {
		void buildNoteMenu({ note: note.value, isDeleted })
			.then((items) => os.contextMenu(items, ev))
			.then(focus);
	}
}

function menu(_viaKeyboard = false): void {
	void buildNoteMenu({ note: note.value, isDeleted })
		.then((items) => os.popupMenu(items, { x: 0, y: 0 }))
		.then(focus);
}

function focus(): void {
	noteEl.value?.focus();
}

function blur(): void {
	noteEl.value?.blur();
}

async function loadChildren(): Promise<void> {
	directReplies.value = null;
	directQuotes.value = null;
	tabError.value = null;
	try {
		const res = await api<NoteDetailed[]>("notes/children", {
			noteId: note.value.id,
			limit: 30,
		});
		const sorted = res.reduce<NoteDetailed[]>((acc, resNote) => {
			if (resNote.userId == note.value.userId) {
				return [...acc, resNote];
			}
			return [resNote, ...acc];
		}, []);
		replies.value = sorted as DetailNote[];
		directReplies.value = (sorted.filter(
			(resNote) => resNote.replyId === note.value.id,
		) as DetailNote[]).reverse();
		directQuotes.value = sorted.filter(
			(resNote) => resNote.renoteId === note.value.id,
		) as DetailNote[];
	} catch {
		// Keeping the lists non-null ends the spinners and lets the error banner
		// take over instead of spinning forever.
		directReplies.value = [];
		directQuotes.value = [];
		tabError.value = t("pageLoadError");
	}
}

void loadChildren();

conversation.value = null;
if (note.value.replyId != null) {
	const replyId = note.value.replyId;
	void os
		.api<Note[]>("notes/conversation", {
			noteId: replyId,
			limit: 30,
		})
		.then((res) => {
			conversation.value = res.reverse() as DetailNote[];
			focus();
		})
		.catch(() => {
			conversation.value = [];
		});
}

async function loadRenotes(): Promise<void> {
	renotes.value = null;
	tabError.value = null;
	try {
		const res = await api<NoteDetailed[]>("notes/renotes", {
			noteId: note.value.id,
			limit: 100,
		});
		renotes.value = res as DetailNote[];
	} catch {
		renotes.value = [];
		tabError.value = t("pageLoadError");
	}
}

async function loadReactions(): Promise<void> {
	reactedUsers.value = null;
	tabError.value = null;
	try {
		reactedUsers.value = await api<NoteReaction[]>("notes/reactions", {
			noteId: note.value.id,
			limit: 30,
		});
	} catch {
		reactedUsers.value = [];
		tabError.value = t("pageLoadError");
	}
}

// The payload is passed in rather than read from `tab` so the fetch does not
// depend on whether v-model's update ran before this listener.
function loadTab(value: string | number): void {
	if (value === "renotes" && renotes.value == null) void loadRenotes();
	if (value === "reactions" && reactedUsers.value == null) void loadReactions();
}

function onWheel(): void {
	isScrolling = true;
}

onMounted(() => {
	window.addEventListener("wheel", onWheel);
	isScrolling = false;
	noteEl.value?.scrollIntoView();
});

onUpdated(() => {
	if (!isScrolling) {
		noteEl.value?.scrollIntoView();
		if (location.hash) {
			location.replace(location.hash); // Jump to highlighted reply
		}
	}
});

onUnmounted(() => {
	window.removeEventListener("wheel", onWheel);
});
</script>

<style lang="scss" scoped>
.lxwezrsl {
	font-size: 1.05em;
	position: relative;
	transition: box-shadow 0.1s ease;
	contain: content;

	&:focus-visible {
		outline: none;

		&:after {
			content: "";
			pointer-events: none;
			display: block;
			position: absolute;
			z-index: 10;
			top: 0;
			left: 0;
			right: 0;
			bottom: 0;
			margin: auto;
			width: calc(100% - 8px);
			height: calc(100% - 8px);
			border: solid 1px var(--focus);
			border-radius: var(--radius);
			box-sizing: border-box;
		}
	}

	> .reply-to {
		margin-bottom: -16px;
		padding-bottom: 16px;
	}

	> :deep(.note-container) {
		padding-block: 28px 0;
		padding-top: 12px;
		font-size: 1.1rem;
		overflow: clip;
		outline: none;
		scroll-margin-top: calc(var(--stickyTop) + 20vh);
		&:not(:last-child) {
			border-bottom: 1px solid var(--divider);
			margin-bottom: 4px;
		}
		.article {
			cursor: unset;
			padding-bottom: 0;
		}
		&:first-child {
			padding-top: 28px;
		}
	}

	> :deep(.chips) {
		padding-block: 6px 12px;
		padding-left: 32px;
		&:last-child {
			margin-bottom: 12px;
		}
	}
	> :deep(.user-card-mini),
	> :deep(.reacted-users > *) {
		padding-inline: 32px;
		border-top: 1px solid var(--divider);
		border-radius: 0;
	}
	> :deep(.reacted-users > div) {
		padding-block: 12px;
	}

	> .reply {
		border-top: solid 0.5px var(--divider);
		padding-top: 24px;
		padding-bottom: 10px;
	}

	// Hover
	:deep(.reply > .main),
	.reply-to,
	:deep(.more) {
		position: relative;
		&::before {
			content: "";
			position: absolute;
			inset: -12px -24px;
			bottom: -0px;
			background: var(--panelHighlight);
			border-radius: var(--radius);
			opacity: 0;
			transition:
				opacity 0.2s,
				background 0.2s;
			z-index: -1;
		}
		&.reply-to {
			&::before {
				inset: 0px 8px;
			}
			&:not(.max-width_500px)::before {
				bottom: 16px;
			}
			&:first-of-type::before {
				top: 12px;
			}
			&.reply.max-width_500px:first-of-type::before {
				top: 4px;
			}
		}
		// &::after {
		// 	content: "";
		// 	position: absolute;
		// 	inset: -9999px;
		// 	background: var(--modalBg);
		// 	opacity: 0;
		// 	z-index: -2;
		// 	pointer-events: none;
		// 	transition: opacity .2s;
		// }
		&.more::before {
			inset: 0 !important;
		}
		&:hover,
		&:focus-within {
			--panel: var(--panelHighlight);
			&::before {
				opacity: 1;
				background: var(--panelHighlight) !important;
			}
		}
		// @media (pointer: coarse) {
		// 	&:has(.button:focus-within) {
		// 		z-index: 2;
		// 		--X13: transparent;
		// 		&::after {
		// 			opacity: 1;
		// 			backdrop-filter: var(--modalBgFilter);
		// 		}
		// 	}
		// }
	}
	:deep(.reply:target > .main),
	:deep(.reply-to:target) {
		z-index: 2;
		&::before {
			outline: auto;
			opacity: 1;
			background: none;
		}
	}
	&.max-width_500px {
		font-size: 0.975em;
		> .reply-to {
			&::before {
				inset-inline: -24px;
			}
			&:first-child {
				padding-top: 14px;
				&::before {
					top: -24px;
				}
			}
		}

		> :deep(.note-container) {
			padding: 12px 0 0 0;
			font-size: 1.05rem;
			> .header > .body {
				padding-left: 10px;
			}
		}
		> .clips,
		> :deep(.user-card-mini),
		> :deep(.reacted-users > *) {
			padding-inline: 16px;
		}
		// Keep the tab bar flush with the note body on narrow screens. The enclosing
		// `.max-width_500px` class already gives this selector more weight than
		// CfTab's own padding, so no `!important` is needed; the 90px end padding is
		// the mask fade CfTab relies on.
		> :deep(.underline) {
			padding-inline: 16px 90px;
		}
	}

	&.max-width_300px {
		font-size: 0.825em;
	}
}

.muted {
	padding: 8px;
	text-align: center;
	opacity: 0.7;
}

.tab-message {
	margin: 12px 32px;
}

.tab-empty {
	padding: 24px;
	text-align: center;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
}

.clips {
	// want to redesign at some point
	padding: 24px 32px;
	padding-top: 0;
	> .item {
		display: block;
		padding: 16px;
		// background: var(--buttonBg);
		border: 1px solid var(--divider);
		margin-bottom: var(--margin);
		transition: background 0.2s;
		&:hover,
		&:focus-within {
			background: var(--panelHighlight);
		}

		> .description {
			padding: 8px 0;
		}

		> .user {
			$height: 32px;
			padding-top: 16px;
			border-top: solid 0.5px var(--divider);
			line-height: $height;

			> .avatar {
				width: $height;
				height: $height;
			}
		}
	}
}
</style>

<!-- Styles for the inlined renote / reaction user rows (MkUserCardMini and
     MkReactedUsers are not part of this wave). -->
<style lang="scss" scoped>
.lxwezrsl {
	> .user-card-mini,
	> .reacted-user {
		display: flex;
		align-items: center;
		gap: 8px;
		padding-block: 12px;
		border-top: 1px solid var(--divider);
		text-decoration: none;
		color: var(--fg);

		> .avatar {
			flex-shrink: 0;
			inline-size: 32px;
			block-size: 32px;
			border-radius: 8px;
		}

		> .name {
			font-weight: bold;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		> .acct {
			color: var(--fgTransparentWeak);
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		> .reaction {
			margin-inline-start: auto;
		}
	}
}
</style>
