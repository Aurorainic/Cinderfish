// SPDX-License-Identifier: AGPL-3.0-only
//
// Applies custom CSS to the document: the per-account sheet from
// `defaultStore.state.customCss`, and the instance-wide sheet an administrator
// saved through `admin/update-meta`. Firefish's settings page wrote a bare
// `customCss` localStorage key and reloaded the page; Cinderfish keeps the
// account value in the store and applies both live through <style> elements.
import { watch } from "vue";
import { instance } from "@/instance";
import { defaultStore } from "@/store";

const STYLE_ID = "cinderfish-custom-css";
const INSTANCE_STYLE_ID = "cinderfish-instance-custom-css";

let initialized = false;

function styleElement(id = STYLE_ID): HTMLStyleElement {
	const existing = document.getElementById(id);
	if (existing instanceof HTMLStyleElement) return existing;

	const style = document.createElement("style");
	style.id = id;
	// Appended last so user rules win the cascade against the bundled sheets
	// without needing `!important`.
	document.head.appendChild(style);
	return style;
}

function apply(css: string, id = STYLE_ID): void {
	const style = styleElement(id);
	style.textContent = css;
}

/**
 * Mounts the custom CSS element and keeps it in sync with the store. Safe to
 * call more than once; only the first call installs the watcher.
 */
export function initCustomCss(): void {
	if (initialized) return;
	initialized = true;

	// Instance first, account second: later <style> elements win at equal
	// specificity, so the per-account sheet must be appended after the
	// instance-wide one.
	apply(instance.customCss ?? "", INSTANCE_STYLE_ID);
	apply(defaultStore.reactiveState.customCss.value);

	watch(defaultStore.reactiveState.customCss, (css) => {
		apply(css);
	});
	watch(
		() => instance.customCss,
		(css) => {
			apply(css ?? "", INSTANCE_STYLE_ID);
		},
	);

	// The instance sheet arrives with `meta`, which main.ts fetches right after
	// this runs; the cached copy is applied above so the first paint is right.
}
