// Ported from firefish@b425f61:packages/client/src/scripts/extract-mentions.ts (AGPL-3.0-only)
//
// mfm-js 0.23.3 is the pinned parser (same as Firefish). Mentions are not
// de-duplicated, matching the reference implementation.
import * as mfm from "mfm-js";

export function extractMentions(
	nodes: mfm.MfmNode[],
): mfm.MfmMention["props"][] {
	const mentionNodes = mfm.extract(nodes, (node) => node.type === "mention");
	return mentionNodes.map((node) => (node as mfm.MfmMention).props);
}
