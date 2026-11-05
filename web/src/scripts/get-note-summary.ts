// Ported from firefish@b425f61:packages/client/src/scripts/get-note-summary.ts (AGPL-3.0-only)
//
// Degradation: Cinderfish's timeline projection (Note) has no `poll` key, only
// NoteDetailed does, so the poll marker is read through an optional shape.
import type { Note } from "@/api/types";

type SummarySource = Pick<Note, "cw" | "text" | "files"> & { poll?: unknown };

/**
 * Plain-text summary of a note, used by collapsed previews.
 */
export const getNoteSummary = (note: SummarySource): string => {
	let summary = "";

	// Body
	if (note.cw != null) {
		summary += note.cw;
	} else {
		summary += note.text ? note.text : "";
	}

	// Attached files
	if ((note.files || []).length !== 0) {
		const len = note.files?.length ?? 0;
		summary += ` 📎${len !== 1 ? ` (${len})` : ""}`;
	}

	// Attached poll
	if (note.poll) {
		summary += " 📊";
	}

	return summary.trim();
};
