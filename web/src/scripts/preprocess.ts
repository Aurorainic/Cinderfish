// Ported from firefish@b425f61:packages/client/src/scripts/preprocess.ts (AGPL-3.0-only)
//
// Firefish's preprocess() only expands custom KaTeX macros (`\newcommand`
// definitions pasted into the client settings). The expansion machinery does not
// need katex, so it is ported verbatim in scripts/katex-macro.ts.
import * as mfm from "mfm-js";
import { defaultStore } from "@/store/base";
import { expandKaTeXMacro } from "@/scripts/katex-macro";

export function preprocess(text: string): string {
	if (!defaultStore.state.enableCustomKaTeXMacro) return text;

	const parsedKaTeXMacro =
		localStorage.getItem("customKaTeXMacroParsed") ?? "{}";
	const maxNumberOfExpansions = 200; // to prevent infinite expansion loops

	const nodes = mfm.parse(text);

	for (const node of nodes) {
		if (node.type === "mathInline" || node.type === "mathBlock") {
			node.props.formula = expandKaTeXMacro(
				node.props.formula,
				parsedKaTeXMacro,
				maxNumberOfExpansions,
			);
		}
	}

	return mfm.toString(nodes);
}
