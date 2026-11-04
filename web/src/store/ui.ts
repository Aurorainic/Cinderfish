import {
	computed,
	inject,
	provide,
	ref,
	type ComputedRef,
	type InjectionKey,
} from "vue";
import { createPizzax } from "./pizzax";
import { store as baseStore } from "./base";

/**
 * Shell layout selection. Firefish chooses its root component in `init.ts`
 * (`?zen` → zen, signed out → visitor, `ui === "deck"` → deck, else universal).
 * This store holds the persisted user choice; `App.vue` adds the session and
 * query-string conditions.
 */

export type DeviceKind = "smartphone" | "tablet" | "desktop";
export type ShellLayout = "universal" | "deck";

export const uiStore = createPizzax("ui", {
	layout: { where: "device", default: "universal" as ShellLayout },
	deckOnDesktop: { where: "device", default: false },
});

function detectDeviceKind(): DeviceKind {
	if (typeof window === "undefined") return "desktop";
	const override = baseStore.state.overridedDeviceKind;
	if (override) return override;
	const width = window.innerWidth;
	const touch =
		typeof navigator !== "undefined" &&
		/Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
	if (width <= 500 || (touch && width <= 700)) return "smartphone";
	if (width <= 1024) return "tablet";
	return "desktop";
}

export interface UiLayoutContext {
	layout: ComputedRef<ShellLayout>;
	setLayout: (layout: ShellLayout) => void;
	toggleDeck: () => void;
	deviceKind: ComputedRef<DeviceKind>;
	isMobile: ComputedRef<boolean>;
	isDesktop: ComputedRef<boolean>;
	zen: ComputedRef<boolean>;
}

export const UI_LAYOUT_KEY: InjectionKey<UiLayoutContext> = Symbol(
	"cinderfish:ui-layout",
);

export function createUiLayoutContext(): UiLayoutContext {
	const viewport = ref(
		typeof window === "undefined" ? 1280 : window.innerWidth,
	);
	const zen = computed<boolean>(
		() =>
			typeof window !== "undefined" &&
			new URLSearchParams(window.location.search).has("zen"),
	);

	if (typeof window !== "undefined") {
		window.addEventListener("resize", () => {
			viewport.value = window.innerWidth;
		});
	}

	const deviceKind = computed<DeviceKind>(() => {
		const override = baseStore.state.overridedDeviceKind;
		if (override) return override;
		if (viewport.value <= 500) return "smartphone";
		if (viewport.value <= 1024) return "tablet";
		return "desktop";
	});

	const layout = computed<ShellLayout>(() => uiStore.state.layout);

	return {
		layout,
		setLayout: (next) => uiStore.set("layout", next),
		toggleDeck: () =>
			uiStore.set(
				"layout",
				uiStore.state.layout === "deck" ? "universal" : "deck",
			),
		deviceKind,
		isMobile: computed(() => deviceKind.value === "smartphone"),
		isDesktop: computed(() => deviceKind.value === "desktop"),
		zen,
	};
}

export function provideUiLayout(
	context: UiLayoutContext = createUiLayoutContext(),
): UiLayoutContext {
	provide(UI_LAYOUT_KEY, context);
	return context;
}

export function useUiLayout(): UiLayoutContext {
	const context = inject(UI_LAYOUT_KEY, null);
	if (context == null) {
		throw new Error(
			"useUiLayout() must be used under a component that called provideUiLayout()",
		);
	}
	return context;
}

export { detectDeviceKind };
