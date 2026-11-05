// Ported from firefish@b425f61:packages/client/src/scripts/scroll.ts (AGPL-3.0-only)
type ScrollBehavior = "auto" | "smooth" | "instant";

export function getScrollContainer(el: HTMLElement | null): HTMLElement | null {
	if (el == null || el.tagName === "HTML") return null;
	const overflow = window.getComputedStyle(el).getPropertyValue("overflow-y");
	if (overflow === "scroll" || overflow === "auto") {
		return el;
	} else {
		return getScrollContainer(el.parentElement);
	}
}

export function getScrollPosition(el: Element | null): number {
	const container = getScrollContainer(el as HTMLElement | null);
	return container == null ? window.scrollY : container.scrollTop;
}

export function isTopVisible(el: Element | null): boolean {
	if (el == null) return true;
	const scrollTop = getScrollPosition(el);
	// Relative position of `el` inside its scroll container is not computed;
	// Firefish has the same approximation.
	const topPosition = (el as HTMLElement).offsetTop;

	return scrollTop <= topPosition;
}

export function isBottomVisible(
	el: HTMLElement,
	tolerance = 1,
	container = getScrollContainer(el),
): boolean {
	if (container != null)
		return (
			el.scrollHeight <=
			container.clientHeight + Math.abs(container.scrollTop) + tolerance
		);
	return el.scrollHeight <= window.innerHeight + window.scrollY + tolerance;
}

export function onScrollTop(el: Element, cb: () => void): void {
	const container = (getScrollContainer(el as HTMLElement) ??
		window) as unknown as EventTarget;
	const onScroll = (): void => {
		if (!document.body.contains(el)) return;
		if (isTopVisible(el)) {
			cb();
			container.removeEventListener("scroll", onScroll);
		}
	};
	container.addEventListener("scroll", onScroll, { passive: true });
}

export function onScrollBottom(el: Element, cb: () => void): void {
	const container = (getScrollContainer(el as HTMLElement) ??
		window) as unknown as EventTarget;
	const onScroll = (): void => {
		if (!document.body.contains(el)) return;
		const target = el as HTMLElement;
		const pos = getScrollPosition(el);
		if (pos + target.clientHeight > target.scrollHeight - 1) {
			cb();
			container.removeEventListener("scroll", onScroll);
		}
	};
	container.addEventListener("scroll", onScroll, { passive: true });
}

export function scroll(
	el: Element,
	options: {
		top?: number;
		left?: number;
		behavior?: ScrollBehavior;
	},
): void {
	const container = getScrollContainer(el as HTMLElement);
	if (container == null) {
		window.scroll(options);
	} else {
		container.scroll(options);
	}
}

export function scrollToTop(
	el: Element | null,
	options: { behavior?: ScrollBehavior } = {},
): void {
	if (el == null) {
		window.scroll({ top: 0, ...options });
		return;
	}
	scroll(el, { top: 0, ...options });
}

export function scrollToBottom(
	el: Element,
	options: { behavior?: ScrollBehavior } = {},
): void {
	scroll(el, { top: 99999, ...options });
}

export function isBottom(el: Element, asobi = 0): boolean {
	const container = getScrollContainer(el as HTMLElement);
	const target = el as HTMLElement;
	const current = container
		? target.scrollTop + target.offsetHeight
		: window.scrollY + window.innerHeight;
	const max = container ? target.scrollHeight : document.body.offsetHeight;
	return current >= max - asobi;
}
