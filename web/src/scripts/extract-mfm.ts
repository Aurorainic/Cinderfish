// Ported from firefish@b425f61:packages/client/src/scripts/extract-mfm.ts (AGPL-3.0-only)
import * as mfm from "mfm-js";

const animatedMfm = [
	"tada",
	"jelly",
	"twitch",
	"shake",
	"spin",
	"jump",
	"bounce",
	"rainbow",
	"fade",
];

export function extractMfmWithAnimation(nodes: mfm.MfmNode[]): string[] {
	const mfmNodes = mfm.extract(nodes, (node) => {
		return node.type === "fn" && animatedMfm.indexOf(node.props.name) > -1;
	});
	return mfmNodes.map((x) => (x.type === "fn" ? x.props.name : ""));
}
