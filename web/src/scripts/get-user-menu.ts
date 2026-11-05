// Ported from firefish@b425f61:packages/client/src/scripts/get-user-menu.ts (AGPL-3.0-only)
//
// Degradations versus Firefish, all driven by missing backend endpoints
// (docs/research/api-contract/): lists (`users/lists/*`), groups
// (`users/groups/*`), mute (`mute/*`), renote-mute, block (`blocking/*`),
// silence (`admin/silence-user`), following/invalidate, report abuse,
// RSS/Atom/JSON feed notifications and `userActions` plugin entries are all
// dropped. "Send message" is dropped too: Firefish opens the post form with
// `{ specified: user }`, and the Cinderfish `os.post` contract has no
// `specified` / `visibleUserIds` field yet.
//
// Two call shapes are supported so other waves can adopt it without guessing:
//   * `getUserMenu(user)` returns the items (for tests or custom popup);
//   * `getUserMenu(user, ev)` also pops the menu at the event, which is what
//     every ported call site does.
import type { UserDetailed } from "@/api/types";
import { me } from "@/account";
import { host as instanceHost } from "@/config";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { router } from "@/router";
import copyToClipboard from "@/scripts/copy-to-clipboard";

function confirmed(text: string): Promise<boolean> {
	return os
		.confirm({ type: "warning", text })
		.then(({ canceled }) => !canceled);
}

export function getUserMenu(
	user: UserDetailed,
	ev?: MouseEvent | { x: number; y: number },
): MenuItem[] {
	const current = me.value;
	const isSelf = current != null && current.id === user.id;

	function toggleSuspend(): void {
		if (current == null || !current.isAdmin) return;
		void confirmed(t("suspendConfirm")).then((ok) => {
			if (!ok) return;
			void os.apiWithDialog(
				user.isSuspended ? "admin/unsuspend-user" : "admin/suspend-user",
				{ userId: user.id },
			);
		});
	}

	const items: MenuItem[] = [
		{
			type: "label",
			text: user.host == null
				? `@${user.username}`
				: `@${user.username}@${user.host}`,
		},
		{
			icon: "ph-at ph-bold ph-lg",
			text: t("copyUsername"),
			action: () => {
				void copyToClipboard(
					user.host == null
						? `@${user.username}@${instanceHost}`
						: `@${user.username}@${user.host}`,
				);
			},
		},
		{
			icon: "ph-info ph-bold ph-lg",
			text: t("info"),
			action: () => {
				void router.push(`/user-info/${user.id}`);
			},
		},
	];

	// `url`/`uri` are always empty strings on Cinderfish, so this entry stays
	// hidden until the backend exposes the remote actor URL.
	if (user.host != null && user.uri !== "") {
		items.push({
			type: "a",
			icon: "ph-arrow-square-out ph-bold ph-lg",
			text: t("showOnRemote"),
			href: user.url !== "" ? user.url : user.uri,
			target: "_blank",
		});
	}

	if (isSelf) {
		items.push(
			{ separator: true },
			{
				icon: "ph-pencil ph-bold ph-lg",
				text: t("editProfile"),
				action: () => {
					void router.push("/settings/profile");
				},
			},
		);
	} else if (current?.isAdmin === true) {
		items.push(
			{ separator: true },
			{
				icon: "ph-snowflake ph-bold ph-lg",
				text: user.isSuspended ? t("unsuspend") : t("suspend"),
				danger: !user.isSuspended,
				action: toggleSuspend,
			},
		);
	}

	if (ev != null) void os.popupMenu(items, ev);
	return items;
}
