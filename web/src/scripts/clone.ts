// Ported from firefish@b425f61:packages/client/src/scripts/clone.ts (AGPL-3.0-only)
//
// Firefish notes that structuredClone() is slow and hand-rolls the copy. The
// generic parameter is unconstrained here (Firefish required a recursive
// Cloneable alias, which rejects ordinary interfaces with optional fields).

export function deepClone<T>(x: T): T {
	if (typeof x !== "object" || x === null) return x;
	if (Array.isArray(x)) return x.map((item) => deepClone(item)) as unknown as T;
	const source = x as Record<string, unknown>;
	const copy: Record<string, unknown> = {};
	for (const [key, value] of Object.entries(source)) {
		copy[key] = deepClone(value);
	}
	return copy as T;
}
