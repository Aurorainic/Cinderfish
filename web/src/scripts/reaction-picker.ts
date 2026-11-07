// Ported from firefish@b425f61:packages/client/src/scripts/reaction-picker.ts (AGPL-3.0-only)
//
// Firefish opens MkEmojiPickerDialog with `asReactionPicker: true`. Earlier waves
// used a compact in-file panel because the dialog did not exist yet; now that it
// does, the same dialog is used here too, so the picker's size/width/height
// preferences and the category/skin-tone UI are the ones that actually show.
import { popup, type PopupHandle } from "@/os";
import MkEmojiPickerDialog from "@/components/MkEmojiPickerDialog.vue";

class ReactionPicker {
	private handle: PopupHandle | null = null;

	public show(
		src: HTMLElement | null | undefined,
		onChosen: (reaction: string) => void,
		onClosed: () => void,
	): void {
		this.close();

		let handle: PopupHandle | null = null;
		handle = popup(
			MkEmojiPickerDialog,
			{
				src: src ?? null,
				asReactionPicker: true,
				showPinned: true,
			},
			{
				done: (reaction: string) => {
					onChosen(reaction);
					handle?.dispose();
				},
				closed: () => {
					if (this.handle === handle) this.handle = null;
					onClosed();
				},
			},
			"closed",
		);
		this.handle = handle;
	}

	public close(): void {
		if (this.handle == null) return;
		const handle = this.handle;
		this.handle = null;
		handle.dispose();
	}
}

export const reactionPicker = new ReactionPicker();
