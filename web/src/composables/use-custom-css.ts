// SPDX-License-Identifier: AGPL-3.0-only
//
// Applies the per-account custom CSS (`defaultStore.state.customCss`) to the
// document. Firefish's settings page wrote a bare `customCss` localStorage key
// and reloaded the page; Cinderfish keeps the value in the store (so it syncs
// with the account) and applies it live through a single <style> element.
import { watch } from "vue";
import { defaultStore } from "@/store";

const STYLE_ID = "cinderfish-custom-css";

let initialized = false;

function styleElement(): HTMLStyleElement {
	const existing = document.getElementById(STYLE_ID);
	if (existing instanceof HTMLStyleElement) return existing;

	const style = document.createElement("style");
	style.id = STYLE_ID;
	// Appended last so user rules win the cascade against the bundled sheets
	// without needing `!important`.
	document.head.appendChild(style);
	return style;
}

function apply(css: string): void {
	const style = styleElement();
	style.textContent = css;
}

/**
 * Mounts the custom CSS element and keeps it in sync with the store. Safe to
 * call more than once; only the first call installs the watcher.
 */
export function initCustomCss(): void {
	if (initialized) return;
	initialized = true;

	apply(defaultStore.reactiveState.customCss.value);

	watch(defaultStore.reactiveState.customCss, (css) => {
		apply(css);
	});
}
