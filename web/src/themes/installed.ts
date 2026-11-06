// SPDX-License-Identifier: AGPL-3.0-only
// User-installed themes, ported from the "installed themes" half of Firefish's
// `scripts/theme.ts` (the upstream build keeps them in ColdDeviceStorage).
//
// Cinderfish persists them in localStorage under `cinderfish.installedThemes`
// as a JSON array. A malformed entry is skipped instead of discarding the
// whole list, so one bad install cannot wipe a user's themes.
import { validateTheme } from "./json5";
import type { Theme } from "./types";

export const INSTALLED_THEMES_KEY = "cinderfish.installedThemes";

let cache: Theme[] | null = null;

function storage(): Storage | null {
	if (typeof localStorage === "undefined") return null;
	return localStorage;
}

function load(): Theme[] {
	if (cache != null) return cache;

	const store = storage();
	if (store == null) {
		cache = [];
		return cache;
	}
	const raw = store.getItem(INSTALLED_THEMES_KEY);
	if (raw == null || raw === "") {
		cache = [];
		return cache;
	}
	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch {
		cache = [];
		return cache;
	}
	if (!Array.isArray(parsed)) {
		cache = [];
		return cache;
	}

	const seen = new Set<string>();
	const themes: Theme[] = [];
	for (const entry of parsed) {
		if (!validateTheme(entry)) continue;
		if (seen.has(entry.id)) continue;
		seen.add(entry.id);
		themes.push(entry);
	}
	cache = themes;
	return cache;
}

function save(themes: Theme[]): void {
	cache = themes;
	const store = storage();
	if (store == null) return;
	try {
		store.setItem(INSTALLED_THEMES_KEY, JSON.stringify(themes));
	} catch {
		// Quota/private-mode failures leave the in-memory list usable.
	}
}

/** Installed themes in install order. */
export function listInstalledThemes(): Theme[] {
	return load().slice();
}

export function findInstalledTheme(id: string): Theme | undefined {
	return load().find((theme) => theme.id === id);
}

/**
 * Adds a theme, replacing an installed theme with the same id. Installing over
 * a built-in id is refused: `builtin.ts` resolves built-ins first, so the
 * installed copy would never be selectable.
 */
export function installTheme(theme: Theme): void {
	if (!validateTheme(theme)) {
		throw new Error("installTheme: not a valid theme");
	}
	const themes = load();
	const index = themes.findIndex((entry) => entry.id === theme.id);
	if (index === -1) themes.push(theme);
	else themes[index] = theme;
	save(themes);
}

export function uninstallTheme(id: string): void {
	save(load().filter((theme) => theme.id !== id));
}

/** Drops the in-memory cache; used by tests and the storage event listener. */
export function reloadInstalledThemes(): void {
	cache = null;
}

if (typeof window !== "undefined") {
	window.addEventListener("storage", (event) => {
		if (event.key === INSTALLED_THEMES_KEY) cache = null;
	});
}
