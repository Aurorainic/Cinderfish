// Ported from firefish@b425f61:packages/client/src/scripts/get-note-menu.ts (AGPL-3.0-only)
//
// Degradation: only entries whose endpoints exist in Cinderfish are rendered.
// Dropped from the Firefish menu (no endpoint, see the capability matrix):
// notes/state (so the favourite entry cannot know the current state and always
// creates one), watch / thread mute, translate, delete-and-edit, report
// abuse, promote, clips, plugin actions and the user submenu. Edit and the
// history page are Cinderfish's own additions.
import type { Ref } from "vue";
import type { Note, NoteDetailed, NoteState } from "@/api/types";
import { me } from "@/account";
import { url } from "@/config";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { router } from "@/router";
import copyToClipboard from "@/scripts/copy-to-clipboard";

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
	return getNoteMenu({ ...props, state });
}

export function getNoteMenu(props: NoteMenuProps): NoteMenuItem[] {
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

	function copyLink(): void {
		copyToClipboard(`${url}/notes/${appearNote.id}`);
		os.success();
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
			{
				icon: "ph-link-simple ph-bold ph-lg",
				text: t("copyLink"),
				action: copyLink,
			},
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
		{
			icon: "ph-link-simple ph-bold ph-lg",
			text: t("copyLink"),
			action: copyLink,
		},
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
