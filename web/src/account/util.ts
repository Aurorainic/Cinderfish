/**
 * Small user-formatting helpers shared by the account components. Firefish
 * keeps these in `filters/user.ts`.
 */
import type { UserLite } from "./types";

/** `@name` for local accounts, `@name@host` for remote ones. */
export function acct(user: Pick<UserLite, "username" | "host">): string {
	return user.host == null || user.host === ""
		? `@${user.username}`
		: `@${user.username}@${user.host}`;
}

export function displayName(user: Pick<UserLite, "name" | "username">): string {
	return user.name != null && user.name !== "" ? user.name : user.username;
}

export function userPagePath(
	user: Pick<UserLite, "username" | "host">,
): string {
	if (user.host != null && user.host !== "") {
		return `/@${user.username}@${user.host}`;
	}
	return `/@${user.username}`;
}

export function isLocalUser(user: Pick<UserLite, "host">): boolean {
	return user.host == null || user.host === "";
}

/** Initial used when a remote avatar cannot be loaded. */
export function avatarInitial(
	user: Pick<UserLite, "name" | "username">,
): string {
	return displayName(user).slice(0, 1).toUpperCase();
}
