<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/index.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer v-if="!admin" :content-max="600" :margin-min="20" :margin-max="32">
		<CfInfo warn class="denied">
			需要管理员权限，无法打开管理台。
			<CfA to="/" class="_link">返回首页</CfA>
		</CfInfo>
	</MkSpacer>
	<div v-else ref="el" class="hiyeyicy" :class="{ wide: !narrow }">
		<div v-if="!narrow || !hasChild" class="nav">
			<MkSuperMenu :def="menuDef" :grid="narrow" />
		</div>
		<section v-if="!(narrow && !hasChild)" class="main">
			<RouterView />
		</section>
	</div>
</template>

<script setup lang="ts">
// Firefish's administration shell. This port keeps the responsive split (menu
// column beside the child route on wide containers, menu-only or child-only on
// narrow ones) and redirects the bare /admin route to the user list, as the
// wave-4 contract requires. Upstream only redirects on wide containers; here
// the redirect is unconditional so /admin never renders the empty child.
//
// Differences from upstream, all forced by what the backend serves:
//   * the menu lists only ported pages; the rest of the router table is still
//     `TODO: port`, and `admin/abuse-user-reports` / `latest-version` do not
//     exist, so the warnings banner and the quick-action block are gone rather
//     than fake;
//   * the instance banner image is dropped (`$instance.iconUrl` is not
//     mirrored by `@/instance`), and no abuse/report count is shown.
import {
	computed,
	onActivated,
	onBeforeUnmount,
	onMounted,
	ref,
	watch,
} from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import { isAdmin } from "@/account";
import MkSuperMenu, { type MkSuperMenuGroup } from "@/components/MkSuperMenu.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

const NARROW_THRESHOLD = 600;

const route = useRoute();
const router = useRouter();

const admin = computed<boolean>(() => isAdmin());
const el = ref<HTMLElement | null>(null);
const narrow = ref(false);
let observer: ResizeObserver | null = null;

// The router table names the empty child "admin-index", so "is a child page
// open?" is a name test rather than Firefish's `currentPage.route.name == null`.
const hasChild = computed<boolean>(
	() =>
		route.name !== "admin-index" &&
		route.path !== "/admin" &&
		route.path !== "/admin/",
);

function isActive(path: string): boolean {
	return route.path === path || route.path.startsWith(`${path}/`);
}

const menuDef = computed<MkSuperMenuGroup[]>(() => [
	{
		title: "管理",
		items: [
			{
				icon: "ph-users ph-bold ph-lg",
				text: t("users"),
				to: "/admin/users",
				active: isActive("/admin/users"),
			},
			{
				icon: "ph-sliders-horizontal ph-bold ph-lg",
				text: "实例设置",
				to: "/admin/settings",
				active: isActive("/admin/settings"),
			},
			{
				icon: "ph-gauge ph-bold ph-lg",
				text: t("overview"),
				to: "/admin/overview",
				active: isActive("/admin/overview"),
			},
		],
	},
	{
		title: "内容",
		items: [
			{
				icon: "ph-warning-octagon ph-bold ph-lg",
				text: t("abuseReports"),
				to: "/admin/abuses",
				active: isActive("/admin/abuses"),
			},
			{
				icon: "ph-megaphone-simple ph-bold ph-lg",
				text: t("announcements"),
				to: "/admin/announcements",
				active: isActive("/admin/announcements"),
			},
			{
				icon: "ph-smiley ph-bold ph-lg",
				text: t("customEmojis"),
				to: "/admin/emojis",
				active: isActive("/admin/emojis"),
			},
			{
				icon: "ph-hash ph-bold ph-lg",
				text: t("hiddenTags"),
				to: "/admin/hashtags",
				active: isActive("/admin/hashtags"),
			},
			{
				icon: "ph-tray ph-bold ph-lg",
				text: "投递队列",
				to: "/admin/queue",
				active: isActive("/admin/queue"),
			},
			{
				icon: "ph-database ph-bold ph-lg",
				text: t("database"),
				to: "/admin/database",
				active: isActive("/admin/database"),
			},
			{
				icon: "ph-cloud-arrow-up ph-bold ph-lg",
				text: "对象存储",
				to: "/admin/object-storage",
				active: isActive("/admin/object-storage"),
			},
			{
				icon: "ph-folder-open ph-bold ph-lg",
				text: t("files"),
				to: "/admin/files",
				active: isActive("/admin/files"),
			},
			{
				icon: "ph-prohibit ph-bold ph-lg",
				text: "实例屏蔽",
				to: "/admin/instance-block",
				active: isActive("/admin/instance-block"),
			},
			{
				icon: "ph-planet ph-bold ph-lg",
				text: t("federation"),
				to: "/admin/federation",
				active: isActive("/admin/federation"),
			},
			{
				icon: "ph-envelope-simple ph-bold ph-lg",
				text: t("emailServer"),
				to: "/admin/email-settings",
				active: isActive("/admin/email-settings"),
			},
			{
				icon: "ph-code ph-bold ph-lg",
				text: "实例 CSS",
				to: "/admin/custom-css",
				active: isActive("/admin/custom-css"),
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
	if (!admin.value || hasChild.value) return;
	void router.replace("/admin/users");
}

onMounted(() => {
	if (!admin.value) return;

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
	title: t("controlPanel"),
	icon: "ph-gear-six ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.denied {
	flex-wrap: wrap;

	> a {
		margin-inline-start: 4px;
	}
}

.hiyeyicy {
	&.wide {
		display: flex;
		align-items: flex-start;
		height: 100%;

		> .nav {
			width: 32%;
			max-width: 280px;
			height: 100%;
			padding: 16px 16px 24px;
			box-sizing: border-box;
			border-inline-end: solid 0.5px var(--divider);
			overflow: auto;
		}

		> .main {
			flex: 1;
			min-width: 0;
		}
	}

	&:not(.wide) {
		> .nav {
			padding: 12px 16px 0;
		}
	}
}
</style>
