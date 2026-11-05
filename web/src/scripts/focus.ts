// Ported from firefish@b425f61:packages/client/src/scripts/focus.ts (AGPL-3.0-only)
//
// Keyboard navigation between sibling list items (MkMenu uses it for arrows).
// Firefish's focusNext recursed into focusPrev by mistake, which made Down walk
// upwards; both helpers share one directional walk here.

function walk(start: Element | null, direction: 1 | -1, scroll: boolean): void {
	let el: Element | null = start;
	while (el != null) {
		if (el.hasAttribute("tabindex")) {
			(el as HTMLElement).focus({ preventScroll: !scroll });
			return;
		}
		el =
			direction === 1 ? el.nextElementSibling : el.previousElementSibling;
	}
}

export function focusPrev(el: Element | null, self = false, scroll = true): void {
	if (el == null) return;
	const start = self ? el : el.previousElementSibling;
	if (start == null) return;
	walk(start, -1, scroll);
}

export function focusNext(el: Element | null, self = false, scroll = true): void {
	if (el == null) return;
	const start = self ? el : el.nextElementSibling;
	if (start == null) return;
	walk(start, 1, scroll);
}
