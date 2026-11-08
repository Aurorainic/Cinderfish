// Ported from firefish@b425f61:packages/client/src/filters/user.ts (AGPL-3.0-only)
//
// Firefish called firefish-js/built/acct; the function is four lines, so it is
// inlined here. NOTE: acct() deliberately does NOT include the leading "@"
// (templates write "@{{ acct(user) }}"), which is what firefish-js returns.
import type { UserLite } from "../api/types";
import { url } from "../config";

/** "alice" on this instance, "alice@remote.example" for a remote user. */
export function acct(user: Pick<UserLite, "username" | "host">): string {
	return user.host == null ? user.username : user.username + "@" + user.host;
}

/** Parsed acct: host is null for the local instance. */
export interface ParsedAcct {
	username: string;
	host: string | null;
}

/**
 * firefish-js `acct.parse`: "@user@host" / "user@host" / "user" -> parts.
 * Trailing dots are the FQDN root convention firefish-js strips; the local
 * host compares case-insensitively.
 */
export function parseAcct(value: string): ParsedAcct | null {
	let rest = value.trim();
	if (rest.startsWith("@")) rest = rest.slice(1);

	const at = rest.indexOf("@", 1);
	let username: string;
	let host: string | null;
	if (at === -1) {
		username = rest;
		host = null;
	} else {
		username = rest.slice(0, at);
		host = rest.slice(at + 1).toLowerCase().replace(/\.$/, "");
	}
	if (!/^\w+$/.test(username) || (host !== null && host === "")) return null;
	return { username, host };
}

export function userName(user: Pick<UserLite, "username" | "name">): string {
	return user.name ?? user.username;
}

/** Router path of a user profile: "/@alice", optionally with a sub-path. */
export function userPage(
	user: Pick<UserLite, "username" | "host">,
	path?: string | null,
	absolute = false,
): string {
	const base = (absolute ? url : "") + "/@" + acct(user);
	return path != null && path !== "" ? base + "/" + path : base;
}
