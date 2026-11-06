// Ported from firefish@b425f61:packages/client/src/scripts/keycode.ts (AGPL-3.0-only).
//
// Firefish builds a full a-z/0-9 alias table; the port generates the same table
// and keeps only the named aliases the templates use. Matching is against
// `KeyboardEvent.code`, so the layout-independence is the same.
const aliases: Record<string, string | string[]> = {
	esc: "Escape",
	enter: ["Enter", "NumpadEnter"],
	up: "ArrowUp",
	down: "ArrowDown",
	left: "ArrowLeft",
	right: "ArrowRight",
	plus: ["NumpadAdd", "Semicolon"],
};

for (let i = 97; i < 123; i++) {
	const char = String.fromCharCode(i);
	aliases[char] = `Key${char.toUpperCase()}`;
}
for (let i = 0; i < 10; i++) {
	aliases[i] = [`Numpad${i}`, `Digit${i}`];
}

/** Returns the `KeyboardEvent.code` values an alias (or raw code) stands for. */
export function onKey(input: string): string[] {
	const lower = input.toLowerCase();
	for (const [alias, codes] of Object.entries(aliases)) {
		if (alias.toLowerCase() !== lower) continue;
		return Array.isArray(codes) ? codes : [codes];
	}
	return [input];
}
