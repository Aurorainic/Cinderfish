// SPDX-License-Identifier: AGPL-3.0-only
// User-installed themes, ported from firefish@b425f61:packages/client/src/theme-store.ts
// (AGPL-3.0-only).
//
// Firefish keeps the list in the server registry (scope ["client"], key
// "themes") and caches it in localStorage under `themes:<userId>`. The cache
// stays synchronous so the first paint can resolve a selected theme, and the
// settings pages call `fetchInstalledThemes()` to pick up a list installed on
// another device. A malformed entry is skipped instead of discarding the whole
// list, so one bad install cannot wipe a user's themes.
import { readStoredAccount } from "../account/storage";
import { api } from "../api/client";
import { validateTheme } from "./json5";
import type { Theme } from "./types";

/** Base name of the localStorage cache; the account id is appended. */
export const INSTALLED_THEMES_KEY = "cinderfish.installedThemes";

const REGISTRY_KEY = "themes";

let cache: { account: string | null; themes: Theme[] } | null = null;

function accountId(): string | null {
	return readStoredAccount()?.id ?? null;
}

function storage(): Storage | null {
	if (typeof localStorage === "undefined") return null;
	return localStorage;
}

/** localStorage key; Firefish scopes it per account (`themes:<id>`). */
export function installedThemesKey(): string {
	const id = accountId();
	return id == null ? INSTALLED_THEMES_KEY : `${INSTALLED_THEMES_KEY}:${id}`;
}

function sanitize(value: unknown): Theme[] {
	if (!Array.isArray(value)) return [];
	const seen = new Set<string>();
	const themes: Theme[] = [];
	for (const entry of value) {
		if (!validateTheme(entry)) continue;
		if (seen.has(entry.id)) continue;
		seen.add(entry.id);
		themes.push(entry);
	}
	return themes;
}

function load(): Theme[] {
	const account = accountId();
	if (cache != null && cache.account === account) return cache.themes;

	const store = storage();
	if (store == null) {
		cache = { account, themes: [] };
		return cache.themes;
	}
	const raw = store.getItem(installedThemesKey());
	if (raw == null || raw === "") {
		cache = { account, themes: [] };
		return cache.themes;
	}
	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch {
		cache = { account, themes: [] };
		return cache.themes;
	}
	cache = { account, themes: sanitize(parsed) };
	return cache.themes;
}

function save(themes: Theme[]): void {
	cache = { account: accountId(), themes };
	const store = storage();
	if (store == null) return;
	try {
		store.setItem(installedThemesKey(), JSON.stringify(themes));
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
 * Pulls the server list into the local cache. Without a signed-in account the
 * registry is not addressable, so the local list is returned unchanged.
 *
 * A missing registry entry is not an error: the local cache may hold a theme
 * installed offline, and Firefish leaves the cache alone in the same case.
 */
export async function fetchInstalledThemes(): Promise<Theme[]> {
	if (accountId() == null) return listInstalledThemes();
	try {
		const value = await api<unknown>("i/registry/get", {
			scope: ["client"],
			key: REGISTRY_KEY,
		});
		save(sanitize(value));
	} catch (error) {
		const code =
			error != null && typeof error === "object" && "code" in error
				? (error as { code?: unknown }).code
				: null;
		if (code !== "NO_SUCH_KEY") throw error;
	}
	return listInstalledThemes();
}

async function writeRegistry(themes: Theme[]): Promise<void> {
	if (accountId() == null) return;
	await api("i/registry/set", {
		scope: ["client"],
		key: REGISTRY_KEY,
		value: themes,
	});
}

/**
 * Adds a theme, replacing an installed theme with the same id. Installing over
 * a built-in id is refused: `builtin.ts` resolves built-ins first, so the
 * installed copy would never be selectable.
 *
 * The registry write comes first: if it fails the local cache is untouched and
 * the caller can report the failure, matching Firefish's `addTheme`.
 */
export async function installTheme(theme: Theme): Promise<void> {
	if (!validateTheme(theme)) {
		throw new Error("installTheme: not a valid theme");
	}
	await fetchInstalledThemes();
	const themes = load().slice();
	const index = themes.findIndex((entry) => entry.id === theme.id);
	if (index === -1) themes.push(theme);
	else themes[index] = theme;
	await writeRegistry(themes);
	save(themes);
}

export async function uninstallTheme(id: string): Promise<void> {
	const themes = load().filter((theme) => theme.id !== id);
	await writeRegistry(themes);
	save(themes);
}

/** Drops the in-memory cache; used by tests and the storage event listener. */
export function reloadInstalledThemes(): void {
	cache = null;
}

if (typeof window !== "undefined") {
	window.addEventListener("storage", (event) => {
		if (event.key === installedThemesKey()) cache = null;
	});
}
