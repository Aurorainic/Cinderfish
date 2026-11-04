import { compileTheme } from "./compile";
import type { CompiledTheme, Theme } from "./types";
import { themeMode } from "./types";

const THEME_TRANSITION_MS = 1000;
let transitionTimer: number | undefined;

export interface ApplyThemeOptions {
	/**
	 * Firefish persists the compiled `--var` map under `localStorage.theme`
	 * and the schema under `localStorage.colorSchema`. The reactive store
	 * persists the theme *name* instead, so persistence is opt-in here.
	 */
	persist?: boolean;
	/** Element that receives the variables (documentElement by default). */
	target?: HTMLElement;
}

/**
 * Apply a theme by writing every prop as a CSS custom property on `<html>`,
 * mirroring Firefish's `applyTheme` in `scripts/theme.ts`.
 */
export function applyTheme(
	theme: Theme,
	options: ApplyThemeOptions = {},
): CompiledTheme {
	const target = options.target ?? document.documentElement;
	const compiled = compileTheme(theme);
	const mode = themeMode(theme);

	for (const [key, value] of Object.entries(compiled)) {
		target.style.setProperty(`--${key}`, value);
	}

	target.dataset.theme = mode;
	// Firefish wrote a misspelled `color-schema` custom property; the real
	// property is `color-scheme`, and it makes native form controls and
	// scrollbars follow the theme.
	target.style.setProperty("color-scheme", mode);
	target.classList.add("_themeChanging_");
	if (transitionTimer !== undefined) window.clearTimeout(transitionTimer);
	transitionTimer = window.setTimeout(() => {
		target.classList.remove("_themeChanging_");
		transitionTimer = undefined;
	}, THEME_TRANSITION_MS);

	const themeColor = compiled.htmlThemeColor;
	if (themeColor != null) {
		for (const tag of document.head.children) {
			if (tag.tagName === "META" && tag.getAttribute("name") === "theme-color") {
				tag.setAttribute("content", themeColor);
				break;
			}
		}
	}

	if (options.persist) {
		try {
			localStorage.setItem("theme", JSON.stringify(compiled));
			localStorage.setItem("colorSchema", mode);
		} catch {
			// Persistence is best-effort; the applied variables still stand.
		}
	}

	return compiled;
}

/** `prefers-color-scheme` probe; false where `matchMedia` is unavailable. */
export function prefersDarkMode(): boolean {
	return (
		typeof window !== "undefined" &&
		typeof window.matchMedia === "function" &&
		window.matchMedia("(prefers-color-scheme: dark)").matches
	);
}
