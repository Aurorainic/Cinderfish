// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/scripts/get-note-summary.ts
// (plain-text variant: the original feeds the summary through Mfm with
// `plain`; Cinderfish renders it as text since inline emoji codes are not
// resolved here).
import type { Note } from "../../api/types";

export function getNoteSummary(note: Note): string {
	let summary = "";

	if (note.cw != null) {
		summary += note.cw;
	} else {
		summary += note.text ?? "";
	}

	if ((note.files ?? []).length !== 0) {
		const len = note.files?.length ?? 0;
		summary += ` 📎${len !== 1 ? ` (${len})` : ""}`;
	}

	if (note.poll) {
		summary += " 📊";
	}

	return summary.trim();
}
