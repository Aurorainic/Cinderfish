// Ported from firefish@b425f61:packages/client/src/scripts/get-user-menu.ts (AGPL-3.0-only)
//
// Degradations versus Firefish, all driven by missing backend endpoints
// (docs/research/api-contract/): lists (`users/lists/*`), groups
// (`users/groups/*`), renote-mute, silence (`admin/silence-user`),
// RSS/Atom/JSON feed notifications and `userActions` plugin entries are all
// dropped. Mute/block and abuse reporting have since
// been ported and are offered again. "Send message" is dropped too: Firefish opens the post form with
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

async function reportAbuse(user: UserDetailed): Promise<void> {
	const handle = user.host == null ? `@${user.username}` : `@${user.username}@${user.host}`;
	const { canceled, result } = await os.inputParagraph({
		title: t("reportAbuse"),
		text: handle,
		placeholder: "请说明发生了什么（对方的行为、时间或帖子链接）",
	});
	const comment = String(result ?? "").trim();
	if (canceled || comment === "") return;
	try {
		await os.apiWithDialog("users/report-abuse", { userId: user.id, comment });
		os.toast(t("abuseReported"), "success");
	} catch {
		// apiWithDialog already showed the server error.
	}
}

async function invalidateFollow(user: UserDetailed): Promise<void> {
	if (!(await confirmed(t("breakFollowConfirm")))) return;
	try {
		await os.apiWithDialog("following/invalidate", { userId: user.id });
		user.isFollowed = false;
	} catch {
		// apiWithDialog already showed the server error.
	}
}

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
	} else if (current != null && current.isAdmin !== true) {
		items.push(
			{ separator: true },
			{
				icon: "ph-warning-octagon ph-bold ph-lg",
				text: t("reportAbuse"),
				danger: true,
				action: () => {
					void reportAbuse(user);
				},
			},
		);
	}

	// A follower can be removed without their consent; Firefish offers this on
	// the profile of someone who follows you.
	if (!isSelf && user.isFollowed) {
		items.push(
			{ separator: true },
			{
				icon: "ph-user-minus ph-bold ph-lg",
				text: t("breakFollow"),
				danger: true,
				action: () => {
					void invalidateFollow(user);
				},
			},
		);
	}

	if (current?.isAdmin === true && !isSelf) {
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
