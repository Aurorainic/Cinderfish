import { computed, inject, ref, type ComputedRef, type InjectionKey, type Ref } from "vue";
import {
	createLayoutState,
	LAYOUT_INJECTION_KEY,
	type LayoutMode,
	type LayoutState,
} from "@/store/layout";
import { me } from "@/account/me";

export type { LayoutMode };

export interface LayoutContext extends LayoutState {
	isMobile: Ref<boolean>;
	isDesktop: ComputedRef<boolean>;
}

/** Provided by App.vue and injected by layout children and pages. */
export const LAYOUT_KEY: InjectionKey<LayoutContext> = Symbol(
	"cinderfish:layout-context",
);

export function useLayout(): LayoutContext {
	const context = inject(LAYOUT_KEY);
	if (!context) {
		throw new Error("useLayout() must be used under App.vue's layout provider");
	}
	return context;
}

function createMediaFlag(query: string): Ref<boolean> {
	const media = window.matchMedia(query);
	const flag = ref(media.matches);
	media.addEventListener("change", (event) => {
		flag.value = event.matches;
	});
	return flag;
}

/**
 * Resolves the shell the same way Firefish `init.ts` does:
 * `?zen` → zen, anonymous → visitor, `ui=deck` → deck, else universal.
 * The preference itself is owned by `store/layout.ts`.
 */
export function createLayoutContext(): LayoutContext {
	const state = createLayoutState({
		isSignedIn: () => me.value != null,
		isZenRequested: () => new URLSearchParams(location.search).has("zen"),
	});

	const isMobile = createMediaFlag("(max-width: 500px)");
	const isDesktop = computed(
		() => window.matchMedia("(min-width: 1025px)").matches,
	);

	return { ...state, isMobile, isDesktop };
}

export { LAYOUT_INJECTION_KEY };
