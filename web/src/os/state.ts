import { markRaw, ref, type Component } from "vue";

export type PopupPriority = "low" | "middle" | "high";

export interface PopupEntry {
	id: number;
	component: Component;
	props: Record<string, unknown>;
	events: Record<string, (...args: never[]) => void>;
}

export interface PopupHandle {
	dispose: () => void;
}

let popupIdCount = 0;
let uniqueId = 0;

/** Firefish's OS popup stack: every dialog, toast and window is an entry. */
export const popups = ref<PopupEntry[]>([]);

const zIndexes: Record<PopupPriority, number> = {
	low: 1_000_000,
	middle: 2_000_000,
	high: 3_000_000,
};

/**
 * Layer manager. Each claim moves the counter for that band by 100 so a later
 * popup always stacks above an earlier one without guessing CSS z-indexes.
 */
export function claimZIndex(priority: PopupPriority = "low"): number {
	zIndexes[priority] += 100;
	return zIndexes[priority];
}

export function getUniqueId(): string {
	return String(uniqueId++);
}

/**
 * Add a component to the popup stack. `disposeEvent` wires a component event
 * (usually `closed`) to removal; Vue resolves refs inside `props` for us.
 */
export function popup(
	component: Component,
	props: Record<string, unknown>,
	events: Record<string, (...args: never[]) => void> = {},
	disposeEvent?: string,
): PopupHandle {
	markRaw(component);

	const id = ++popupIdCount;
	const dispose = (): void => {
		// The zero-delay timeout mirrors Firefish: removing synchronously breaks
		// components that close themselves from their own event handler.
		window.setTimeout(() => {
			popups.value = popups.value.filter((entry) => entry.id !== id);
		}, 0);
	};

	const entry: PopupEntry = {
		id,
		component,
		props,
		events: disposeEvent == null
			? events
			: { ...events, [disposeEvent]: dispose },
	};

	popups.value = [...popups.value, entry];

	return { dispose };
}
