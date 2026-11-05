<!-- Ported from firefish@b425f61:packages/client/src/pages/about.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600">
			<CfLoading v-if="loading" />
			<MkError v-else-if="failed" @retry="load" />
			<div v-else class="_formRoot">
				<div class="hero _formBlock" :style="heroStyle">
					<div class="content">
						<img class="icon" :src="iconUrl" alt="" />
						<div class="name">
							<b>{{ displayName }}</b>
						</div>
					</div>
				</div>

				<CfKeyValue class="_formBlock">
					<template #key>{{ t("description") }}</template>
					<template #value>
						<Mfm v-if="description !== ''" :text="description" />
						<span v-else class="empty">—</span>
					</template>
				</CfKeyValue>

				<div class="_formBlock version">
					<CfKeyValue v-if="version !== ''" :copy="version">
						<template #key>{{ t("version") }}</template>
						<template #value>{{ version }}</template>
					</CfKeyValue>
					<CfKeyValue class="_formBlock">
						<template #key>{{ t("host") }}</template>
						<template #value>{{ host }}</template>
					</CfKeyValue>
					<CfA class="link" to="/about-firefish">{{
						t("aboutFirefish")
					}}</CfA>
				</div>

				<div class="_formBlock">
					<CfKeyValue>
						<template #key>{{ t("administrator") }}</template>
						<template #value>{{
							maintainerName !== "" ? maintainerName : "—"
						}}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("contact") }}</template>
						<template #value>{{
							maintainerEmail !== "" ? maintainerEmail : "—"
						}}</template>
					</CfKeyValue>
					<a
						v-if="tosUrl != null && tosUrl !== ''"
						class="link"
						:href="tosUrl"
						target="_blank"
						rel="noopener"
						>{{ t("tos") }}</a
					>
				</div>

				<div class="_formBlock">
					<div class="sectionLabel">{{ t("statistics") }}</div>
					<CfInfo icon="chart-bar">
						统计接口尚未提供：后端没有注册 <code>stats</code> 端点（见
						docs/research/api-contract/meta-federation.md），因此用户数、帖子数无法获取，此处不显示数字。
					</CfInfo>
				</div>

				<div class="_formBlock">
					<div class="sectionLabel">Well-known resources</div>
					<div class="links">
						<a
							v-for="resource in wellKnownResources"
							:key="resource"
							class="link"
							:href="resource"
							target="_blank"
							rel="noopener"
							>{{ resource }}</a
						>
					</div>
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's about.vue is a four-slide swiper (overview / emojis / charts /
// federation) built on the `swiper` package, which is not part of the
// Cinderfish dependency set (wave-3 contract §2). The overview slide is ported
// as a single scrolling column; the other three slides are dropped because
// their data sources do not exist:
//   * charts  -> MkInstanceStats needs the `stats` / `charts` endpoints;
//   * emojis  -> covered by the admin emoji page, kept out of About;
//   * federation -> needs admin-only instance lists (`federation/instances`).
// The icon easter egg (woozy mode) is dropped: defaultStore has no
// `woozyMode` flag and /static-assets/woozy.png is not shipped.
import { computed, onMounted, ref } from "vue";
import type { MetaDetailed, MetaLite } from "@/api/types";
import { api } from "@/api/client";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { host } from "@/config";
import { t } from "@/i18n";
import { instance } from "@/instance";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAbout" });

const loading = ref(true);
const failed = ref(false);
const meta = ref<MetaDetailed | null>(null);

/** Falls back to the cached meta so the hero does not flash empty. */
const view = computed<MetaLite>(() => meta.value ?? instance);

const displayName = computed(() => {
	const name = view.value.name;
	return name != null && name !== "" ? name : host;
});

// meta.description is HTML in Firefish and is rendered with v-html there; the
// backend always returns "" (meta-federation.md). It is parsed as MFM instead
// so a future non-empty description cannot inject markup.
const description = computed(() => view.value.description ?? "");

const version = computed(() => view.value.version ?? "");
const maintainerName = computed(() => view.value.maintainerName ?? "");
const maintainerEmail = computed(() => view.value.maintainerEmail ?? "");
// MetaLite types tosUrl as the literal null; the backend may gain a real URL.
const tosUrl = computed<string | null>(
	() => (view.value.tosUrl as string | null) ?? null,
);

const iconUrl = computed(() => {
	const icon = view.value.iconUrl;
	return icon != null && icon !== "" ? icon : "/favicon.ico";
});

const heroStyle = computed<Record<string, string>>(() => {
	const banner = view.value.bannerUrl;
	if (banner != null && banner !== "") {
		return { backgroundImage: `url(${banner})` };
	}
	return {
		backgroundImage:
			"linear-gradient(135deg, var(--accentedBg), transparent)",
	};
});

const wellKnownResources = [
	"/.well-known/host-meta",
	"/.well-known/host-meta.json",
	"/.well-known/nodeinfo",
	"/robots.txt",
	"/manifest.json",
];

const headerActions = computed(() => []);

async function load(): Promise<void> {
	loading.value = true;
	failed.value = false;
	try {
		meta.value = (await api("meta", { detail: true })) as MetaDetailed;
	} catch {
		failed.value = true;
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("instanceInfo"),
	icon: "ph-info ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.hero {
	overflow: clip;
	border-radius: var(--radius);
	background-size: cover;
	background-position: center center;
	text-align: center;

	> .content {
		overflow: hidden;

		> .icon {
			display: block;
			margin: 16px auto 0 auto;
			height: 64px;
			border-radius: 8px;
		}

		> .name {
			display: block;
			padding: 16px;
			color: #e0def4;
			text-shadow: 0 0 8px var(--shadow);
			background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));
		}
	}
}

.version {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 6px;
}

.sectionLabel {
	padding-block-end: 6px;
	font-size: 0.85em;
	color: var(--fgTransparentWeak);
}

.links {
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.link {
	color: var(--accent);
	text-decoration: none;

	&:hover {
		text-decoration: underline;
	}
}

.empty {
	color: var(--fgTransparentWeak);
}
</style>
