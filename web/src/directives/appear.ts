// Ported from firefish@b425f61:packages/client/src/directives/appear.ts (AGPL-3.0-only)
//
// Callback runs once, the first time the element enters the viewport (Firefish
// could fire repeatedly, but every ported use is an infinite-scroll loader on a
// freshly mounted button). A WeakMap holds the observer so the element keeps no
// expando properties.
import type { Directive } from "vue";

type AppearHandler = (() => void) | null | undefined;

const observers = new WeakMap<Element, IntersectionObserver>();

export default {
	mounted(el: Element, binding: { value: AppearHandler }) {
		const fn = binding.value;
		if (typeof fn !== "function") return;

		const observer = new IntersectionObserver((entries) => {
			if (!entries.some((entry) => entry.isIntersecting)) return;
			observer.disconnect();
			observers.delete(el);
			fn();
		});
		observers.set(el, observer);
		observer.observe(el);
	},

	unmounted(el: Element) {
		observers.get(el)?.disconnect();
		observers.delete(el);
	},
} as Directive<Element, AppearHandler>;
