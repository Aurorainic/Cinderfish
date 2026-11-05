// Ported from firefish@b425f61:packages/client/src/scripts/use-tooltip.ts (AGPL-3.0-only)
//
// Hover/tap detection only: the caller renders the tooltip itself (see
// MkReactionTooltip). Firefish's copy is unchanged apart from typing.
import { onUnmounted, ref, watch, type Ref } from "vue";

export function useTooltip(
	elRef: Ref<HTMLElement | { $el: HTMLElement } | null | undefined>,
	onShow: (showing: Ref<boolean>) => void,
	delay = 300,
): void {
	let isHovering = false;

	// iOS (and Android) fire a synthetic mouseover right after a tap; ignoring
	// it avoids showing a tooltip when the user only touched the screen.
	let shouldIgnoreMouseover = false;

	let timeoutId: number;
	let changeShowingState: (() => void) | null = null;

	const open = (): void => {
		close();
		if (!isHovering) return;
		if (elRef.value == null) return;
		const el = elRef.value instanceof Element ? elRef.value : elRef.value.$el;
		// The element can already be gone from the DOM when the timer fires.
		if (!document.body.contains(el)) return;

		const showing = ref(true);
		onShow(showing);
		changeShowingState = () => {
			showing.value = false;
		};
	};

	const close = (): void => {
		if (changeShowingState != null) {
			changeShowingState();
			changeShowingState = null;
		}
	};

	const onMouseover = (): void => {
		if (isHovering) return;
		if (shouldIgnoreMouseover) return;
		isHovering = true;
		timeoutId = window.setTimeout(open, delay);
	};

	const onMouseleave = (): void => {
		if (!isHovering) return;
		isHovering = false;
		window.clearTimeout(timeoutId);
		close();
	};

	const onTouchstart = (): void => {
		shouldIgnoreMouseover = true;
		if (isHovering) return;
		isHovering = true;
		timeoutId = window.setTimeout(open, delay);
	};

	const onTouchend = (): void => {
		if (!isHovering) return;
		isHovering = false;
		window.clearTimeout(timeoutId);
		close();
	};

	let attached: HTMLElement | null = null;

	const stop = watch(
		elRef,
		() => {
			if (elRef.value == null) return;
			stop();
			const el = elRef.value instanceof Element ? elRef.value : elRef.value.$el;
			attached = el;
			el.addEventListener("mouseover", onMouseover, { passive: true });
			el.addEventListener("mouseleave", onMouseleave, { passive: true });
			el.addEventListener("touchstart", onTouchstart, { passive: true });
			el.addEventListener("touchend", onTouchend, { passive: true });
			el.addEventListener("click", close, { passive: true });
		},
		{ immediate: true, flush: "post" },
	);

	onUnmounted(() => {
		close();
		if (attached != null) {
			attached.removeEventListener("mouseover", onMouseover);
			attached.removeEventListener("mouseleave", onMouseleave);
			attached.removeEventListener("touchstart", onTouchstart);
			attached.removeEventListener("touchend", onTouchend);
			attached.removeEventListener("click", close);
		}
	});
}
