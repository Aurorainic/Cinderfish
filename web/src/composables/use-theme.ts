// Light / dark / auto theme controller.
//
// The theme engine (parsing, compiling, applying CSS variables) lives in
// web/src/themes; this file owns only the reactive choice: which mode the user
// picked, whether `auto` currently resolves to dark, and which concrete theme
// id is active for each mode. The preference is stored under
// `cinderfish.theme` (light | dark | auto), the key index.html's no-flash
// bootstrap reads; the two palette ids live under `cinderfish.themeId.light`
// and `cinderfish.themeId.dark`.
import { computed, ref, type ComputedRef, type Ref } from "vue";
import {
	applyTheme,
	defaultThemeIds,
	findTheme,
	getDefaultTheme,
	prefersDarkMode,
	themeMode,
	type ColorSchema,
	type Theme,
	type ThemeMode as PaletteMode,
} from "../themes";

export type ThemeMode = "light" | "dark" | "auto";

const MODE_KEY = "cinderfish.theme";
/** Single-palette key shipped before wave 4; migrated into a mode slot. */
const LEGACY_THEME_KEY = "cinderfish.themeId";
const THEME_KEYS: Readonly<Record<PaletteMode, string>> = {
	light: "cinderfish.themeId.light",
	dark: "cinderfish.themeId.dark",
};
const SYNC_CHANNEL = "cinderfish:theme";

function storage(): Storage | null {
	try {
		return typeof localStorage === "undefined" ? null : localStorage;
	} catch {
		return null;
	}
}

function readMode(): ThemeMode {
	try {
		const raw = storage()?.getItem(MODE_KEY);
		if (raw === "light" || raw === "dark" || raw === "auto") return raw;
	} catch {
		// ignore
	}
	return "auto";
}

function readSlot(key: string): string | null {
	try {
		const raw = storage()?.getItem(key);
		return raw != null && raw.length > 0 ? raw : null;
	} catch {
		return null;
	}
}

function writeSlots(ids: Readonly<Record<PaletteMode, string | null>>): void {
	const store = storage();
	if (store == null) return;
	try {
		for (const mode of ["light", "dark"] as const) {
			const id = ids[mode];
			if (id == null) store.removeItem(THEME_KEYS[mode]);
			else store.setItem(THEME_KEYS[mode], id);
		}
	} catch {
		// ignore
	}
}

/**
 * Reads both palette slots and migrates the pre-wave-4 single id into the slot
 * matching that theme's mode. The old key is removed so the migration runs
 * exactly once.
 */
function readThemeIds(): Record<PaletteMode, string | null> {
	const ids: Record<PaletteMode, string | null> = {
		light: readSlot(THEME_KEYS.light),
		dark: readSlot(THEME_KEYS.dark),
	};
	const legacy = readSlot(LEGACY_THEME_KEY);
	if (legacy == null) return ids;

	const theme = findTheme(legacy);
	const mode: PaletteMode = theme != null ? themeMode(theme) : "light";
	if (ids[mode] == null) ids[mode] = legacy;

	try {
		storage()?.removeItem(LEGACY_THEME_KEY);
	} catch {
		// ignore
	}
	writeSlots(ids);
	return ids;
}

export const modeRef: Ref<ThemeMode> = ref<ThemeMode>(readMode());

const initialThemeIds = readThemeIds();

/** Chosen palette id per mode (null = that mode's default). */
export const selectedThemeIds: {
	light: Ref<string | null>;
	dark: Ref<string | null>;
} = {
	light: ref<string | null>(initialThemeIds.light),
	dark: ref<string | null>(initialThemeIds.dark),
};

const systemPrefersDark: Ref<boolean> = ref<boolean>(prefersDarkMode());

export const resolvedMode: ComputedRef<ColorSchema> = computed<ColorSchema>(() =>
	modeRef.value === "auto"
		? systemPrefersDark.value
			? "dark"
			: "light"
		: modeRef.value,
);

/** The active theme object. `findTheme("light" | "dark")` maps to the default. */
export const activeTheme: ComputedRef<Theme> = computed<Theme>(() => {
	const id = selectedThemeIds[resolvedMode.value].value;
	if (id != null) {
		const found = findTheme(id);
		if (found != null) return found;
	}
	return getDefaultTheme(resolvedMode.value);
});

/** Theme id, e.g. "l-rosepinedawn". */
export const themeName: ComputedRef<string> = computed(() => activeTheme.value.id);

let initialized = false;
let mediaQuery: MediaQueryList | null = null;
let syncChannel: BroadcastChannel | null = null;

function persist(): void {
	const store = storage();
	if (store == null) return;
	try {
		store.setItem(MODE_KEY, modeRef.value);
	} catch {
		// ignore
	}
	writeSlots({
		light: selectedThemeIds.light.value,
		dark: selectedThemeIds.dark.value,
	});
}

export function applyCurrentTheme(): void {
	applyTheme(activeTheme.value);
}

export function setMode(next: ThemeMode): void {
	modeRef.value = next;
	persist();
	applyCurrentTheme();
	syncChannel?.postMessage({ mode: next });
}

function broadcastThemeIds(): void {
	syncChannel?.postMessage({
		themeIds: {
			light: selectedThemeIds.light.value,
			dark: selectedThemeIds.dark.value,
		},
	});
}

/** Selects (or clears) the palette for one mode. */
export function setThemeForMode(mode: PaletteMode, id: string | null): void {
	if (id != null && findTheme(id) == null) return;
	selectedThemeIds[mode].value = id;
	persist();
	applyCurrentTheme();
	broadcastThemeIds();
}

/**
 * Accepts "light" | "dark" | "auto" or a theme id. Mode names switch mode and
 * keep the mode's palette; an id is stored in the slot of the theme's own
 * mode.
 */
export function setTheme(name: string): void {
	if (name === "light" || name === "dark" || name === "auto") {
		setMode(name);
		return;
	}

	const theme = findTheme(name);
	if (theme != null) setThemeForMode(themeMode(theme), name);
}

/** Switching between light and dark keeps each mode's palette. */
export function toggleMode(): void {
	setMode(resolvedMode.value === "dark" ? "light" : "dark");
}

export function initTheme(): void {
	if (initialized) return;
	initialized = true;

	if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
		mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
		const onChange = (event: MediaQueryListEvent): void => {
			systemPrefersDark.value = event.matches;
			if (modeRef.value === "auto") applyCurrentTheme();
		};
		if (typeof mediaQuery.addEventListener === "function") {
			mediaQuery.addEventListener("change", onChange);
		} else {
			// Safari < 14
			mediaQuery.addListener(onChange);
		}
	}

	if (typeof BroadcastChannel !== "undefined") {
		syncChannel = new BroadcastChannel(SYNC_CHANNEL);
		syncChannel.addEventListener(
			"message",
			(
				event: MessageEvent<{
					mode?: ThemeMode;
					themeIds?: { light: string | null; dark: string | null };
					themeId?: string | null;
				}>,
			) => {
				const nextMode = event.data?.mode;
				if (nextMode === "light" || nextMode === "dark" || nextMode === "auto") {
					modeRef.value = nextMode;
					applyCurrentTheme();
				}
				const ids = event.data?.themeIds;
				if (ids != null) {
					selectedThemeIds.light.value = ids.light;
					selectedThemeIds.dark.value = ids.dark;
					applyCurrentTheme();
				}
			},
		);
	}

	applyCurrentTheme();
}

export interface UseThemeReturn {
	/** Id of the active theme (e.g. "l-rosepinedawn"). */
	themeName: ComputedRef<string>;
	/** "light" or "dark" after resolving `auto`. */
	resolvedMode: ComputedRef<ColorSchema>;
	/** The user's raw preference, including `auto`. */
	mode: Ref<ThemeMode>;
	/** The chosen palette id per mode. */
	selectedThemeIds: {
		light: Ref<string | null>;
		dark: Ref<string | null>;
	};
	theme: ComputedRef<Theme>;
	setTheme: (name: string) => void;
	setThemeForMode: (mode: PaletteMode, id: string | null) => void;
	setMode: (mode: ThemeMode) => void;
	toggleMode: () => void;
}

export function useTheme(): UseThemeReturn {
	initTheme();

	return {
		themeName,
		resolvedMode,
		mode: modeRef,
		selectedThemeIds,
		theme: activeTheme,
		setTheme,
		setThemeForMode,
		setMode,
		toggleMode,
	};
}

export { defaultThemeIds };
