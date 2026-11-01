// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/scripts/get-note-menu.ts
// (subset). Firefish's menu carries favorites, clips, thread watching and
// muting, translation, edit history and abuse reports; every one of those
// needs an endpoint Cinderfish lacks, so the port keeps only the items the
// backend can serve: copy content/link, remote link, pin/unpin and delete.
// TODO: favorites/clips/watch/mute/translate when those endpoints exist.
import type { Ref } from "vue";
import type { Note } from "../../api/types";
import { api } from "../../api/client";
import { i18n } from "../../i18n";
import type { MenuItem } from "../../types/menu";
import { isModerator, isSignedIn, me } from "../../me";
import { confirm } from "../../os";

// Mirrors the original's isRenote: a renote target with no own content.
export function isPureRenote(note: Note): boolean {
	return (
		note.renote != null &&
		note.text == null &&
		note.fileIds.length === 0 &&
		note.poll == null
	);
}

export function getNoteMenu(props: {
	note: Note;
	isDeleted: Ref<boolean>;
}): MenuItem[] {
	const appearNote = isPureRenote(props.note)
		? (props.note.renote as Note)
		: props.note;

	if (!isSignedIn(me)) {
		const menu: MenuItem[] = [];
		if (appearNote.url || appearNote.uri) {
			menu.push({
				icon: "ph-arrow-square-out",
				text: i18n.ts.remote,
				action: () => {
					window.open(appearNote.url ?? appearNote.uri ?? undefined, "_blank");
				},
			});
		}
		menu.push(
			{
				icon: "ph-clipboard-text",
				text: i18n.ts.copyContent,
				action: () => {
					navigator.clipboard.writeText(appearNote.text ?? "");
				},
			},
			{
				icon: "ph-link-simple",
				text: i18n.ts.copyLink,
				action: () => {
					navigator.clipboard.writeText(`${location.origin}/notes/${appearNote.id}`);
				},
			},
		);
		return menu;
	}

	const isAppearAuthor = appearNote.userId === me.id;
	const pinned = (me.pinnedNoteIds ?? []).includes(appearNote.id);

	const menu: MenuItem[] = [];

	if (appearNote.url || appearNote.uri) {
		menu.push({
			icon: "ph-arrow-square-out",
			text: i18n.ts.remote,
			action: () => {
				window.open(appearNote.url ?? appearNote.uri ?? undefined, "_blank");
			},
		});
	}

	menu.push(
		{
			icon: "ph-clipboard-text",
			text: i18n.ts.copyContent,
			action: () => {
				navigator.clipboard.writeText(appearNote.text ?? "");
			},
		},
		{
			icon: "ph-link-simple",
			text: i18n.ts.copyLink,
			action: () => {
				navigator.clipboard.writeText(`${location.origin}/notes/${appearNote.id}`);
			},
		},
	);

	if (isAppearAuthor) {
		menu.push({
			icon: "ph-push-pin",
			text: pinned ? i18n.ts.unpin : i18n.ts.pin,
			action: async () => {
				try {
					const next = pinned ? await api.unpin(appearNote.id) : await api.pin(appearNote.id);
					if (me) Object.assign(me, next);
				} catch (err) {
					confirm({
						type: "error",
						text: err instanceof Error ? err.message : i18n.ts.somethingHappened,
					});
				}
			},
		});
	}

	menu.push(null);

	if (isAppearAuthor || isModerator) {
		menu.push({
			icon: "ph-trash",
			text: i18n.ts.delete,
			danger: true,
			action: async () => {
				const { canceled } = await confirm({
					type: "warning",
					text: i18n.ts.deleteConfirm,
				});
				if (canceled) return;
				await api.deleteNote(appearNote.id);
				props.isDeleted.value = true;
			},
		});
	}

	return menu;
}
