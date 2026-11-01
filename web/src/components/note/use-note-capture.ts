// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/scripts/use-note-capture.ts
// (adapted: the original opens a dedicated per-note channel through the
// connection pool; Cinderfish's backend multiplexes note updates on "main",
// so the port listens there and filters by note id).
import { onBeforeUnmount, onMounted, type Ref, watch } from "vue";
import type { Note } from "../../api/types";
import type { NoteUpdatedEvent } from "../../stores/streaming-frames";
import { useChannel } from "../../stores/streaming";

export function useNoteCapture(options: {
	rootEl: Ref<HTMLElement | null>;
	note: Ref<Note>;
	isDeletedRef: Ref<boolean>;
}) {
	const { note, isDeletedRef } = options;

	let disconnect: (() => void) | undefined;

	function capture() {
		if (disconnect) disconnect();

		const channel = useChannel("main");
		const handler = (body: unknown) => {
			const event = body as NoteUpdatedEvent;
			if (!event || event.id !== note.value.id) return;
			apply(event);
		};
		channel.on("noteUpdated", handler);
		disconnect = () => {
			channel.off("noteUpdated", handler);
			channel.dispose();
		};
	}

	function apply(event: NoteUpdatedEvent) {
		switch (event.type) {
			case "reacted": {
				const reaction = (event.body as { reaction?: string })?.reaction;
				if (reaction) {
					note.value.reactions[reaction] = (note.value.reactions[reaction] ?? 0) + 1;
					note.value.reactionCount += 1;
				}
				break;
			}
			case "unreacted": {
				const reaction = (event.body as { reaction?: string })?.reaction;
				if (reaction && note.value.reactions[reaction] !== undefined) {
					note.value.reactions[reaction] -= 1;
					if (note.value.reactions[reaction] <= 0) delete note.value.reactions[reaction];
					note.value.reactionCount = Math.max(0, note.value.reactionCount - 1);
				}
				break;
			}
			case "deleted": {
				isDeletedRef.value = true;
				break;
			}
		}
	}

	function decapture() {
		disconnect?.();
		disconnect = undefined;
	}

	watch(
		() => note.value.id,
		() => capture(),
	);

	onMounted(() => capture());
	onBeforeUnmount(() => decapture());
}
