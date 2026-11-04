import type { UserLite } from "../../api/types";

/**
 * Firefish renders Phosphor icons through `scripts/icon.ts`, keyed off the
 * user's icon-set preference. The port keeps the same class names so markup
 * and selectors are identical; the actual icon font is a follow-up asset.
 */
export function icon(name: string, bold = true): string {
	const prefix = bold ? "ph-bold " : "";
	return `${prefix}ph-${name}`;
}

export function acct(user: Pick<UserLite, "username" | "host">): string {
	return user.host == null ? `@${user.username}` : `@${user.username}@${user.host}`;
}

export function userPage(user: Pick<UserLite, "username" | "host">): string {
	return `/${acct(user)}`;
}

export function notePage(note: { id: string }): string {
	return `/notes/${note.id}`;
}
