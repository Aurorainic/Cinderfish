// Ported from firefish@b425f61:packages/client/src/os.ts (`openEmojiPicker`) (AGPL-3.0-only)
//
// Contract §3.2: `openEmojiPicker(target, props, textarea)`. The picker writes the
// chosen shortcode / unicode emoji into `recentlyUsedEmojis` itself, so the
// reaction picker in scripts/reaction-picker.ts and the composer share one
// recents list without extra plumbing here.
import { popup } from "./state";
import MkEmojiPickerDialog from "@/components/MkEmojiPickerDialog.vue";

export interface OpenEmojiPickerProps {
	asReactionPicker?: boolean;
	showPinned?: boolean;
}

/** `insert-text-at-cursor` replacement: setRangeText + a bubbling input event. */
function insertAtCursor(
	el: HTMLTextAreaElement | HTMLInputElement,
	value: string,
): void {
	const start = el.selectionStart ?? el.value.length;
	const end = el.selectionEnd ?? start;
	el.setRangeText(value, start, end, "end");
	el.dispatchEvent(new Event("input", { bubbles: true }));
	el.focus();
}

export function openEmojiPicker(
	target: HTMLElement,
	props: OpenEmojiPickerProps = {},
	textarea: HTMLTextAreaElement | HTMLInputElement | null = null,
): void {
	popup(
		MkEmojiPickerDialog,
		{
			src: target,
			showPinned: props.showPinned ?? true,
			asReactionPicker: props.asReactionPicker ?? false,
		},
		{
			done: (emoji: string) => {
				if (textarea != null) insertAtCursor(textarea, emoji);
			},
		},
		"closed",
	);
}
