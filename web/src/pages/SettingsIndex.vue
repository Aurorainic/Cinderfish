<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/index.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="900" :margin-min="20" :margin-max="32">
		<div ref="el" class="vvcocwet" :class="{ wide: !narrow }">
			<div class="body">
				<div v-if="!narrow || !hasChild" class="nav">
					<MkSuperMenu :def="menuDef" :grid="narrow" />
					<CfInfo class="missing">
						菜单只列出已有页面与端点的设置。邮箱、插件、导入、迁移等仍缺后端端点，未列入。
					</CfInfo>
				</div>
				<section v-if="!(narrow && !hasChild)" class="main">
					<div class="child">
						<RouterView />
					</div>
				</section>
			</div>
		</div>
	</MkSpacer>
</template>

<script setup lang="ts">
// Firefish's settings shell: a sticky vertical menu beside the child route on
// wide containers, menu-only (or child-only) on narrow ones, and an automatic
// redirect to /settings/profile when the bare /settings route is opened wide.
//
// Differences from upstream, all forced by what this wave has:
//   * the menu lists only the pages that exist as real ports (12 of the ~30
//     children); the rest of the router table is still `TODO: port` and linking
//     to it would be a dead end;
//   * no accounts block, no `emailNotConfiguredWarning` banner (the email page it
//     links to is not ported), no clear-cache / logout buttons (they need
//     `unisonReload`, which does not exist here);
//   * no MkStickyContainer/MkPageHeader: upstream hides the header on this page
//     anyway (`hideHeader: true`), and the shell renders the title.
import {
	computed,
	onActivated,
	onBeforeUnmount,
	onMounted,
	ref,
	watch,
} from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import MkSuperMenu, { type MkSuperMenuGroup } from "@/components/MkSuperMenu.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

const NARROW_THRESHOLD = 600;

const route = useRoute();
const router = useRouter();

const el = ref<HTMLElement | null>(null);
const narrow = ref(false);
let observer: ResizeObserver | null = null;

// The router table names the empty child "settings-index", so "is a child page
// open?" is a name test rather than the `matched.length` Firefish's nirax gives.
const hasChild = computed(
	() => route.name !== "settings-index" && route.path !== "/settings",
);

function isActive(path: string): boolean {
	return route.path === path || route.path.startsWith(`${path}/`);
}

const menuDef = computed<MkSuperMenuGroup[]>(() => [
	{
		title: t("basicSettings"),
		items: [
			{
				icon: "ph-user ph-bold ph-lg",
				text: t("profile"),
				to: "/settings/profile",
				active: isActive("/settings/profile"),
			},
			{
				icon: "ph-keyhole ph-bold ph-lg",
				text: t("privacy"),
				to: "/settings/privacy",
				active: isActive("/settings/privacy"),
			},
			{
				icon: "ph-smiley ph-bold ph-lg",
				text: t("reaction"),
				to: "/settings/reaction",
				active: isActive("/settings/reaction"),
			},
			{
				icon: "ph-cloud ph-bold ph-lg",
				text: t("drive"),
				to: "/settings/drive",
				active: isActive("/settings/drive"),
			},
			{
				icon: "ph-bell ph-bold ph-lg",
				text: t("notifications"),
				to: "/settings/notifications",
				active: isActive("/settings/notifications"),
			},
			{
				icon: "ph-lock ph-bold ph-lg",
				text: t("security"),
				to: "/settings/security",
				active: isActive("/settings/security"),
			},
			{
				icon: "ph-mask-happy ph-bold ph-lg",
				text: t("accessibility"),
				to: "/settings/accessibility",
				active: isActive("/settings/accessibility"),
			},
			{
				icon: "ph-user-circle-minus ph-bold ph-lg",
				text: t("muteAndBlock"),
				to: "/settings/mute-block",
				active: isActive("/settings/mute-block"),
			},
			{
				icon: "ph-server ph-bold ph-lg",
				text: t("instanceMute"),
				to: "/settings/instance-mute",
				active: isActive("/settings/instance-mute"),
			},
		],
	},
	{
		title: t("clientSettings"),
		items: [
			{
				icon: "ph-gear-six ph-bold ph-lg",
				text: t("general"),
				to: "/settings/general",
				active: isActive("/settings/general"),
			},
			{
				icon: "ph-palette ph-bold ph-lg",
				text: t("theme"),
				to: "/settings/theme",
				active: isActive("/settings/theme"),
			},
			{
				icon: "ph-brackets-curly ph-bold ph-lg",
				text: t("customCss"),
				to: "/settings/custom-css",
				active: isActive("/settings/custom-css"),
			},
			{
				icon: "ph-speaker-slash ph-bold ph-lg",
				text: t("wordMute"),
				to: "/settings/word-mute",
				active: isActive("/settings/word-mute"),
			},
			{
				icon: "ph-list ph-bold ph-lg",
				text: t("navbar"),
				to: "/settings/navbar",
				active: isActive("/settings/navbar"),
			},
			{
				icon: "ph-speaker-high ph-bold ph-lg",
				text: t("sounds"),
				to: "/settings/sounds",
				active: isActive("/settings/sounds"),
			},
			{
				icon: "ph-columns ph-bold ph-lg",
				text: t("deck"),
				to: "/settings/deck",
				active: isActive("/settings/deck"),
			},
			{
				icon: "ph-archive ph-bold ph-lg",
				text: t("preferencesBackups"),
				to: "/settings/preferences-backups",
				active: isActive("/settings/preferences-backups"),
			},
			{
				icon: "ph-terminal-window ph-bold ph-lg",
				text: "API",
				to: "/settings/api",
				active: isActive("/settings/api"),
			},
		],
	},
	{
		title: t("otherSettings"),
		items: [
			{
				icon: "ph-identification-card ph-bold ph-lg",
				text: t("accountInfo"),
				to: "/settings/account-info",
				active: isActive("/settings/account-info"),
			},
			{
				icon: "ph-database ph-bold ph-lg",
				text: t("registry"),
				to: "/registry",
				active: isActive("/registry"),
			},
			{
				icon: "ph-webhooks-logo ph-bold ph-lg",
				text: "Webhook",
				to: "/settings/webhook",
				active: isActive("/settings/webhook"),
			},
			{
				icon: "ph-arrows-down-up ph-bold ph-lg",
				text: t("importAndExport"),
				to: "/settings/import-export",
				active: isActive("/settings/import-export"),
			},
			{
				icon: "ph-devices ph-bold ph-lg",
				text: t("accounts"),
				to: "/settings/accounts",
				active: isActive("/settings/accounts"),
			},
			{
				icon: "ph-app-window ph-bold ph-lg",
				text: t("apps"),
				to: "/settings/apps",
				active: isActive("/settings/apps"),
			},
			{
				icon: "ph-trash ph-bold ph-lg",
				text: t("deleteAccount"),
				to: "/settings/delete-account",
				active: isActive("/settings/delete-account"),
			},
			{
				icon: "ph-dots-three-outline ph-bold ph-lg",
				text: t("other"),
				to: "/settings/other",
				active: isActive("/settings/other"),
			},
		],
	},
]);

function measure(): void {
	const target = el.value;
	if (target == null) return;
	narrow.value = target.offsetWidth < NARROW_THRESHOLD;
}

function redirectIfNeeded(): void {
	if (narrow.value || hasChild.value) return;
	void router.replace("/settings/profile");
}

onMounted(() => {
	measure();

	if (typeof ResizeObserver !== "undefined" && el.value != null) {
		observer = new ResizeObserver((entries) => {
			const entry = entries[0];
			if (entry == null) return;
			const size =
				entry.borderBoxSize?.[0]?.inlineSize ?? entry.contentRect.width;
			narrow.value = size < NARROW_THRESHOLD;
		});
		observer.observe(el.value);
	}

	redirectIfNeeded();
});

onActivated(() => {
	measure();
	redirectIfNeeded();
});

onBeforeUnmount(() => {
	observer?.disconnect();
	observer = null;
});

watch(
	() => route.path,
	() => {
		redirectIfNeeded();
	},
);

definePageMetadata({
	title: t("settings"),
	icon: "ph-gear-six ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.vvcocwet {
	> .body {
		> .nav {
			> .missing {
				margin-top: 16px;
				font-size: 0.85em;
			}
		}
	}

	&.wide {
		> .body {
			display: flex;
			align-items: flex-start;
			height: 100%;

			> .nav {
				width: 34%;
				padding-inline-end: 32px;
				box-sizing: border-box;
			}

			> .main {
				flex: 1;
				min-width: 0;
			}
		}
	}
}
</style>
