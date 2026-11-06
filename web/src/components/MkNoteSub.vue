<!-- Ported from firefish@b425f61:packages/client/src/components/MkNoteSub.vue (AGPL-3.0-only) -->
<template>
	<article
		v-if="!muted.muted || muted.what === 'reply'"
		ref="el"
		v-size="{ max: [450, 500] }"
		class="wrpstxzv"
		:id="detailedView ? appearNote.id : undefined"
		tabindex="-1"
		:class="{
			children: depth > 1,
			singleStart: replies.length == 1,
			firstColumn: depth == 1 && conversation,
		}"
		@contextmenu.stop="onContextmenu"
	>
		<div v-if="conversation && depth > 1" class="line"></div>
		<div
			class="main"
			@click="noteClick"
			:style="{ cursor: expandOnNoteClick ? 'pointer' : '' }"
		>
			<div class="avatar-container">
				<MkAvatar class="avatar" :user="appearNote.user" />
				<div
					v-if="!conversation || replies.length > 0"
					class="line"
				></div>
			</div>
			<div class="body">
				<MkNoteHeader class="header" :note="note" :mini="true" />
				<div class="body">
					<MkSubNoteContent
						class="text"
						:note="note"
						:parentId="parentId"
						:conversation="conversation != null"
						:detailedView="detailedView"
						@focusfooter="footerEl?.focus()"
					/>
				</div>
				<footer ref="footerEl" class="footer" tabindex="-1">
					<MkReactionsViewer
						v-if="enableEmojiReactions"
						:note="appearNote"
					/>
					<button
						v-tooltip.noDelay.bottom="t('reply')"
						class="button _button"
						@click.stop="reply()"
					>
						<i class="ph-arrow-u-up-left ph-bold ph-lg"></i>
						<template v-if="appearNote.repliesCount > 0">
							<p class="count">{{ appearNote.repliesCount }}</p>
						</template>
					</button>
					<MkRenoteButton
						class="button"
						:note="appearNote"
						:count="appearNote.renoteCount"
					/>
					<MkStarButtonNoEmoji
						v-if="!enableEmojiReactions"
						class="button"
						:note="appearNote"
						:count="
							Object.values(appearNote.reactions).reduce(
								(partialSum, val) => partialSum + val,
								0,
							)
						"
						:reacted="appearNote.myReaction != null"
					/>
					<MkStarButton
						v-if="
							enableEmojiReactions &&
							appearNote.myReaction == null
						"
						class="button"
						:note="appearNote"
					/>
					<button
						v-if="
							enableEmojiReactions &&
							appearNote.myReaction == null
						"
						ref="reactButton"
						v-tooltip.noDelay.bottom="t('reaction')"
						class="button _button"
						@click.stop="react()"
					>
						<i class="ph-smiley ph-bold ph-lg"></i>
					</button>
					<button
						v-if="
							enableEmojiReactions &&
							appearNote.myReaction != null
						"
						ref="reactButton"
						class="button _button reacted"
						@click.stop="undoReact(appearNote)"
						v-tooltip.noDelay.bottom="t('removeReaction')"
					>
						<i class="ph-minus ph-bold ph-lg"></i>
					</button>
					<MkQuoteButton class="button" :note="appearNote" />
					<button
						ref="menuButton"
						v-tooltip.noDelay.bottom="t('more')"
						class="button _button"
						@click.stop="menu()"
					>
						<i class="ph-dots-three-outline ph-bold ph-lg"></i>
					</button>
				</footer>
			</div>
		</div>
		<template v-if="conversation">
			<MkNoteSub
				v-if="replyLevel < 11 && depth < 5"
				v-for="reply in replies"
				:key="reply.id"
				:note="reply"
				class="reply"
				:class="{ single: replies.length == 1 }"
				:conversation="conversation"
				:depth="replies.length == 1 ? depth : depth + 1"
				:replyLevel="replyLevel + 1"
				:parentId="appearNote.id"
				:detailedView="detailedView"
			/>
			<div v-else-if="replies.length > 0" class="more">
				<div class="line"></div>
				<MkA class="text _link" :to="notePage(note)"
					>{{ t("continueThread") }}
					<i class="ph-caret-double-right ph-bold ph-lg"></i
				></MkA>
			</div>
		</template>
	</article>
	<div v-else class="muted" @click="muted.muted = false">
		<small>{{ softMuteText }}</small>
	</div>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * translation is dropped (no `notes/translate` endpoint);
//   * the muted-note placeholder uses plain `t()` interpolation instead of the
//     port's missing `<I18n>` component, so the author is not a link there;
//   * "open in window" is dropped (`os.pageWindow` is not part of the os shim)
//     and the router has no `forcePage` mode;
//   * `v-hotkey` is registered as a no-op by the foundation layer.
import { computed, ref, type Ref } from "vue";
import { api } from "@/api/client";
import type { Note, NoteDetailed } from "@/api/types";
import { post } from "@/os/post";
import MkNoteHeader from "@/components/MkNoteHeader.vue";
import MkSubNoteContent from "@/components/MkSubNoteContent.vue";
import MkReactionsViewer from "@/components/MkReactionsViewer.vue";
import MkStarButton from "@/components/MkStarButton.vue";
import MkStarButtonNoEmoji from "@/components/MkStarButtonNoEmoji.vue";
import MkRenoteButton from "@/components/MkRenoteButton.vue";
import MkQuoteButton from "@/components/MkQuoteButton.vue";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { url } from "@/config";
import { pleaseLogin } from "@/scripts/please-login";
import { buildNoteMenu } from "@/scripts/get-note-menu";
import { getWordSoftMute, type Muted } from "@/scripts/check-word-mute";
import { notePage } from "@/filters/note";
import { useRouter } from "@/router";
import { reactionPicker } from "@/scripts/reaction-picker";
import { useNoteCapture } from "@/composables/use-note-capture";
import { deepClone } from "@/scripts/clone";
import { defaultStore } from "@/store/base";
import { me } from "@/account";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";

type SubNote = Note &
	Partial<
		Pick<
			NoteDetailed,
			"reply" | "renote" | "poll" | "myReaction" | "hasPoll"
		>
	>;

const router = useRouter();

const props = withDefaults(
	defineProps<{
		note: SubNote;
		conversation?: SubNote[];
		parentId?: string;
		detailedView?: boolean;

		// how many notes are in between this one and the note being viewed in detail
		depth?: number;
		// the actual reply level of this note within the conversation thread
		replyLevel?: number;
	}>(),
	{
		depth: 1,
		replyLevel: 1,
	},
);

const note = ref(deepClone(props.note)) as Ref<SubNote>;
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

const el = ref<HTMLElement | null>(null);
const footerEl = ref<HTMLElement | null>(null);
const menuButton = ref<HTMLElement | null>(null);
const reactButton = ref<HTMLElement | null>(null);
const appearNote = computed<SubNote>(() =>
	isRenote ? (note.value.renote as SubNote) : note.value,
);
const isDeleted = ref(false);

const replies = computed<SubNote[]>(() =>
	props.conversation
		?.filter(
			(item) =>
				item.replyId === props.note.id ||
				item.renoteId === props.note.id,
		)
		.reverse() ?? [],
);
const enableEmojiReactions = defaultStore.state.enableEmojiReactions;
const expandOnNoteClick = defaultStore.state.expandOnNoteClick;

useNoteCapture({
	note: appearNote as unknown as Ref<NoteDetailed>,
	isDeletedRef: isDeleted,
});

function reply(_viaKeyboard = false): void {
	pleaseLogin();
	void post({ reply: props.note as unknown as NoteDetailed });
}

function react(_viaKeyboard = false): void {
	pleaseLogin();
	blur();
	reactionPicker.show(
		reactButton.value,
		(reaction) => {
			void api("notes/reactions/create", {
				noteId: appearNote.value.id,
				reaction,
			});
		},
		() => {
			focus();
		},
	);
}

function undoReact(target: SubNote): void {
	const oldReaction = target.myReaction;
	if (!oldReaction) return;
	void api("notes/reactions/delete", {
		noteId: target.id,
	});
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
		const contextMenuItems: (MenuItem | undefined)[] = [
				{
					type: "label",
					text: notePage(appearNote.value),
				},
				notePage(appearNote.value) != location.pathname
					? {
							icon: "ph-arrows-out-simple ph-bold ph-lg",
							text: t("showInPage"),
							action: () => {
								void router.push(notePage(appearNote.value));
							},
					  }
					: undefined,
				{ separator: true },
				{
					icon: "ph-arrow-square-out ph-bold ph-lg",
					text: t("openInNewTab"),
					action: () => {
						window.open(notePage(appearNote.value), "_blank");
					},
				},
				{
					icon: "ph-link-simple ph-bold ph-lg",
					text: t("copyLink"),
					action: () => {
						copyToClipboard(`${url}${notePage(appearNote.value)}`);
					},
				},
				note.value.user.host != null
					? {
							icon: "ph-arrow-square-up-right ph-bold ph-lg",
							text: t("showOnRemote"),
							action: () => {
								const remote = note.value.url ?? note.value.uri;
								if (remote != null) window.open(remote, "_blank");
							},
					  }
					: undefined,
		];
		void os.contextMenu(
			contextMenuItems.filter((item): item is MenuItem => item !== undefined),
			ev,
		);
	}
}

function menu(_viaKeyboard = false): void {
	const rect = menuButton.value?.getBoundingClientRect();
	void buildNoteMenu({ note: note.value, isDeleted })
		.then((items) =>
			os.popupMenu(items, {
				x: rect?.left ?? 0,
				y: (rect?.bottom ?? 0) + 4,
			}),
		)
		.then(focus);
}

function focus(): void {
	el.value?.focus();
}

function blur(): void {
	el.value?.blur();
}

function noteClick(e: MouseEvent): void {
	if (document.getSelection()?.type === "Range" || !expandOnNoteClick) {
		e.stopPropagation();
	} else {
		void router.push(notePage(props.note));
	}
}
</script>

<style lang="scss" scoped>
.wrpstxzv {
	padding: 16px 32px;
	outline: none;
	&.children {
		padding: 10px 0 0 var(--indent);
		padding-left: var(--indent) !important;
		font-size: 1em;
		cursor: auto;

		&.max-width_500px {
			padding: 10px 0 0 8px;
		}
	}

	> .main {
		display: flex;

		> .avatar-container {
			margin-right: 8px;
			z-index: 2;
			> .avatar {
				flex-shrink: 0;
				display: block;
				width: 38px;
				height: 38px;
				border-radius: 8px;
			}
		}

		> .body {
			position: relative;
			flex: 1;
			min-width: 0;
			margin: 0 -200px;
			padding: 0 200px;
			overflow: clip;
			@media (pointer: coarse) {
				cursor: default;
			}

			> .header {
				margin-bottom: 2px;
				cursor: auto;
			}

			> .body {
				> .translation {
					border: solid 0.5px var(--divider);
					border-radius: var(--radius);
					padding: 12px;
					margin-top: 8px;
				}
			}
			> .footer {
				position: relative;
				z-index: 2;
				display: flex;
				flex-wrap: wrap;

				> :deep(.button) {
					position: relative;
					margin: 0;
					padding: 8px;
					opacity: 0.7;
					&:disabled {
						opacity: 0.5 !important;
					}
					flex-grow: 1;
					max-width: 3.5em;
					width: max-content;
					min-width: max-content;
					height: auto;
					transition: opacity 0.2s;
					&::before {
						content: "";
						position: absolute;
						inset: 0;
						bottom: 2px;
						background: var(--panel);
						z-index: -1;
						transition: background 0.2s;
					}
					&:first-of-type {
						margin-left: -0.5em;
						&::before {
							border-radius: 100px 0 0 100px;
						}
					}
					&:last-of-type {
						&::before {
							border-radius: 0 100px 100px 0;
						}
					}
					&:hover {
						color: var(--fgHighlighted);
					}

					> i {
						display: inline !important;
					}

					> .count {
						display: inline;
						margin: 0 0 0 8px;
						opacity: 0.7;
					}

					&.reacted {
						color: var(--accent);
					}
				}
			}
		}
	}

	&.reply > .main {
		margin-inline: -200px;
		padding-inline: 200px;
		&::before {
			inset-inline: 176px !important;
		}
	}
	&.reply,
	&.reply-to {
		> .main > .body {
			margin-right: -24px;
			padding-right: 24px;
			margin-top: -12px;
			padding-top: 12px;
			margin-left: calc(0px - var(--avatarSize) - 32px);
			padding-left: calc(var(--avatarSize) + 32px);
			border-radius: var(--radius);
		}
	}
	&.reply-to {
		> .main > .body {
			margin-left: calc(0px - var(--avatarSize) - 38px);
			padding-left: calc(var(--avatarSize) + 38px);
			margin-top: -16px;
			padding-top: 16px;
		}
	}
	&.reply {
		--avatarSize: 38px;
		.avatar-container {
			margin-right: 8px !important;
		}
	}
	> .reply,
	> .more {
		margin-top: 10px;
		&.single {
			padding: 0 !important;
			> .line {
				display: none;
			}
		}
	}

	> .more {
		display: flex;
		padding-block: 10px;
		font-weight: 600;
		> .line {
			position: relative;
			z-index: 2;
			flex-grow: 0 !important;
			margin-top: -10px !important;
			margin-bottom: 10px !important;
			margin-right: 10px !important;
			&::before {
				border-left-style: dashed !important;
				border-bottom-left-radius: 100px !important;
			}
		}
		i {
			font-size: 1em !important;
			vertical-align: middle !important;
		}
		a {
			position: static;
			&::before {
				content: "";
				position: absolute;
				inset: 0;
			}
			&::after {
				content: unset;
			}
		}
	}

	&.reply,
	&.reply-to,
	&.reply-to-more {
		> .main:hover,
		> .main:focus-within {
			:deep(.footer .button) {
				opacity: 1;
			}
		}
	}

	&.reply-to,
	&.reply-to-more {
		padding-bottom: 0;
		&:first-child {
			padding-top: 24px;
		}
		.line::before {
			margin-bottom: -16px;
		}
	}

	// Reply Lines
	&.reply,
	&.reply-to,
	&.reply-to-more {
		--indent: calc(var(--avatarSize) - 5px);
		> .main {
			> .avatar-container {
				display: flex;
				flex-direction: column;
				align-items: center;
				margin-right: 14px;
				width: var(--avatarSize);
				> .avatar {
					width: var(--avatarSize);
					height: var(--avatarSize);
					margin: 0;
				}
			}
		}
		.line {
			position: relative;
			z-index: 2;
			width: var(--avatarSize);
			display: flex;
			flex-grow: 1;
			margin-bottom: -10px;
			pointer-events: none;
			opacity: 0.25;
			&::before {
				content: "";
				position: absolute;
				border-left: 2px solid currentColor;
				margin-left: calc((var(--avatarSize) / 2) - 1px);
				width: calc(var(--indent) / 2);
				inset-block: 0;
				min-height: 8px;
			}
		}
	}
	&.reply-to,
	&.reply-to-more {
		> .main > .avatar-container > .line {
			margin-bottom: 0px !important;
		}
	}
	&.single,
	&.singleStart {
		> .main > .avatar-container > .line {
			margin-bottom: -10px !important;
		}
	}
	.reply.children:not(:last-child) {
		// Line that goes through multiple replies
		position: relative;
		> .line {
			position: absolute;
			top: 0;
			left: 0;
			bottom: 0;
		}
	}
	// Reply line connectors
	.reply.children:not(.single) {
		position: relative;
		> .line {
			position: absolute;
			z-index: 2;
			left: 0;
			top: 0;
			opacity: 0.25;
			&::after {
				content: "";
				position: absolute;
				border-left: 2px solid currentColor;
				border-bottom: 2px solid currentColor;
				margin-left: calc((var(--avatarSize) / 2) - 1px);
				width: calc(var(--indent) / 2);
				height: calc((var(--avatarSize) / 2));
				border-bottom-left-radius: calc(var(--indent) / 2);
				top: 8px;
			}
		}
		&:not(:last-child) > .line::after {
			mask: linear-gradient(to right, transparent 2px, black 2px);
			-webkit-mask: linear-gradient(to right, transparent 2px, black 2px);
		}
	}
	// End Reply Divider
	.children > .main:last-child {
		padding-bottom: 1em;
		&::before {
			bottom: 1em;
		}
		// &::after {
		// 	content: "";
		// 	border-top: 1px solid var(--X13);
		// 	position: absolute;
		// 	bottom: 0;
		// 	margin-left: calc(var(--avatarSize) + 12px);
		// 	inset-inline: 0;
		// }
	}
	&.firstColumn > .children:last-child > .main {
		padding-bottom: 0 !important;
		&::before {
			bottom: 0 !important;
		}
		// &::after { content: unset }
	}

	&.max-width_500px {
		padding: 14px 16px;
		:not(.reply) > & {
			.reply {
				--avatarSize: 24px;
				--indent: calc(var(--avatarSize) - 4px);
			}
		}
		&.firstColumn {
			> .main,
			> .line,
			> .children:not(.single) > .line {
				--avatarSize: 35px;
				--indent: 35px;
			}
			> .children:not(.single) {
				padding-left: 28px !important;
			}
		}
		&.reply-to {
			--avatarSize: 46px;
			padding: 14px 16px;
			padding-top: 14px !important;
			padding-bottom: 0 !important;
			margin-bottom: 0 !important;
		}
		> .main > .avatar-container {
			margin-right: 10px;
		}
		&:first-child > .main > .body {
			margin-top: -20px;
			padding-top: 22px;
		}
	}
}

.muted {
	padding: 8px;
	text-align: center;
	opacity: 0.7;
}
</style>
