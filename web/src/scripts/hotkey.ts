// Ported from firefish@b425f61:packages/client/src/scripts/hotkey.ts and
// keycode.ts (AGPL-3.0-only).
//
// The port keeps the pattern syntax the templates already use (`r`, `e|a|plus`,
// `up|k`, `(ctrl+k)`, `esc`) and matches `KeyboardEvent.code`, not the deprecated
// `keyCode`. Only the aliases used in this repository are mapped; every other
// name is matched as-is.
import { onKey } from "./keycode";

type Callback = (ev: KeyboardEvent) => void;

type Keymap = Record<string, Callback | undefined>;

type Pattern = {
	which: string[];
	ctrl?: boolean;
	shift?: boolean;
	alt?: boolean;
};

type Action = {
	patterns: Pattern[];
	callback: Callback;
	allowRepeat: boolean;
};

const IGNORE_ELEMENTS = ["input", "textarea"];

function parseKeymap(keymap: Keymap): Action[] {
	return Object.entries(keymap)
		.filter(([, callback]) => callback != null)
		.map(([rawPatterns, callback]): Action => {
		let patterns = rawPatterns;
		let allowRepeat = true;
		if (patterns.match(/^\(.*\)$/) !== null) {
			allowRepeat = false;
			patterns = patterns.slice(1, -1);
		}
		return {
			allowRepeat,
			callback: callback as Callback,
			patterns: patterns.split("|").map((part) => {
				const pattern: Pattern = { which: [], ctrl: false, alt: false, shift: false };
				for (const key of part.trim().split("+").map((x) => x.trim().toLowerCase())) {
					switch (key) {
						case "ctrl":
							pattern.ctrl = true;
							break;
						case "alt":
							pattern.alt = true;
							break;
						case "shift":
							pattern.shift = true;
							break;
						default:
							pattern.which = onKey(key).map((k) => k.toLowerCase());
					}
				}
				return pattern;
			}),
		};
	});
}

function match(ev: KeyboardEvent, patterns: Pattern[]): boolean {
	const key = ev.code.toLowerCase();
	return patterns.some(
		(pattern) =>
			pattern.which.includes(key) &&
			pattern.ctrl === ev.ctrlKey &&
			pattern.shift === ev.shiftKey &&
			pattern.alt === ev.altKey &&
			!ev.metaKey,
	);
}

/** Builds the keydown handler a `v-hotkey` element installs. */
export function makeHotkey(keymap: Keymap): (ev: KeyboardEvent) => void {
	const actions = parseKeymap(keymap);
	return (ev: KeyboardEvent) => {
		const active = document.activeElement;
		if (active != null) {
			if (IGNORE_ELEMENTS.some((el) => active.matches(el))) return;
			if (active.hasAttribute("contenteditable")) return;
		}
		for (const action of actions) {
			if (!match(ev, action.patterns)) continue;
			if (!action.allowRepeat && ev.repeat) return;
			ev.preventDefault();
			ev.stopPropagation();
			action.callback(ev);
			break;
		}
	};
}
