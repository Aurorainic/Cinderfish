// Ported from firefish@b425f61:packages/client/src/directives/size.ts (AGPL-3.0-only)
//
// Adds "max-width_<n>px" / "min-width_<n>px" classes from the element's own
// width, which the ported component SCSS keys off. Uses ResizeObserver and,
// while the element has no width yet (e.g. inside an inactive tab), waits for
// an IntersectionObserver before measuring again.
import type { Directive } from "vue";

type SizeValue = { max?: number[]; min?: number[] };

interface Mounting {
	value: SizeValue;
	resize: ResizeObserver;
	intersection?: IntersectionObserver;
	previousWidth: number;
}

interface ClassOrder {
	add: string[];
	remove: string[];
}

const mountings = new Map<Element, Mounting>();
const cache = new Map<string, ClassOrder>();

function getMaxClass(v: number): string {
	return "max-width_" + v + "px";
}

function getMinClass(v: number): string {
	return "min-width_" + v + "px";
}

function getClassOrder(width: number, queue: SizeValue): ClassOrder {
	return {
		add: [
			...(queue.max ? queue.max.filter((v) => width <= v).map(getMaxClass) : []),
			...(queue.min ? queue.min.filter((v) => width >= v).map(getMinClass) : []),
		],
		remove: [
			...(queue.max ? queue.max.filter((v) => width > v).map(getMaxClass) : []),
			...(queue.min ? queue.min.filter((v) => width < v).map(getMinClass) : []),
		],
	};
}

function applyClassOrder(el: Element, order: ClassOrder): void {
	el.classList.add(...order.add);
	el.classList.remove(...order.remove);
}

function getOrderName(width: number, queue: SizeValue): string {
	return (
		width +
		"|" +
		(queue.max ? queue.max.join(",") : "") +
		"|" +
		(queue.min ? queue.min.join(",") : "")
	);
}

function calc(el: Element): void {
	const info = mountings.get(el);
	const width = el.clientWidth;
	if (info == null) return;
	if (info.previousWidth === width) return;

	// Not laid out yet (inactive tab, hidden container): wait until visible.
	if (width === 0) {
		if (info.intersection == null) {
			info.intersection = new IntersectionObserver((entries) => {
				if (entries.some((entry) => entry.isIntersecting)) calc(el);
			});
		}
		info.intersection.observe(el);
		return;
	}
	if (info.intersection != null) {
		info.intersection.disconnect();
		info.intersection = undefined;
	}

	info.previousWidth = width;
	const name = getOrderName(width, info.value);
	let order = cache.get(name);
	if (order == null) {
		order = getClassOrder(width, info.value);
		cache.set(name, order);
	}
	applyClassOrder(el, order);
}

export default {
	mounted(el: Element, binding: { value: SizeValue }) {
		const resize = new ResizeObserver(() => {
			calc(el);
		});
		mountings.set(el, { value: binding.value, resize, previousWidth: 0 });
		calc(el);
		resize.observe(el);
	},

	updated(el: Element, binding: { value: SizeValue }) {
		const info = mountings.get(el);
		if (info == null) return;
		info.value = binding.value;
		calc(el);
	},

	unmounted(el: Element) {
		const info = mountings.get(el);
		if (info == null) return;
		info.resize.disconnect();
		info.intersection?.disconnect();
		mountings.delete(el);
	},
} as Directive<Element, SizeValue>;
