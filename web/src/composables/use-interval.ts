import { onMounted, onUnmounted } from "vue";

export interface UseIntervalOptions {
	immediate?: boolean;
	afterMounted?: boolean;
}

/**
 * Port of Firefish `scripts/use-interval.ts`.
 * The interval is cleared automatically when the component unmounts.
 */
export function useInterval(
	fn: () => void,
	interval: number,
	options: UseIntervalOptions = {},
): void {
	if (Number.isNaN(interval)) return;

	let intervalId: number | null = null;

	if (options.afterMounted) {
		onMounted(() => {
			if (options.immediate) fn();
			intervalId = window.setInterval(fn, interval);
		});
	} else {
		if (options.immediate) fn();
		intervalId = window.setInterval(fn, interval);
	}

	onUnmounted(() => {
		if (intervalId != null) window.clearInterval(intervalId);
	});
}
