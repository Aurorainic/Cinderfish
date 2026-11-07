import { onUnmounted, type Ref } from "vue";
import { useStream } from "../api/stream";
import type { NoteDetailed } from "../api/types";
import { api } from "../api/client";

/**
 * Port of Firefish `scripts/use-note-capture.ts`.
 *
 * Cinderfish's `subNote` stream emits `reacted`, `unreacted`, `deleted` and
 * (since wave 172) `pollVoted` (see docs/research/api-contract/streaming.md).
 * Firefish also handled `replied` and `updated`; the backend does not send
 * those, so the switch below only reacts to what actually arrives.
 */

export const deletedNoteIds = new Map<string, boolean>();

const noteRefs = new Map<string, Ref<NoteDetailed>[]>();

function addToNoteRefs(note: Ref<NoteDetailed>): void {
	const refs = noteRefs.get(note.value.id);
	if (refs == null) {
		noteRefs.set(note.value.id, [note]);
	} else {
		refs.push(note);
	}
}

function eachNote(id: string, callback: (note: Ref<NoteDetailed>) => void): void {
	const refs = noteRefs.get(id);
	if (refs == null) return;
	for (const note of refs) {
		// The ref may have been repointed to another note after a re-render.
		if (note.value.id === id) callback(note);
	}
}

export interface UseNoteCaptureOptions {
	note: Ref<NoteDetailed>;
	isDeletedRef?: Ref<boolean>;
	onReplied?: (note: NoteDetailed) => void;
}

interface ReactedBody {
	reaction?: string;
	emoji?: string | null;
	userId?: string;
}

export function useNoteCapture(options: UseNoteCaptureOptions): { close: () => void } {
	let closed = false;
	const note = options.note;
	addToNoteRefs(note);

	const stream = useStream();

	function markDeleted(): void {
		if (options.isDeletedRef != null) options.isDeletedRef.value = true;
		deletedNoteIds.set(note.value.id, true);

		if (note.value.replyId != null) {
			eachNote(note.value.replyId, (parent) => {
				parent.value.repliesCount = Math.max(0, parent.value.repliesCount - 1);
			});
		}
		if (note.value.renoteId != null) {
			eachNote(note.value.renoteId, (target) => {
				target.value.renoteCount = Math.max(0, target.value.renoteCount - 1);
			});
		}
	}

	const disposeNote = stream.subscribeNote(note.value.id, (event) => {
		if (closed || event.id !== note.value.id) return;

		if (event.type === "reacted") {
			const body = event.body as ReactedBody;
			if (body.reaction == null) return;
			note.value.reactions = {
				...note.value.reactions,
				[body.reaction]: (note.value.reactions[body.reaction] ?? 0) + 1,
			};
			note.value.reactionCount += 1;
			return;
		}

		if (event.type === "unreacted") {
			const body = event.body as ReactedBody;
			if (body.reaction == null) return;
			const next = Math.max(0, (note.value.reactions[body.reaction] ?? 0) - 1);
			const reactions = { ...note.value.reactions };
			if (next === 0) delete reactions[body.reaction];
			else reactions[body.reaction] = next;
			note.value.reactions = reactions;
			note.value.reactionCount = Math.max(0, note.value.reactionCount - 1);
			return;
		}

		if (event.type === "pollVoted") {
			// Everyone looking at the poll sees the new counts; the voter's own
			// client already applied its vote, so the duplicate is corrected
			// only when the numbers would otherwise drift (a second vote on a
			// multiple-choice poll arrives here as well).
			const body = event.body as { choice?: unknown };
			const choice = typeof body.choice === "number" ? body.choice : -1;
			const poll = note.value.poll;
			if (poll == null || choice < 0 || poll.choices[choice] == null) return;
			poll.choices[choice].votes += 1;
			poll.votesCount += 1;
			return;
		}

		if (event.type === "deleted") {
			markDeleted();
		}
	});

	onUnmounted(() => {
		closed = true;
		disposeNote();
	});

	return {
		close: () => {
			closed = true;
			disposeNote();
		},
	};
}

/** Refetch a note after a `deleted` event turned out to be wrong. */
export async function reloadNote(noteId: string): Promise<NoteDetailed> {
	return api<NoteDetailed>("notes/show", { noteId });
}
