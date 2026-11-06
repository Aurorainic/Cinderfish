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
					<div class="links">
						<a
							v-for="link in policyLinks"
							:key="link.href"
							class="link"
							:href="link.href"
							target="_blank"
							rel="noopener"
							>{{ link.label }}</a
						>
					</div>
				</div>

				<div v-if="serverRules.length > 0" class="_formBlock">
					<div class="sectionLabel">实例规则</div>
					<ol class="rules">
						<li v-for="(rule, index) in serverRules" :key="index">{{ rule }}</li>
					</ol>
				</div>

				<div class="_formBlock">
					<div class="sectionLabel">{{ t("statistics") }}</div>
					<CfInfo v-if="statsError !== null" icon="chart-bar">
						统计暂时不可用（{{ statsError }}）。页面其余部分不受影响。
					</CfInfo>
					<div v-else-if="stats != null" class="stats">
						<div class="stat">
							<span class="value">{{ stats.usersCount }}</span>
							<span class="label">用户</span>
						</div>
						<div class="stat">
							<span class="value">{{ stats.notesCount }}</span>
							<span class="label">帖子</span>
						</div>
						<div class="stat">
							<span class="value">{{ stats.instancesCount }}</span>
							<span class="label">已知实例</span>
						</div>
						<div class="stat">
							<span class="value">{{ stats.driveFiles }}</span>
							<span class="label">网盘文件</span>
						</div>
					</div>
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
import type { InstanceStats, MetaDetailed, MetaLite } from "@/api/types";
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
const stats = ref<InstanceStats | null>(null);
const statsError = ref<string | null>(null);

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
const policyLinks = computed(() => {
	const items: { label: string; href: string }[] = [];
	if (view.value.tosUrl != null && view.value.tosUrl !== "") {
		items.push({ label: t("tos"), href: view.value.tosUrl });
	}
	if (view.value.privacyPolicyUrl !== "") {
		items.push({ label: "隐私政策", href: view.value.privacyPolicyUrl });
	}
	if (view.value.repositoryUrl !== "") {
		items.push({ label: "源码仓库", href: view.value.repositoryUrl });
	}
	if (view.value.feedbackUrl !== "") {
		items.push({ label: "反馈", href: view.value.feedbackUrl });
	}
	if (view.value.inquiryUrl !== "") {
		items.push({ label: "咨询", href: view.value.inquiryUrl });
	}
	if (view.value.impressumUrl !== "") {
		items.push({ label: "版本说明", href: view.value.impressumUrl });
	}
	return items;
});

const serverRules = computed<string[]>(() => view.value.serverRules ?? []);

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

	try {
		stats.value = await api("stats", {});
	} catch (cause) {
		statsError.value =
			cause instanceof Error ? cause.message : String(cause);
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

.rules {
	margin: 0;
	padding-left: 20px;
	font-size: 0.9em;
	line-height: 1.6;
}

.stats {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
	gap: 8px;
}

.stat {
	display: flex;
	flex-direction: column;
	padding: 12px;
	background: var(--panel);
	border-radius: var(--radius);

	.value {
		font-size: 1.3em;
		font-weight: 700;
	}

	.label {
		font-size: 0.78em;
		opacity: 0.65;
	}
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
