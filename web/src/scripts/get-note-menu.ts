// Ported from firefish@b425f61:packages/client/src/scripts/get-note-menu.ts (AGPL-3.0-only)
//
// Degradation: only entries whose endpoints exist in Cinderfish are rendered.
// Dropped from the Firefish menu (no endpoint, see the capability matrix):
// notes/state (so the favourite entry cannot know the current state and always
// creates one), watch / thread mute, translate, delete-and-edit, report
// abuse, promote, plugin actions and the user submenu. Edit and the
// history page are Cinderfish's own additions.
import type { Ref } from "vue";
import { ApiError } from "@/api/errors";
import { ENDPOINTS } from "@/api/endpoints";
import type { Clip, Note, NoteDetailed, NoteState } from "@/api/types";
import { me } from "@/account";
import { url } from "@/config";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { router } from "@/router";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { makeCopyLinkMenuItem } from "@/scripts/copy-link";

/** `clips/add-note` for a note already in the clip; Firefish matches it verbatim. */
const ALREADY_CLIPPED_ERROR_ID = "734806c4-542c-463a-9311-15c512803965";

/** Firefish types the menu against the full note shape; the timeline projection is a subset. */
export type NoteMenuNote = Note &
	Partial<
		Pick<
			NoteDetailed,
			"renote" | "reply" | "poll" | "myReaction" | "hasPoll" | "clippedCount"
		>
	>;

export interface NoteMenuItem extends MenuItem {
	danger?: boolean;
	accent?: boolean;
}

export interface NoteMenuProps {
	note: NoteMenuNote;
	isDeleted: Ref<boolean>;
	/** Fresh `notes/state` result; null when unknown (logged out or failed). */
	state?: NoteState | null;
}

/**
 * Fetches `notes/state` and builds the menu. Call sites use this instead of
 * getNoteMenu so favorite/pin entries reflect the real state.
 */
export async function buildNoteMenu(
	props: NoteMenuProps,
	/** Where the note menu was opened; the clip sub-popup reuses it. */
	point?: MouseEvent | { x: number; y: number },
): Promise<NoteMenuItem[]> {
	const note = props.note;
	const isPureRenote =
		note.renote != null &&
		note.text == null &&
		note.fileIds.length === 0 &&
		note.poll == null;
	const target = (isPureRenote ? note.renote : note) as NoteMenuNote;
	let state: NoteState | null = null;
	if (me.value != null) {
		try {
			state = await os.api("notes/state", { noteId: target.id });
		} catch {
			state = null;
		}
	}
	return getNoteMenu({ ...props, state }, point);
}

export function getNoteMenu(
	props: NoteMenuProps,
	/** Where the note menu was opened; the clip sub-popup reuses it. */
	point?: MouseEvent | { x: number; y: number },
): NoteMenuItem[] {
	const note = props.note;
	const isRenote =
		note.renote != null &&
		note.text == null &&
		note.fileIds.length === 0 &&
		note.poll == null;

	const appearNote = isRenote
		? (note.renote as NoteMenuNote)
		: note;

	const current = me.value;
	const isAppearAuthor = current != null && appearNote.userId === current.id;
	const isMyRenote = current != null && isRenote && note.userId === current.id;

	function del(): void {
		void os
			.confirm({
				type: "warning",
				text: t("noteDeleteConfirm"),
			})
			.then(({ canceled }) => {
				if (canceled) return;
				void os.apiWithDialog("notes/delete", { noteId: appearNote.id });
				props.isDeleted.value = true;
			});
	}

	function unrenote(): void {
		void os.apiWithDialog("notes/delete", { noteId: note.id });
		props.isDeleted.value = true;
	}

	function toggleFavorite(): void {
		void os.apiWithDialog(
			props.state?.isFavorited
				? "notes/favorites/delete"
				: "notes/favorites/create",
			{ noteId: appearNote.id },
		);
	}

	function togglePin(pin: boolean): void {
		void os.apiWithDialog(pin ? "i/pin" : "i/unpin", {
			noteId: appearNote.id,
		});
	}

	function copyContent(): void {
		copyToClipboard(appearNote.text ?? "");
		os.success();
	}

	/**
	 * Firefish opens `os.form` for a new clip. The port has no form helper, so
	 * the popup asks for the name only; the clip is created private with no
	 * description and the note is added to it.
	 */
	async function clip(): Promise<void> {
		const clips = await os.api(ENDPOINTS.clipsList, {});
		await os.popupMenu(
			[
				{
					icon: "ph-plus ph-bold ph-lg",
					text: t("createNewClip"),
					action: async () => {
						const { canceled, result } = await os.inputText({
							title: t("createNewClip"),
							placeholder: t("name"),
						});
						const name = String(result ?? "").trim();
						if (canceled || name === "") return;
						try {
							const created = await os.apiWithDialog<Clip>(
								ENDPOINTS.clipsCreate,
								{ name },
							);
							await os.apiWithDialog(ENDPOINTS.clipsAddNote, {
								clipId: created.id,
								noteId: appearNote.id,
							});
						} catch {
							// apiWithDialog already showed the server error.
						}
					},
				},
				{ separator: true },
				...clips.map((entry) => ({
					text: entry.name,
					action: () => {
						void os.promiseDialog(
							os.api(ENDPOINTS.clipsAddNote, {
								clipId: entry.id,
								noteId: appearNote.id,
							}),
							null,
							async (error: unknown) => {
								if (
									error instanceof ApiError &&
									error.errorId === ALREADY_CLIPPED_ERROR_ID
								) {
									const { canceled } = await os.confirm({
										type: "warning",
										text: t("confirmToUnclipAlreadyClippedNote", {
											name: entry.name,
										}),
									});
									if (canceled) return;
									await os.apiWithDialog(ENDPOINTS.clipsRemoveNote, {
										clipId: entry.id,
										noteId: appearNote.id,
									});
									return;
								}
								const text =
									error instanceof ApiError
										? `${error.message}\n${error.errorId ?? error.code ?? ""}`.trim()
										: String(error);
								void os.alert({ type: "error", text });
							},
						);
					},
				})),
			],
			point ?? { x: 0, y: 0 },
		);
	}

	function copyOriginal(): void {
		const original = appearNote.url ?? appearNote.uri;
		if (original == null) return;
		copyToClipboard(original);
		os.success();
	}

	function showOnRemote(): void {
		const remote = appearNote.url ?? appearNote.uri;
		if (remote == null) return;
		window.open(remote, "_blank");
	}

	function share(): void {
		void navigator.share({
			title: t("noteOf", {
				user: appearNote.user.name ?? appearNote.user.username,
			}),
			text: appearNote.text ?? undefined,
			url: `${url}/notes/${appearNote.id}`,
		});
	}

	const shareAvailable = typeof navigator !== "undefined" && typeof navigator.share === "function";
	const hasRemote = appearNote.url != null || appearNote.uri != null;

	if (current == null) {
		const loggedOut: (NoteMenuItem | undefined)[] = [
			{
				icon: "ph-clipboard-text ph-bold ph-lg",
				text: t("copyContent"),
				action: copyContent,
			},
			makeCopyLinkMenuItem(`${url}/notes/${appearNote.id}`, {
				onCopied: () => os.success(),
			}),
			hasRemote
				? {
						icon: "ph-link-simple ph-bold ph-lg",
						text: `${t("copyLink")} (${t("remote")})`,
						action: copyOriginal,
				  }
				: undefined,
			hasRemote
				? {
						icon: "ph-arrow-square-out ph-bold ph-lg",
						text: t("showOnRemote"),
						action: showOnRemote,
				  }
				: undefined,
			shareAvailable
				? {
						icon: "ph-share-network ph-bold ph-lg",
						text: t("share"),
						action: share,
				  }
				: undefined,
		];
		return loggedOut.filter(
			(item): item is NoteMenuItem => item !== undefined,
		);
	}

	const items: (NoteMenuItem | undefined)[] = [
		{
			icon: "ph-bookmark-simple ph-bold ph-lg",
			text: props.state?.isFavorited ? t("unfavorite") : t("favorite"),
			action: toggleFavorite,
		},
		{
			icon: "ph-paperclip ph-bold ph-lg",
			text: t("clip"),
			action: () => void clip(),
		},
		hasRemote
			? {
					icon: "ph-arrow-square-out ph-bold ph-lg",
					text: t("showOnRemote"),
					action: showOnRemote,
			  }
			: undefined,
		isAppearAuthor
			? (props.state?.isPinned ?? (current.pinnedNoteIds ?? []).includes(appearNote.id))
				? {
						icon: "ph-push-pin ph-bold ph-lg",
						text: t("unpin"),
						action: () => togglePin(false),
				  }
				: {
						icon: "ph-push-pin ph-bold ph-lg",
						text: t("pin"),
						action: () => togglePin(true),
				  }
			: undefined,
		{
			icon: "ph-clipboard-text ph-bold ph-lg",
			text: t("copyContent"),
			action: copyContent,
		},
		makeCopyLinkMenuItem(`${url}/notes/${appearNote.id}`, {
			onCopied: () => os.success(),
		}),
		hasRemote
			? {
					icon: "ph-link-simple ph-bold ph-lg",
					text: `${t("copyLink")} (${t("remote")})`,
					action: copyOriginal,
			  }
			: undefined,
		shareAvailable
			? {
					icon: "ph-share-network ph-bold ph-lg",
					text: t("share"),
					action: share,
			  }
			: undefined,
		{ separator: true },
		isMyRenote
			? {
					icon: "ph-trash ph-bold ph-lg",
					text: t("unrenote"),
					danger: true,
					action: unrenote,
			  }
			: undefined,
		isAppearAuthor
			? {
					icon: "ph-pencil-simple ph-bold ph-lg",
					text: t("edit"),
					action: () => void router.push(`/notes/${appearNote.id}/edit`),
			  }
			: undefined,
		isAppearAuthor
			? {
					icon: "ph-clock-counter-clockwise ph-bold ph-lg",
					text: "编辑历史",
					action: () => void router.push(`/notes/${appearNote.id}/history`),
			  }
			: undefined,
		isAppearAuthor
			? {
					icon: "ph-trash ph-bold ph-lg",
					text: t("delete"),
					danger: true,
					action: del,
			  }
			: undefined,
	];
	return items.filter((item): item is NoteMenuItem => item !== undefined);
}
