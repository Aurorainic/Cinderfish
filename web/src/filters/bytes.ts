// Ported from firefish@b425f61:packages/client/src/filters/bytes.ts (AGPL-3.0-only)
//
// Byte-size formatter. Kept byte-for-byte compatible with the Firefish filter
// (including its "0" and "?" edge cases) so the drive tiles and the detail
// views print the same strings as upstream.
export default function bytes(v: number | null | undefined, digits = 0): string {
	if (v == null) return "?";

	const sizes = ["B", "KB", "MB", "GB", "TB"];

	if (v === 0) return "0";

	const isMinus = v < 0;
	let value = v;
	if (isMinus) value = -value;

	const i = Math.floor(Math.log(value) / Math.log(1024));
	const size = sizes[i] ?? sizes[sizes.length - 1];

	return (
		(isMinus ? "-" : "") +
		(value / Math.pow(1024, i)).toFixed(digits).replace(/\.0+$/, "") +
		size
	);
}
