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
