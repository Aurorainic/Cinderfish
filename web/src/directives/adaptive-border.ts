// Pattern adapted from Miuno (AGPL-3.0-only): frontend/packages/frontend/src/directives/adaptive-border.ts
//
// A field border is usually a fixed token (`--inputBorder`), which can vanish
// when the field sits on a surface of the same colour and clash on one that
// differs. This directive compares the element's own computed background with
// the nearest opaque background behind it:
//
//   * same colour -> the element is flush with its parent, so it exposes a
//                    visible divider-coloured edge;
//   * different   -> the element is already a raised surface, so it exposes
//                    its own fill colour instead of a contrasting outline.
//
// The result is written to `--cf-adaptive-border-color` on the element, not to
// `style.borderColor`: consumers opt in from their SCSS with
// `border-color: var(--cf-adaptive-border-color, var(--inputBorder));`, which
// leaves focused/hover rules able to override the border. An inline
// `style.borderColor` would beat those rules.
import type { Directive } from "vue";

const BORDER_PROPERTY = "--cf-adaptive-border-color";
/** Stops the ancestor walk on pathological trees; real UIs are far shallower. */
const MAX_ANCESTOR_DEPTH = 64;

const recalculators = new Map<HTMLElement, () => void>();
let observer: MutationObserver | null = null;

/** Alpha channel of a computed `background-color` value, 0..1. */
function alphaOf(color: string): number {
	const value = color.trim().toLowerCase();
	if (value === "" || value === "transparent") return 0;

	if (value.startsWith("#")) {
		const hex = value.slice(1);
		if (hex.length === 4) return Number.parseInt(hex[3]! + hex[3]!, 16) / 255;
		if (hex.length === 8) return Number.parseInt(hex.slice(6), 16) / 255;
		return 1;
	}

	const match = /^rgba?\(([^)]+)\)$/.exec(value);
	if (match == null) return 1;

	const parts = match[1]!.includes(",")
		? match[1]!.split(",").map((part) => part.trim())
		: match[1]!.split(/\s*\/\s*|\s+/).map((part) => part.trim());
	if (parts.length < 4) return 1;

	const alpha = parts[3]!;
	return alpha.endsWith("%") ? Number.parseFloat(alpha) / 100 : Number.parseFloat(alpha);
}

function isTransparent(color: string): boolean {
	return alphaOf(color) === 0;
}

/** Nearest opaque background colour visible behind `start`, if any. */
function surfaceColorBehind(start: Element | null): string | null {
	let current = start;
	let depth = 0;
	while (current != null && depth < MAX_ANCESTOR_DEPTH) {
		const background = window.getComputedStyle(current).backgroundColor;
		if (!isTransparent(background)) return background;
		current = current.parentElement;
		depth += 1;
	}
	return null;
}

function recalculate(el: HTMLElement): void {
	const own = window.getComputedStyle(el).backgroundColor;
	const behind = surfaceColorBehind(el.parentElement);
	// A transparent element shows the surface behind it, so it is flush by
	// definition; an opaque one stands out only when its fill differs.
	const effective = isTransparent(own) ? behind : own;

	if (effective == null) {
		el.style.removeProperty(BORDER_PROPERTY);
		return;
	}

	el.style.setProperty(
		BORDER_PROPERTY,
		effective === behind ? "var(--divider)" : effective,
	);
}

function ensureObserver(): void {
	if (observer != null || typeof MutationObserver === "undefined") return;
	observer = new MutationObserver(() => {
		for (const recalculateEntry of recalculators.values()) recalculateEntry();
	});
	// The theme engine re-applies its palette as inline custom properties and
	// `data-theme` on <html>; both need to refresh every adaptive border.
	observer.observe(document.documentElement, {
		attributes: true,
		attributeFilter: ["style", "class", "data-theme"],
	});
}

function releaseObserver(): void {
	if (observer == null || recalculators.size > 0) return;
	observer.disconnect();
	observer = null;
}

export default {
	mounted(el: HTMLElement) {
		recalculators.set(el, () => recalculate(el));
		ensureObserver();
		recalculate(el);
	},

	updated(el: HTMLElement) {
		recalculate(el);
	},

	unmounted(el: HTMLElement) {
		recalculators.delete(el);
		el.style.removeProperty(BORDER_PROPERTY);
		releaseObserver();
	},
} as Directive<HTMLElement>;
