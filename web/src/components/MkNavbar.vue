<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/navbar.vue (AGPL-3.0-only) -->
<!-- The drawer variant is the port of firefish@b425f61:packages/client/src/ui/_common_/navbar-for-mobile.vue (AGPL-3.0-only) -->
<template>
	<header class="mvcprjjd sidebar" :class="{ iconOnly, drawer }">
		<div class="body">
			<div class="top">
				<div
					v-if="me != null && me.bannerUrl != null && me.bannerUrl !== ''"
					class="banner"
					:style="{ backgroundImage: `url(${me.bannerUrl})` }"
				></div>
				<button
					v-tooltip="accountLabel"
					class="item _button account"
					@click="openAccountMenu"
				>
					<CfAvatar
						v-if="me != null"
						:user="me"
						class="icon"
						disable-link
					/>
					<span v-else class="icon guest ph-bold ph-user-circle"></span>
				</button>
			</div>

			<nav class="middle">
				<CfA
					to="/"
					v-tooltip="t('timeline')"
					class="item index"
					active-class="active"
				>
					<i class="icon ph-house ph-bold ph-lg ph-fw"></i>
					<span class="text">{{ t("timeline") }}</span>
				</CfA>

				<CfA
					v-for="nav in menu"
					:key="nav.id"
					:to="nav.to"
					v-tooltip="t(nav.title)"
					class="item"
					active-class="active"
				>
					<i class="icon ph-lg ph-fw" :class="nav.icon"></i>
					<span class="text">{{ t(nav.title) }}</span>
					<span v-if="nav.indicated" class="indicator">
						<i class="icon ph-circle ph-fill"></i>
					</span>
				</CfA>

				<div class="divider"></div>

				<CfA
					v-if="moderator"
					to="/admin"
					v-tooltip="t('controlPanel')"
					class="item"
					active-class="active"
				>
					<i class="icon ph-door ph-bold ph-lg ph-fw"></i>
					<span class="text">{{ t("controlPanel") }}</span>
				</CfA>

				<button
					v-tooltip="t('more')"
					class="item _button"
					@click="openMoreMenu"
				>
					<i class="icon ph-dots-three-outline ph-bold ph-lg ph-fw"></i>
					<span class="text">{{ t("more") }}</span>
					<span v-if="overflowIndicated" class="indicator">
						<i class="icon ph-circle ph-fill"></i>
					</span>
				</button>

				<CfA
					to="/settings"
					v-tooltip="t('settings')"
					class="item"
					active-class="active"
				>
					<i class="icon ph-gear-six ph-bold ph-lg ph-fw"></i>
					<span class="text">{{ t("settings") }}</span>
				</CfA>
			</nav>

			<div class="bottom">
				<button
					v-tooltip="t('note')"
					class="item _button post"
					data-cy-open-post-form
					@click="openPost"
				>
					<i class="icon ph-pencil ph-bold ph-lg ph-fw"></i>
					<span class="text">{{ t("note") }}</span>
				</button>
				<button
					v-tooltip="t('help')"
					class="item _button help"
					@click="openHelpMenu"
				>
					<i class="icon ph-info ph-bold ph-xl ph-fw"></i>
				</button>
			</div>
		</div>
	</header>
</template>

<script setup lang="ts">
// Firefish's navbar keeps one item table (`src/navbar.ts`) and renders whichever
// entries the user enabled in `defaultStore.menu`. The port does the same: the
// menu order comes from the store, the entries from the table below. Entries the
// menu leaves out stay reachable through the "more" menu, as MkLaunchPad does
// upstream.
//
// The header banner uses `$i.bannerUrl` like upstream; MkLaunchPad is replaced
// by `os.popupMenu`, and the unread indicators come from the counts on
// `POST /api/i`.
import { computed } from "vue";
import { useRouter } from "vue-router";
import { popupMenu, type MenuItem } from "@/os";
import { post } from "@/os/post";
import { me, signOut } from "@/account";
import { useLocale } from "@/i18n";
import { store, type MenuOption } from "@/store/base";
import { userPage } from "@/components/ui/icon";
import CfA from "@/components/ui/CfA.vue";
import CfAvatar from "@/components/ui/CfAvatar.vue";

withDefaults(
	defineProps<{
		/** Collapse to the 80px icon rail (Firefish's `iconOnly`). */
		iconOnly?: boolean;
		/** Render the mobile drawer body (navbar-for-mobile.vue). */
		drawer?: boolean;
	}>(),
	{
		iconOnly: false,
		drawer: false,
	},
);

interface NavEntry {
	id: string;
	/** Flat i18n key, kept in Firefish's spelling. */
	title: string;
	/** Phosphor classes after the shared `ph-lg ph-fw` prefix. */
	icon: string;
	to: string;
	authRequired?: boolean;
	indicated?: boolean;
}

const NAV_ENTRIES: Record<MenuOption, NavEntry> = {
	notifications: {
		id: "notifications",
		title: "notifications",
		icon: "ph-bell ph-bold",
		to: "/my/notifications",
		authRequired: true,
	},
	followRequests: {
		id: "followRequests",
		title: "followRequests",
		icon: "ph-hand-waving ph-bold",
		to: "/my/follow-requests",
		authRequired: true,
	},
	messaging: {
		id: "messaging",
		title: "messaging",
		icon: "ph-chats-teardrop ph-bold",
		to: "/my/messaging",
		authRequired: true,
	},
	explore: {
		id: "explore",
		title: "explore",
		icon: "ph-compass ph-bold",
		to: "/explore",
	},
	favorites: {
		id: "favorites",
		title: "favorites",
		icon: "ph-bookmark-simple ph-bold",
		to: "/my/favorites",
		authRequired: true,
	},
	channels: {
		id: "channels",
		title: "channel",
		icon: "ph-television ph-bold",
		to: "/channels",
	},
	gallery: {
		id: "gallery",
		title: "gallery",
		icon: "ph-image-square ph-bold",
		to: "/gallery",
	},
	search: {
		id: "search",
		title: "search",
		icon: "ph-magnifying-glass ph-bold",
		to: "/search",
	},
};

/** Firefish's navbarItemDef entries that the default menu leaves out. */
const OVERFLOW_ENTRIES: NavEntry[] = [
	{
		id: "drive",
		title: "drive",
		icon: "ph-cloud ph-bold",
		to: "/my/drive",
		authRequired: true,
	},
	{
		id: "announcements",
		title: "announcements",
		icon: "ph-megaphone-simple ph-bold",
		to: "/announcements",
	},
	{
		id: "gallery",
		title: "gallery",
		icon: "ph-image-square ph-bold",
		to: "/gallery",
	},
	{
		id: "registry",
		title: "registry",
		icon: "ph-database ph-bold",
		to: "/registry",
		authRequired: true,
	},
];

/**
 * First-release scope keeps channels / clips / pages / antennas / lists as
 * placeholder routes (Misskey API coverage register §5). Messaging, channels
 * and gallery are wired now, so their entries are no longer filtered out.
 * Search is listed again now that `notes/search` and `users/search` exist; it
 * opens /search, which carries its own query box.
 * Gallery is wired now and listed as a default menu entry. It also stays in
 * OVERFLOW_ENTRIES so accounts whose stored `menu` predates it can still reach
 * it from the "more" menu; `overflow` drops it when the menu already has it.
 */
const UNAVAILABLE_MENU: ReadonlySet<MenuOption> = new Set<MenuOption>([
	"channels",
]);

const router = useRouter();
const { t } = useLocale();

const moderator = computed(
	() => me.value != null && (me.value.isAdmin || me.value.isModerator),
);

const accountLabel = computed(() =>
	me.value == null
		? t("login")
		: `${t("account")}: @${me.value.username}`,
);

function isIndicated(id: string): boolean {
	const account = me.value;
	if (account == null) return false;
	switch (id) {
		case "notifications":
			return (
				account.unreadNotificationsCount > 0 ||
				account.hasUnreadNotification
			);
		case "followRequests":
			return account.hasPendingReceivedFollowRequest;
		case "messaging":
			return account.hasUnreadMessagingMessage;
		case "announcements":
			return account.hasUnreadAnnouncement;
		default:
			return false;
	}
}

const menu = computed<NavEntry[]>(() =>
	store.state.menu
		.filter((key) => !UNAVAILABLE_MENU.has(key))
		.map((key) => NAV_ENTRIES[key])
		.filter((entry) => entry != null)
		.filter((entry) => entry.authRequired !== true || me.value != null)
		.map((entry) => ({ ...entry, indicated: isIndicated(entry.id) })),
);

const overflow = computed<NavEntry[]>(() => {
	const inMenu = new Set(menu.value.map((entry) => entry.id));
	return OVERFLOW_ENTRIES.filter(
		(entry) =>
			!inMenu.has(entry.id) &&
			(entry.authRequired !== true || me.value != null),
	);
});

const overflowIndicated = computed(() =>
	overflow.value.some((entry) => isIndicated(entry.id)),
);

function openAccountMenu(ev: MouseEvent): void {
	const account = me.value;
	if (account == null) {
		void router.push("/");
		return;
	}
	const items: MenuItem[] = [
		{
			text: t("profile"),
			icon: "ph-bold ph-user",
			action: () => {
				void router.push(userPage(account));
			},
		},
		{
			text: t("settings"),
			icon: "ph-bold ph-gear",
			action: () => {
				void router.push("/settings");
			},
		},
		{ separator: true },
		{
			text: t("logout"),
			icon: "ph-bold ph-sign-out",
			action: () => {
				void signOut();
			},
		},
	];
	void popupMenu(items, ev);
}

function openMoreMenu(ev: MouseEvent): void {
	const items: MenuItem[] = overflow.value.map((entry) => ({
		text: t(entry.title),
		icon: entry.icon,
		indicate: isIndicated(entry.id),
		action: () => {
			void router.push(entry.to);
		},
	}));
	if (items.length === 0) return;
	void popupMenu(items, ev);
}

function openHelpMenu(ev: MouseEvent): void {
	void popupMenu(
		[
			{
				text: t("about"),
				icon: "ph-bold ph-info",
				action: () => {
					void router.push("/about");
				},
			},
			{
				text: t("_mfm.cheatSheet"),
				icon: "ph-bold ph-code",
				action: () => {
					void router.push("/mfm-cheat-sheet");
				},
			},
			{
				text: t("scratchpad"),
				icon: "ph-bold ph-terminal",
				action: () => {
					void router.push("/scratchpad");
				},
			},
		],
		ev,
	);
}

function openPost(): void {
	void post();
}
</script>

<style lang="scss" scoped>
.mvcprjjd {
	--nav-width: 250px;
	--nav-icon-only-width: 80px;

	flex: 0 0 var(--nav-width);
	width: var(--nav-width);
	box-sizing: border-box;

	> .body {
		position: sticky;
		inset-block-start: 0;
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100vh;
		box-sizing: border-box;
		overflow-y: auto;
		overflow-x: clip;
		background: var(--navBg);
		border-inline-end: solid 0.5px var(--divider);
	}

	.item {
		outline: none;

		&:focus-visible {
			outline: auto;
		}
	}

	> .body > .top {
		position: relative;
		z-index: 1;
		padding: 2rem 0;

		> .banner {
			position: absolute;
			inset: 0;
			background-size: cover;
			background-position: center;
			mask-image: linear-gradient(to bottom, #000 0%, transparent 100%);
			-webkit-mask-image: linear-gradient(to bottom, #000 0%, transparent 100%);
			pointer-events: none;
		}

		> .account {
			position: relative;
			display: block;
			width: 100%;
			text-align: center;

			> .icon {
				--avatarSize: 55px;
				display: inline-block;
				width: 55px;
				height: 55px;
			}

			> .guest {
				font-size: 55px;
				color: var(--fgTransparentWeak);
				line-height: 1;
			}
		}
	}

	> .body > .middle {
		flex: 0.1;

		> .divider {
			margin: 16px;
			border-top: solid 0.5px var(--divider);
		}

		> .item {
			position: relative;
			display: flex;
			align-items: center;
			width: calc(100% - 16px);
			margin-inline: 8px;
			margin-bottom: 0.5rem;
			padding-inline-start: 22px;
			box-sizing: border-box;
			line-height: 2.85rem;
			white-space: nowrap;
			text-align: left;
			color: var(--navFg);
			border-radius: 999px;
			transition:
				background 0.4s ease,
				color 0.4s ease;

			> .icon {
				position: relative;
				width: 32px;
				margin-inline-end: 8px;
			}

			> .text {
				position: relative;
				font-size: 0.9em;
				overflow: hidden;
				text-overflow: ellipsis;
			}

			> .indicator {
				position: absolute;
				inset-block-start: 0;
				inset-inline-start: 12px;
				color: var(--navIndicator);
				font-size: 8px;
			}

			&:hover,
			&:focus-within {
				color: var(--navHoverFg);
				background: var(--buttonHoverBg);
				text-decoration: none;
			}

			&.active {
				color: var(--accent);
				background: var(--accentedBg);
			}
		}
	}

	> .body > .bottom {
		padding: 20px 0;

		> .post {
			position: relative;
			display: flex;
			align-items: center;
			width: calc(100% - 16px);
			margin-inline: 8px;
			height: 40px;
			color: var(--fgOnAccent);
			font-weight: bold;
			text-align: left;
			border-radius: 999px;
			background: linear-gradient(
				90deg,
				var(--buttonGradateA),
				var(--buttonGradateB)
			);
			transition: background 0.4s ease;

			&:hover,
			&:focus-within {
				background: var(--accentLighten);
			}

			> .icon {
				margin-inline-start: 18px;
			}

			> .text {
				margin-inline-start: 1rem;
			}
		}

		> .help {
			display: block;
			width: 100%;
			margin-block-start: 1rem;
			text-align: center;
			color: var(--navFg);

			&:hover {
				color: var(--navHoverFg);
			}
		}
	}

	// ---- icon rail ---------------------------------------------------------
	&.iconOnly {
		flex: 0 0 var(--nav-icon-only-width);
		width: var(--nav-icon-only-width);

		> .body > .top > .account > .icon {
			--avatarSize: 40px;
			width: 40px;
			height: 40px;
		}

		> .body > .middle {
			> .divider {
				margin: 8px auto;
				width: calc(100% - 32px);
			}

			> .item {
				display: flex;
				justify-content: center;
				width: calc(100% - 16px);
				padding-inline-start: 0;
				padding-block: 1.1rem;
				margin-bottom: 0.2rem;
				line-height: 1;

				> .icon {
					display: block;
					margin: 0 auto;
					opacity: 0.7;
				}

				> .text {
					display: none;
				}

				> .indicator {
					inset-block-start: 6px;
					inset-inline-start: 22px;
				}

				&:hover,
				&:focus-within,
				&.active {
					color: var(--accent);

					> .icon {
						opacity: 1;
					}
				}
			}
		}

		> .body > .bottom > .post {
			width: 52px;
			height: 52px;
			margin-inline: auto;
			margin-block-end: 16px;
			justify-content: center;
			border-radius: 100%;

			> .icon {
				margin-inline-start: 0;
			}

			> .text {
				display: none;
			}
		}

		> .body > .bottom > .help {
			display: block;
			width: 100%;
			text-align: center;
		}
	}

	// ---- drawer body (navbar-for-mobile.vue) ------------------------------
	&.drawer {
		flex: 1 1 auto;
		width: 100%;

		> .body {
			position: static;
			height: 100%;
			border-inline-end: none;
			background: var(--navBg);
		}

		> .body > .middle > .item {
			line-height: 2.85rem;
			margin-bottom: 0;
		}
	}
}
</style>
