// Ported from firefish@b425f61:packages/client/src/directives/tooltip.ts (AGPL-3.0-only)
//
// Firefish renders a styled tooltip through os.popup. The port now does the
// same with CfTooltip: hover/focus opens it after 500ms (or immediately with
// `.noDelay`), leaving closes after 100ms, and the text is also exposed as
// `aria-label` unless `.noLabel` is set. `.mfm` is accepted and ignored: the
// bubble renders plain text.
import { defineAsyncComponent, ref, type Directive, type Ref } from "vue";
import { popup } from "@/os";

type TooltipValue = string | null | undefined;

interface Binding {
	value: TooltipValue;
	modifiers: Record<string, boolean>;
}

class Tooltip {
	private el: HTMLElement;
	private showTimer: number | null = null;
	private hideTimer: number | null = null;
	private showing: Ref<boolean> | null = null;

	constructor(el: HTMLElement) {
		this.el = el;
		this.el.addEventListener("mouseenter", this.onEnter);
		this.el.addEventListener("mouseleave", this.onLeave);
		this.el.addEventListener("focusin", this.onEnter);
		this.el.addEventListener("focusout", this.onLeave);
		this.el.addEventListener("click", this.onClick);
	}

	private text: string = "";
	private noDelay = false;
	private direction: "top" | "bottom" = "top";

	configure(value: TooltipValue, binding: Binding): void {
		this.text = typeof value === "string" ? value : "";
		this.noDelay = binding.modifiers.noDelay === true;
		this.direction = binding.modifiers.bottom === true ? "bottom" : "top";
		if (this.text === "") {
			this.el.removeAttribute("aria-label");
			return;
		}
		if (binding.modifiers.noLabel === true) this.el.removeAttribute("aria-label");
		else this.el.setAttribute("aria-label", this.text);
	}

	private show = (): void => {
		if (this.text === "" || this.showing != null) return;
		const showing = ref(true);
		this.showing = showing;
		void popup(
			defineAsyncComponent(() => import("@/components/CfTooltip.vue")),
			{
				showing,
				text: this.text,
				source: this.el,
				direction: this.direction,
			},
			{},
			"closed",
		);
	};

	private hide = (): void => {
		if (this.showing == null) return;
		this.showing.value = false;
		this.showing = null;
	};

	private clearTimers(): void {
		if (this.showTimer != null) window.clearTimeout(this.showTimer);
		if (this.hideTimer != null) window.clearTimeout(this.hideTimer);
		this.showTimer = null;
		this.hideTimer = null;
	}

	private onEnter = (): void => {
		this.clearTimers();
		this.showTimer = window.setTimeout(this.show, this.noDelay ? 0 : 500);
	};

	private onLeave = (): void => {
		this.clearTimers();
		this.hideTimer = window.setTimeout(this.hide, 100);
	};

	private onClick = (): void => {
		this.clearTimers();
		this.hide();
	};

	detach(): void {
		this.clearTimers();
		this.el.removeEventListener("mouseenter", this.onEnter);
		this.el.removeEventListener("mouseleave", this.onLeave);
		this.el.removeEventListener("focusin", this.onEnter);
		this.el.removeEventListener("focusout", this.onLeave);
		this.el.removeEventListener("click", this.onClick);
		this.hide();
		this.el.removeAttribute("aria-label");
	}
}

const instances = new WeakMap<HTMLElement, Tooltip>();

function ensure(el: HTMLElement, binding: Binding): Tooltip {
	let tooltip = instances.get(el);
	if (tooltip == null) {
		tooltip = new Tooltip(el);
		instances.set(el, tooltip);
	}
	tooltip.configure(binding.value, binding);
	return tooltip;
}

export default {
	mounted(el: HTMLElement, binding: Binding) {
		ensure(el, binding);
	},

	updated(el: HTMLElement, binding: Binding) {
		ensure(el, binding);
	},

	unmounted(el: HTMLElement) {
		const tooltip = instances.get(el);
		if (tooltip != null) {
			tooltip.detach();
			instances.delete(el);
		}
	},
} as Directive<HTMLElement, TooltipValue>;
