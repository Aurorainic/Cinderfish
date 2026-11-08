<!-- Ported from firefish@b425f61:packages/client/src/components/MkNote.vue (AGPL-3.0-only) -->
<template>
	<div
		:aria-label="accessibleLabel"
		v-if="!muted.muted"
		v-show="!isDeleted"
		ref="el"
		v-hotkey="keymap"
		v-size="{ max: [500, 350] }"
		class="tkcbzcuz note-container"
		:tabindex="!isDeleted ? '-1' : undefined"
		:class="{ renote: isRenote }"
		:id="appearNote.id"
	>
		<MkNoteSub
			v-if="appearNote.reply && !detailedView && !collapsedReply"
			:note="appearNote.reply"
			class="reply-to"
		/>
		<div
			v-if="!detailedView"
			class="note-context"
			@click="noteClick"
			:class="{
				collapsedReply: collapsedReply && appearNote.reply,
			}"
		>
			<div class="line"></div>
			<div v-if="pinned" class="info">
				<i class="ph-push-pin ph-bold ph-lg"></i
				>{{ t("pinnedNote") }}
			</div>
			<div v-if="isRenote" class="renote">
				<i class="ph-repeat ph-bold ph-lg"></i>
				<span>
					{{ renotedBy.before
					}}<MkA
						v-user-preview="note.userId"
						class="name"
						:to="userPage(note.user)"
						@click.stop
					>
						<MkUserName :user="note.user" /> </MkA
					>{{ renotedBy.after }}
				</span>
				<div class="info">
					<button
						ref="renoteTime"
						class="_button time"
						@click.stop="showRenoteMenu()"
					>
						<i
							v-if="isMyRenote"
							class="ph-dots-three-outline ph-bold ph-lg dropdownIcon"
						></i>
						<MkTime :time="note.createdAt" />
					</button>
					<MkVisibility :note="note" />
				</div>
			</div>
			<div v-if="collapsedReply && appearNote.reply" class="info">
				<MkAvatar class="avatar" :user="appearNote.reply.user" />
				<MkUserName
					class="username"
					:user="appearNote.reply.user"
				></MkUserName>
				<Mfm
					class="summary"
					:text="getNoteSummary(appearNote.reply)"
					:plain="true"
					:nowrap="true"
					:custom-emojis="note.emojis"
				/>
			</div>
		</div>
		<article
			class="article"
			@contextmenu.stop="onContextmenu"
			@click="noteClick"
			:style="{
				cursor: expandOnNoteClick && !detailedView ? 'pointer' : '',
			}"
		>
			<div class="main">
				<div class="header-container">
					<MkAvatar class="avatar" :user="appearNote.user" />
					<MkNoteHeader class="header" :note="appearNote" />
				</div>
				<div class="body">
					<MkSubNoteContent
						class="text"
						:note="appearNote"
						:detailed="true"
						:detailedView="detailedView"
						:parentId="appearNote.replyId ?? undefined"
						@push="(e) => router.push(notePage(e))"
						@focusfooter="footerEl?.focus()"
						@expanded="(e) => setPostExpanded(e)"
					></MkSubNoteContent>
				</div>
				<div v-if="detailedView" class="info">
					<MkA class="created-at" :to="notePage(appearNote)">
						<MkTime :time="appearNote.createdAt" mode="absolute" />
					</MkA>
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
						<template
							v-if="appearNote.repliesCount > 0 && !detailedView"
						>
							<p class="count">{{ appearNote.repliesCount }}</p>
						</template>
					</button>
					<MkRenoteButton
						ref="renoteButton"
						class="button"
						:note="appearNote"
						:count="appearNote.renoteCount"
						:detailedView="detailedView"
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
		</article>
	</div>
	<button
		v-else
		class="muted _button"
		@click="muted.muted = false"
		@contextmenu.stop.prevent
	>
		<small>{{ softMuteText }}</small>
	</button>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * promotion / featured banners, translations, channel info and the plugin
//     note-view interruptors are dropped (no endpoints / no plugin system);
//   * the muted placeholder uses `t()` interpolation instead of `<I18n>`;
//   * "open in window" is dropped (`os.pageWindow` is not part of the shim).
import { computed, ref, type Ref } from "vue";
import { api } from "@/api/client";
import type { Note, NoteDetailed } from "@/api/types";
import MkNoteSub from "@/components/MkNoteSub.vue";
import MkSubNoteContent from "@/components/MkSubNoteContent.vue";
import MkNoteHeader from "@/components/MkNoteHeader.vue";
import MkReactionsViewer from "@/components/MkReactionsViewer.vue";
import MkRenoteButton from "@/components/MkRenoteButton.vue";
import MkStarButton from "@/components/MkStarButton.vue";
import MkStarButtonNoEmoji from "@/components/MkStarButtonNoEmoji.vue";
import MkQuoteButton from "@/components/MkQuoteButton.vue";
import MkVisibility from "@/components/MkVisibility.vue";
import { makeCopyLinkMenuItem } from "@/scripts/copy-link";
import { url } from "@/config";
import { pleaseLogin } from "@/scripts/please-login";
import { focusPrev, focusNext } from "@/scripts/focus";
import { getWordSoftMute, type Muted } from "@/scripts/check-word-mute";
import { getNoteSummary } from "@/scripts/get-note-summary";
import { buildNoteMenu } from "@/scripts/get-note-menu";
import { reactionPicker } from "@/scripts/reaction-picker";
import { useNoteCapture } from "@/composables/use-note-capture";
import { deepClone } from "@/scripts/clone";
import { notePage } from "@/filters/note";
import { userPage } from "@/filters/user";
import { useRouter } from "@/router";
import { defaultStore } from "@/store/base";
import { me } from "@/account";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { post } from "@/os/post";
import type { NoteMenuItem } from "@/scripts/get-note-menu";

type ViewNote = Note &
	Partial<
		Pick<
			NoteDetailed,
			"reply" | "renote" | "poll" | "myReaction" | "hasPoll"
		>
	>;

const router = useRouter();

const props = defineProps<{
	note: ViewNote;
	pinned?: boolean;
	detailedView?: boolean;
	collapsedReply?: boolean;
}>();

const note = ref(deepClone(props.note)) as Ref<ViewNote>;
const muted = ref<Muted>(
	getWordSoftMute(note.value, me.value, defaultStore.state.mutedWords),
);

const isRenote =
	note.value.renote != null &&
	note.value.text == null &&
	note.value.fileIds.length === 0 &&
	note.value.poll == null;

const el = ref<HTMLElement | null>(null);
const footerEl = ref<HTMLElement | null>(null);
const menuButton = ref<HTMLElement | null>(null);
const renoteButton = ref<{ renote: () => void } | null>(null);
const renoteTime = ref<HTMLElement | null>(null);
const reactButton = ref<HTMLElement | null>(null);
const appearNote = computed<ViewNote>(() =>
	isRenote ? (note.value.renote as ViewNote) : note.value,
);
const isMyRenote = me.value != null && me.value.id === note.value.userId;
const isDeleted = ref(false);
const enableEmojiReactions = defaultStore.state.enableEmojiReactions;
const expandOnNoteClick = defaultStore.state.expandOnNoteClick;

// `renotedBy` is a sentence with the user name in the middle; splitting the
// template keeps the name a link without the `<I18n>` component.
const renotedBy = computed(() => {
	const template = t("renotedBy", { user: "\u0000" });
	const [before, after] = template.split("\u0000");
	return { before: before ?? "", after: after ?? "" };
});

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

const keymap = {
	r: () => reply(true),
	"e|a|plus": () => react(true),
	q: () => renoteButton.value?.renote(),
	"up|k": focusBefore,
	"down|j": focusAfter,
	esc: blur,
	"m|o": () => menu(true),
};

useNoteCapture({
	note: appearNote as unknown as Ref<NoteDetailed>,
	isDeletedRef: isDeleted,
});

function reply(_viaKeyboard = false): void {
	pleaseLogin();
	// Firefish passes `animation: !viaKeyboard` too; the dialog has no such prop
	// in this port, and MkPostFormDialog owns its own transition.
	void post({ reply: appearNote.value as NoteDetailed }).then(() => {
		focus();
	});
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

function undoReact(target: ViewNote): void {
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
		// The Audio element's context menu is the browser default.
		if (target.tagName === "AUDIO") return true;
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
				makeCopyLinkMenuItem(`${url}${notePage(appearNote.value)}`),
				appearNote.value.user.host != null
					? {
							icon: "ph-arrow-square-up-right ph-bold ph-lg",
							text: t("showOnRemote"),
							action: () => {
								const remote =
									appearNote.value.url ??
									appearNote.value.uri;
								if (remote != null)
									window.open(remote, "_blank");
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

function showRenoteMenu(): void {
	if (!isMyRenote) return;
	const rect = renoteTime.value?.getBoundingClientRect();
	const renoteMenuItems: NoteMenuItem[] = [
		{
			text: t("unrenote"),
			icon: "ph-trash ph-bold ph-lg",
			danger: true,
			action: () => {
				void api("notes/delete", {
					noteId: note.value.id,
				});
				isDeleted.value = true;
			},
		},
	];
	void os.popupMenu(renoteMenuItems, {
		x: rect?.left ?? 0,
		y: (rect?.bottom ?? 0) + 4,
	});
}

function focus(): void {
	el.value?.focus();
}

function blur(): void {
	el.value?.blur();
}

function focusBefore(): void {
	if (el.value != null) focusPrev(el.value);
}

function focusAfter(): void {
	if (el.value != null) focusNext(el.value);
}

function scrollIntoView(): void {
	el.value?.scrollIntoView();
}

function noteClick(e: MouseEvent): void {
	if (
		document.getSelection()?.type === "Range" ||
		props.detailedView ||
		!expandOnNoteClick
	) {
		e.stopPropagation();
	} else {
		void router.push(notePage(appearNote.value));
	}
}

const postIsExpanded = ref(false);

function setPostExpanded(val: boolean): void {
	postIsExpanded.value = val;
}

const accessibleLabel = computed(() => {
	let label = `${appearNote.value.user.username}; `;
	if (appearNote.value.renote) {
		label += `${t("renoted")} ${appearNote.value.renote.user.username}; `;
		if (appearNote.value.renote.cw) {
			label += `${t("cw")}: ${appearNote.value.renote.cw}; `;
			if (postIsExpanded.value) {
				label += `${appearNote.value.renote.text}; `;
			}
		} else {
			label += `${appearNote.value.renote.text}; `;
		}
	} else {
		if (appearNote.value.cw) {
			label += `${t("cw")}: ${appearNote.value.cw}; `;
			if (postIsExpanded.value) {
				label += `${appearNote.value.text}; `;
			}
		} else {
			label += `${appearNote.value.text}; `;
		}
	}
	const date = new Date(appearNote.value.createdAt);
	label += date.toLocaleTimeString();
	return label;
});

defineExpose({
	focus,
	blur,
	scrollIntoView,
});
</script>

<style lang="scss" scoped>
.tkcbzcuz {
	position: relative;
	transition: box-shadow 0.1s ease;
	font-size: 1.05em;
	overflow: clip;
	contain: content;
	-webkit-tap-highlight-color: transparent;

	// これらの指定はパフォーマンス向上には有効だが、ノートの高さは一定でないため、
	// 下の方までスクロールすると上のノートの高さがここで決め打ちされたものに変化し、表示しているノートの位置が変わってしまう
	// ノートがマウントされたときに自身の高さを取得し contain-intrinsic-size を設定しなおせばほぼ解決できそうだが、
	// 今度はその処理自体がパフォーマンス低下の原因にならないか懸念される。また、被リアクションでも高さは変化するため、やはり多少のズレは生じる
	// 一度レンダリングされた要素はブラウザがよしなにサイズを覚えておいてくれるような実装になるまで待った方が良さそう(なるのか？)
	//content-visibility: auto;
	//contain-intrinsic-size: 0 128px;

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

	& > .article > .main {
		&:hover,
		&:focus-within {
			:deep(.footer .button) {
				opacity: 1;
			}
		}
	}

	> .reply-to {
		& + .note-context {
			.line::before {
				content: "";
				display: block;
				margin-bottom: -4px;
				margin-top: 16px;
				border-left: 2px solid currentColor;
				margin-left: calc((var(--avatarSize) / 2) - 1px);
				opacity: 0.25;
			}
		}
	}

	.note-context {
		position: relative;
		padding: 0 32px 0 32px;
		display: flex;
		z-index: 1;
		&:first-child {
			margin-top: 20px;
		}
		> :not(.line) {
			width: 0;
			flex-grow: 1;
			position: relative;
			line-height: 28px;
		}
		> .line {
			position: relative;
			z-index: 2;
			width: var(--avatarSize);
			display: flex;
			margin-right: 14px;
			margin-top: 0;
			flex-grow: 0;
			pointer-events: none;
		}

		> div > i {
			margin-left: -0.5px;
		}
		> .info {
			display: flex;
			align-items: center;
			font-size: 90%;
			white-space: pre;
			color: #f6c177;

			> i {
				margin-right: 4px;
			}

			> .hide {
				margin-left: auto;
				color: inherit;
			}
		}

		> .renote {
			display: flex;
			align-items: center;
			white-space: pre;
			color: var(--renote);
			cursor: pointer;

			> i {
				margin-right: 4px;
			}

			> span {
				overflow: hidden;
				flex-shrink: 1;
				text-overflow: ellipsis;
				white-space: nowrap;

				> .name {
					font-weight: bold;
				}
			}

			> .info {
				margin-left: auto;
				font-size: 0.9em;
				display: flex;

				> .time {
					flex-shrink: 0;
					color: inherit;
					display: inline-flex;
					align-items: center;
					> .dropdownIcon {
						margin-right: 4px;
					}
				}
			}
		}

		&.collapsedReply {
			.line {
				opacity: 0.25;
				&::after {
					content: "";
					position: absolute;
					border-left: 2px solid currentColor;
					border-top: 2px solid currentColor;
					margin-left: calc(var(--avatarSize) / 2 - 1px);
					width: calc(var(--avatarSize) / 2 + 14px);
					border-top-left-radius: calc(var(--avatarSize) / 4);
					top: calc(50% - 1px);
					height: calc(50% + 5px);
				}
			}
			.info {
				color: var(--fgTransparentWeak);
				transition: color 0.2s;
			}
			.avatar {
				width: 1.2em;
				height: 1.2em;
				border-radius: 2em;
				overflow: hidden;
				margin-right: 0.4em;
				background: var(--panelHighlight);
			}
			.username {
				font-weight: 700;
				flex-shrink: 0;
				max-width: 30%;
				&::after {
					content: ": ";
				}
			}
			&:hover,
			&:focus-within {
				.info {
					color: var(--fg);
				}
			}
		}
	}

	> .article {
		position: relative;
		overflow: clip;
		padding: 20px 32px 10px;
		margin-top: -16px;

		&:first-child,
		&:nth-child(2) {
			margin-top: -100px;
			padding-top: 104px;
		}

		@media (pointer: coarse) {
			cursor: default;
		}

		.header-container {
			display: flex;
			position: relative;
			z-index: 2;
			> .avatar {
				flex-shrink: 0;
				display: block;
				margin: 0 14px 0 0;
				width: var(--avatarSize);
				height: var(--avatarSize);
				position: relative;
				top: 0;
				left: 0;
			}
			> .header {
				width: 0;
				flex-grow: 1;
			}
		}
		> .main {
			flex: 1;
			min-width: 0;

			> .body {
				margin-top: 0.7em;
				> .translation {
					border: solid 0.5px var(--divider);
					border-radius: var(--radius);
					padding: 12px;
					margin-top: 8px;
				}
				> .renote {
					padding-top: 8px;
					> * {
						padding: 16px;
						border: solid 1px var(--renote);
						border-radius: 8px;
						transition: background 0.2s;
						&:hover,
						&:focus-within {
							background-color: var(--panelHighlight);
						}
					}
				}
			}
			> .info {
				display: flex;
				justify-content: space-between;
				flex-wrap: wrap;
				gap: 0.7em;
				margin-top: 16px;
				color: var(--fgTransparentWeak);
				font-size: 0.9em;
			}
			> .footer {
				position: relative;
				z-index: 2;
				display: flex;
				flex-wrap: wrap;
				margin-top: 0.4em;
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
						color: var(--fgTransparentWeak);
					}

					&.reacted {
						color: var(--accent);
					}
				}
			}
		}
	}

	> .reply {
		border-top: solid 0.5px var(--divider);
	}

	&.max-width_500px {
		font-size: 0.975em;
		--avatarSize: 46px;
		padding-top: 6px;
		> .note-context {
			padding-inline: 16px;
			margin-top: 8px;
			> :not(.line) {
				margin-top: 0px;
			}
			> .line {
				margin-right: 10px;
				&::before {
					margin-top: 8px;
				}
			}
		}
		> .article {
			padding: 18px 16px 8px;
			&:first-child,
			&:nth-child(2) {
				padding-top: 104px;
			}
			> .main > .header-container > .avatar {
				margin-right: 10px;
				// top: calc(14px + var(--stickyTop, 0px));
			}
		}
	}

	&.max-width_300px {
		--avatarSize: 40px;
	}
}

.muted {
	padding: 8px;
	text-align: center;
	color: var(--fgTransparentWeak);
	width: 100%;

	._blur_text {
		pointer-events: auto;
	}
	&:active ._blur_text {
		filter: blur(0px);
	}
}
</style>
