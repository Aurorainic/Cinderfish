// Ported from firefish@b425f61:packages/client/src/config.ts (AGPL-3.0-only)
//
// Static runtime facts. Firefish injected _VERSION_, _LANGS_ and friends at
// build time; Cinderfish has no such build step, so the pieces that can only
// come from the backend (version, instance name) are kept as live module
// bindings that ./instance.ts updates once "meta" has been fetched. Plain
// "let" exports stay readable from templates and from string concatenation
// exactly like Firefish's constants, and ESM live bindings make the update
// visible to every importer.

const address = new URL(
	typeof location !== "undefined" ? location.href : "http://localhost/",
);

function readLocalStorage(key: string): string | null {
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}

interface CachedInstanceInfo {
	name?: unknown;
	version?: unknown;
	uri?: unknown;
}

function readCachedInstance(): CachedInstanceInfo | null {
	// ./instance.ts keeps the last successful "meta" response here so the first
	// paint has a real instance name/version instead of an empty string.
	const raw = readLocalStorage("instance");
	if (raw == null) return null;
	try {
		const parsed = JSON.parse(raw) as unknown;
		return parsed != null && typeof parsed === "object"
			? (parsed as CachedInstanceInfo)
			: null;
	} catch {
		return null;
	}
}

const cached = readCachedInstance();

export const host = address.host;
export const hostname = address.hostname;
export const url = address.origin;
export const apiUrl = url + "/api";
export const wsUrl = url.replace(/^http/, "ws") + "/streaming";
export const lang = readLocalStorage("lang");
export const ui = readLocalStorage("ui");
export const debug = readLocalStorage("debug") === "true";
export const isDevelopment = import.meta.env.DEV;

/** Instance software version (backend meta.version). */
export let version =
	typeof cached?.version === "string" ? cached.version : "";

/** Instance display name (backend meta.name). */
export let instanceName =
	typeof cached?.name === "string" && cached.name !== "" ? cached.name : host;

/** Called by ./instance.ts after "meta" resolves. */
export function applyInstanceInfo(info: {
	name?: string | null;
	version?: string | null;
}): void {
	if (typeof info.version === "string" && info.version !== "") {
		version = info.version;
	}
	if (typeof info.name === "string" && info.name !== "") {
		instanceName = info.name;
	}
}
