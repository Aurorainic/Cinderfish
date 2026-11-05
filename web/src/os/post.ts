// Ported from firefish@b425f61:packages/client/src/os.ts (`post`, `PostFormProps`) (AGPL-3.0-only)
//
// Firefish keeps `os.post` in the big os.ts barrel; Cinderfish splits it out so
// the composer components never have to import the barrel (which would create a
// cycle: os/index -> post -> MkPostFormDialog -> MkPostForm -> os/*).
import type { DriveFile, NoteDetailed, NoteVisibility } from "@/api/types";
import { popup } from "./state";
import MkPostFormDialog from "./MkPostFormDialog.vue";

export interface PostFormProps {
	reply?: NoteDetailed | null;
	renote?: NoteDetailed | null;
	initialText?: string;
	initialVisibility?: NoteVisibility;
	initialFiles?: DriveFile[];
	initialLocalOnly?: boolean;
	/** Accepted for Firefish compatibility; editing has no endpoint here. */
	initialNote?: NoteDetailed | null;
	fixed?: boolean;
	autofocus?: boolean;
}

/**
 * Opens the composer in a modal and resolves once it closes, whether a note was
 * posted or the user cancelled (contract §3.1).
 */
export function post(props: PostFormProps = {}): Promise<void> {
	return new Promise((resolve) => {
		popup(
			MkPostFormDialog,
			{ ...props },
			{
				closed: () => resolve(),
			},
			"closed",
		);
	});
}
