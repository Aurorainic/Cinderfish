// Ported from firefish@b425f61:packages/client/src/filters/note.ts (AGPL-3.0-only)

/** Router path of a note: "/notes/<id>". */
export function notePage(note: { id: string }): string {
	return "/notes/" + note.id;
}
