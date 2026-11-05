// Ported from firefish@b425f61:packages/client/src/directives/tooltip.ts (AGPL-3.0-only)
//
// DEGRADED: Firefish renders a styled Mfm tooltip through os.popup with
// placement and delay handling. There is no MkTooltip component yet, so the
// shim writes the text into the native "title" attribute, which keeps the
// information and is keyboard/AT visible. Modifiers (.noDelay, .bottom, .mfm,
// .noLabel) are accepted and ignored except that the title is always set -
// .noLabel only suppressed Firefish's aria-label.
import type { Directive } from "vue";

type TooltipValue = string | null | undefined;

function apply(el: HTMLElement, value: TooltipValue): void {
	if (typeof value === "string" && value !== "") {
		el.setAttribute("title", value);
	} else {
		el.removeAttribute("title");
	}
}

export default {
	mounted(el: HTMLElement, binding: { value: TooltipValue }) {
		apply(el, binding.value);
	},

	updated(el: HTMLElement, binding: { value: TooltipValue }) {
		apply(el, binding.value);
	},

	unmounted(el: HTMLElement) {
		el.removeAttribute("title");
	},
} as Directive<HTMLElement, TooltipValue>;
