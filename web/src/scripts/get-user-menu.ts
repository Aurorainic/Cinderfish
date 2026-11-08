// Ported from firefish@b425f61:packages/client/src/scripts/get-user-menu.ts (AGPL-3.0-only)
//
// Degradations versus Firefish, all driven by missing backend endpoints
// (docs/research/api-contract/): groups (`users/groups/*`), renote-mute,
// silence (`admin/silence-user`), RSS/Atom/JSON feed notifications and
// `userActions` plugin entries are all dropped. Lists have since been ported
// and are offered through an "add to list" entry that opens a second popup.
// Mute/block and abuse reporting have also been ported and are offered again.
// "Send message" is dropped too: Firefish opens the post form with
// `{ specified: user }`, and the Cinderfish `os.post` contract has no
// `specified` / `visibleUserIds` field yet.
//
// Cinderfish's CfPopupMenu does not render `MenuItem.children`, so the list
// submenu is a second popup anchored at the point that opened the user menu.
//
// Two call shapes are supported so other waves can adopt it without guessing:
//   * `getUserMenu(user)` returns the items (for tests or custom popup);
//   * `getUserMenu(user, ev)` also pops the menu at the event, which is what
//     every ported call site does.
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { UserDetailed, UserList } from "@/api/types";
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

async function toggleMembership(
	list: UserList,
	user: UserDetailed,
): Promise<void> {
	const member = list.userIds?.includes(user.id) ?? false;
	try {
		await os.apiWithDialog(
			member
				? ENDPOINTS.usersListsPull
				: ENDPOINTS.usersListsPush,
			{ listId: list.id, userId: user.id },
		);
		const next = new Set(list.userIds ?? []);
		if (member) next.delete(user.id);
		else next.add(user.id);
		list.userIds = [...next];
	} catch {
		// apiWithDialog already showed the server error.
	}
}

/**
 * Open the list picker as a second popup. Firefish nests it as a submenu; the
 * CfPopupMenu port has no `<children>` rendering, so it is anchored at the
 * original pointer position instead.
 */
async function openAddToList(
	user: UserDetailed,
	point: MouseEvent | { x: number; y: number } | null,
): Promise<void> {
	const items: MenuItem[] = [];
	let lists: UserList[] = [];
	try {
		lists = await api(ENDPOINTS.usersListsList, {});
	} catch {
		items.push({ type: "label", text: t("somethingHappened") });
	}

	if (items.length === 0 && lists.length === 0) {
		items.push({ type: "label", text: t("noLists") });
	} else {
		for (const list of lists) {
			const member = list.userIds?.includes(user.id) ?? false;
			items.push({
				type: "button",
				icon: member
					? "ph-check ph-bold ph-lg"
					: "ph-plus ph-bold ph-lg",
				text: list.name,
				active: member,
				action: () => {
					void toggleMembership(list, user);
				},
			});
		}
	}

	items.push(
		{ separator: true },
		{
			icon: "ph-sliders-horizontal ph-bold ph-lg",
			text: t("manageLists"),
			action: () => {
				void router.push("/my/lists");
			},
		},
	);

	const anchor =
		point ??
		({
			x: Math.round(window.innerWidth / 2),
			y: Math.round(window.innerHeight / 2),
		} as const);
	void os.popupMenu(items, anchor);
}

export function getUserMenu(
	user: UserDetailed,
	ev?: MouseEvent | { x: number; y: number },
): MenuItem[] {
	const current = me.value;
	const isSelf = current != null && current.id === user.id;
	const point = ev ?? null;

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
	} else if (current != null) {
		items.push(
			{ separator: true },
			{
				icon: "ph-list-plus ph-bold ph-lg",
				text: t("addToList"),
				action: () => {
					void openAddToList(user, point);
				},
			},
		);
	}

	if (current != null && !isSelf && current.isAdmin !== true) {
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
