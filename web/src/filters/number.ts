// Ported from firefish@b425f61:packages/client/src/filters/number.ts (AGPL-3.0-only)
//
// Firefish's filter is a one-liner; the default export is kept so every ported
// call site reads `number(count)` exactly as upstream does.
export default function number(value: number | null | undefined): string {
	return value == null ? "N/A" : value.toLocaleString();
}
