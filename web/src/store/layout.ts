import { computed, ref, type ComputedRef, type InjectionKey } from "vue";

/**
 * Layout selection, ported from Firefish's boot logic in `init.ts`:
 *
 *   ?zen                     -> Zen
 *   signed out               -> Visitor
 *   localStorage.ui = "deck" -> Deck
 *   otherwise                -> Universal
 *
 * Firefish reads the preference once at boot; keeping it reactive lets the
 * settings page switch layouts without a reload.
 */

export type LayoutMode = "universal" | "deck" | "zen" | "visitor";
export type LayoutPreference = "universal" | "deck";

const UI_STORAGE_KEY = "ui";

function readPreference(): LayoutPreference {
	try {
		return localStorage.getItem(UI_STORAGE_KEY) === "deck" ? "deck" : "universal";
	} catch {
		return "universal";
	}
}

export interface LayoutState {
	mode: ComputedRef<LayoutMode>;
	preference: ComputedRef<LayoutPreference>;
	setPreference: (value: LayoutPreference) => void;
}

export function createLayoutState(options: {
	isSignedIn: () => boolean;
	isZenRequested?: () => boolean;
}): LayoutState {
	const preference = ref<LayoutPreference>(readPreference());

	const mode = computed<LayoutMode>(() => {
		if (options.isZenRequested?.() ?? false) return "zen";
		if (!options.isSignedIn()) return "visitor";
		return preference.value === "deck" ? "deck" : "universal";
	});

	function setPreference(value: LayoutPreference): void {
		preference.value = value;
		try {
			localStorage.setItem(UI_STORAGE_KEY, value);
		} catch {
			// Preference is still live for this session.
		}
	}

	return {
		mode,
		preference: computed(() => preference.value),
		setPreference,
	};
}

/** App.vue provides the state; nested components can inject it. */
export const LAYOUT_INJECTION_KEY: InjectionKey<LayoutState> = Symbol(
	"cinderfish:layout",
);
