// Ported from firefish@b425f61:packages/client/src/directives/hotkey.ts (AGPL-3.0-only)
//
// TODO(port): Firefish binds a keymap through scripts/hotkey.ts makeHotkey().
// That helper is not ported yet (it needs the keycode/autocomplete tables), so
// the directive is a no-op placeholder; templates keep v-hotkey="keymap".
import type { Directive } from "vue";

type HotkeyBinding = Record<string, ((ev: KeyboardEvent) => void) | undefined>;

export default {
	mounted(_el: HTMLElement, _binding: { value: HotkeyBinding; modifiers: Record<string, boolean> }) {
		// no-op (see TODO above)
	},

	unmounted(_el: HTMLElement, _binding: { value: HotkeyBinding }) {
		// no-op (see TODO above)
	},
} as Directive<HTMLElement, HotkeyBinding>;
