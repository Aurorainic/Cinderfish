// Ported from firefish@b425f61:packages/client/src/scripts/array.ts (AGPL-3.0-only)
//
// Pure helpers. Firefish's version imports Predicate/EndoRelation from
// "./relation"; those are plain function types, inlined here so this module
// has no extra dependency.

/** Count the elements that satisfy the predicate. */
export function countIf<T>(f: (x: T) => boolean, xs: T[]): number {
	return xs.filter(f).length;
}

/** Count the elements equal to a. */
export function count<T>(a: T, xs: T[]): number {
	return countIf((x) => x === a, xs);
}

/** Concatenate an array of arrays. */
export function concat<T>(xss: T[][]): T[] {
	return ([] as T[]).concat(...xss);
}

/** Intersperse sep between the elements of xs. */
export function intersperse<T>(sep: T, xs: T[]): T[] {
	return concat(xs.map((x) => [sep, x])).slice(1);
}

/** The elements that are not equal to a. */
export function erase<T>(a: T, xs: T[]): T[] {
	return xs.filter((x) => x !== a);
}

/** All elements of xs that are not contained in ys, order kept. */
export function difference<T>(xs: T[], ys: T[]): T[] {
	const set = new Set(ys);
	return xs.filter((x) => !set.has(x));
}

/** Remove duplicates, keeping first occurrences. */
export function unique<T>(xs: T[]): T[] {
	return [...new Set(xs)];
}

export function uniqueBy<TValue, TKey>(
	values: TValue[],
	keySelector: (value: TValue) => TKey,
): TValue[] {
	const map = new Map<TKey, TValue>();
	for (const value of values) {
		const key = keySelector(value);
		if (!map.has(key)) map.set(key, value);
	}
	return [...map.values()];
}

export function sum(xs: number[]): number {
	return xs.reduce((a, b) => a + b, 0);
}

export function maximum(xs: number[]): number {
	return Math.max(...xs);
}

/** Split into groups; the concatenation of the result equals xs. */
export function groupBy<T>(f: (a: T, b: T) => boolean, xs: T[]): T[][] {
	const groups: T[][] = [];
	for (const x of xs) {
		const last = groups[groups.length - 1];
		if (last != null && f(last[0], x)) last.push(x);
		else groups.push([x]);
	}
	return groups;
}

/** groupBy on the equivalence induced by f. */
export function groupOn<T, S>(f: (x: T) => S, xs: T[]): T[][] {
	return groupBy((a, b) => f(a) === f(b), xs);
}

export function groupByX<T>(
	collections: T[],
	keySelector: (x: T) => string,
): Record<string, T[]> {
	return collections.reduce((obj: Record<string, T[]>, item: T) => {
		const key = keySelector(item);
		if (obj[key] === undefined) obj[key] = [];
		obj[key].push(item);
		return obj;
	}, {});
}

/** Compare two number arrays lexicographically. */
export function lessThan(xs: number[], ys: number[]): boolean {
	for (let i = 0; i < Math.min(xs.length, ys.length); i++) {
		if (xs[i] < ys[i]) return true;
		if (xs[i] > ys[i]) return false;
	}
	return xs.length < ys.length;
}

/** The longest prefix whose elements satisfy the predicate. */
export function takeWhile<T>(f: (x: T) => boolean, xs: T[]): T[] {
	const ys: T[] = [];
	for (const x of xs) {
		if (!f(x)) break;
		ys.push(x);
	}
	return ys;
}

export function cumulativeSum(xs: number[]): number[] {
	const ys = Array.from(xs);
	for (let i = 1; i < ys.length; i++) ys[i] += ys[i - 1];
	return ys;
}

export function toArray<T>(x: T | T[] | undefined): T[] {
	return Array.isArray(x) ? x : x != null ? [x] : [];
}

export function toSingle<T>(x: T | T[] | undefined): T | undefined {
	return Array.isArray(x) ? x[0] : x;
}

/** Split into fixed-size groups (last group may be shorter). */
export function chunk<T>(xs: T[], size: number): T[][] {
	if (!Number.isFinite(size) || size <= 0) return xs.length === 0 ? [] : [xs];
	const groups: T[][] = [];
	for (let i = 0; i < xs.length; i += size) groups.push(xs.slice(i, i + size));
	return groups;
}
