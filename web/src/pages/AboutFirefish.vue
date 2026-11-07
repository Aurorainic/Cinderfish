<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="600" :margin-min="20">
			<div class="_formRoot cf-about-firefish">
				<section class="hero">
					<div class="name">Cinderfish</div>
					<div v-if="softwareVersion" class="version">v{{ softwareVersion }}</div>
					<p class="instance">{{ instance.name || host }}</p>
				</section>

				<CfInfo class="intro">
					Cinderfish 是一个以 Misskey API 为兼容目标、支持 ActivityPub 联邦的个人实例实现。
					前端按 Firefish 客户端的页面与组件结构移植，后端是 Go 服务。
				</CfInfo>

				<section class="links">
					<CfA
						v-if="repositoryUrl !== ''"
						class="linkRow"
						:to="repositoryUrl"
						behavior="browser"
					>
						<i class="ph-code ph-bold ph-lg"></i>
						源码仓库
						<span class="suffix">{{ repositoryUrl }}</span>
					</CfA>
					<CfA class="linkRow" to="https://misskey-hub.net/" behavior="browser">
						<i class="ph-book-open ph-bold ph-lg"></i>
						Misskey（API 契约来源）
						<span class="suffix">misskey-hub.net</span>
					</CfA>
					<CfA class="linkRow" to="https://joinfirefish.org/" behavior="browser">
						<i class="ph-fire ph-bold ph-lg"></i>
						Firefish（前端结构参考）
						<span class="suffix">joinfirefish.org</span>
					</CfA>
					<CfA class="linkRow" to="https://iceshrimp.dev/" behavior="browser">
						<i class="ph-planet ph-bold ph-lg"></i>
						Iceshrimp（补充参考）
						<span class="suffix">iceshrimp.dev</span>
					</CfA>
				</section>

				<section class="card">
					<h2 class="sectionTitle">实现说明</h2>
					<ul class="points">
						<li>后端：Go + PostgreSQL，Redis 可选；媒体支持本地文件与 S3 兼容存储。</li>
						<li>前端：Vue 3 + TypeScript + Vite，按 Firefish 的页面/组件结构翻译。</li>
						<li>接口：Misskey 的 <code>paramDef</code> 与响应 schema 是契约来源；Mastodon API 不支持。</li>
						<li>联邦：ActivityPub 入站与出站投递，支持公开、白名单与关闭三种模式。</li>
					</ul>
				</section>

				<section class="card">
					<h2 class="sectionTitle">许可证与致谢</h2>
					<p>
						本项目以 AGPL-3.0-only 分发。参考的 Misskey、Firefish 与 Iceshrimp 客户端同为
						AGPL-3.0；复制或改写自参考实现的文件保留其版权头，来源记录见仓库的
						<code>docs/HANDOVER.md</code>。
					</p>
					<p>
						感谢 Misskey 提供稳定的后端契约，感谢 Firefish 与 Iceshrimp 提供的界面结构参考。
					</p>
				</section>

				<div class="actions">
					<CfButton rounded link to="/about">实例信息</CfButton>
					<CfButton rounded link to="/">返回首页</CfButton>
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Cinderfish's own "about this software" page. Firefish's page advertises the
// Firefish project and its contributors; this port describes Cinderfish and
// credits the three reference implementations instead, so it never claims to
// be Firefish.
import { computed } from "vue";
import { instance } from "@/instance";
import { host } from "@/config";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAboutFirefish" });

// The backend reports the version through meta, so the page never carries a
// second copy that could drift from the release.
const softwareVersion = computed(() => instance.version);
const repositoryUrl = computed(() => instance.repositoryUrl || "https://github.com/Aurorainic/Cinderfish");

definePageMetadata({
	title: "关于 Cinderfish",
	icon: "ph-info ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-about-firefish {
	.hero {
		text-align: center;
		padding: 24px 0 8px;

		.name {
			font-size: 2em;
			font-weight: bold;
		}

		.version {
			margin-block-start: 4px;
			color: var(--fgTransparentWeak);
			font-size: 0.9em;
		}

		.instance {
			margin: 8px 0 0;
			font-size: 0.9em;
		}
	}

	.intro {
		margin-block-start: 1em;
	}

	.links {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-block-start: 1em;
	}

	.linkRow {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 12px;
		border-radius: var(--radius);
		background: var(--panel);
		color: var(--fg);
		text-decoration: none;

		&:hover {
			background: var(--panelHighlight);
		}

		.suffix {
			margin-inline-start: auto;
			color: var(--fgTransparentWeak);
			font-size: 0.8em;
		}
	}

	.card {
		margin-block-start: 12px;
		padding: 16px;
		background: var(--panel);
		border: 1px solid var(--divider);
		border-radius: var(--radius);

		.sectionTitle {
			margin: 0 0 8px;
			font-size: 1em;
			opacity: 0.85;
		}

		p {
			margin: 8px 0 0;
			line-height: 1.6;
		}
	}

	.points {
		margin: 0;
		padding-inline-start: 1.2em;
		line-height: 1.7;
	}

	.actions {
		display: flex;
		gap: 8px;
		margin-block-start: 1.5em;
	}
}
</style>
