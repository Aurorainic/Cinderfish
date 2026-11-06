<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/integration.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-integration">
		<CfInfo warn class="note">
			<strong>第三方账号登录未实现。</strong>
			Firefish 的这一页用于把 Discord / GitHub / Twitter 账号与本站账号绑定，Cinderfish
			既没有这些 OAuth 客户端，也没有对应的账号关联表；后端不会接受任何第三方登录回调。
		</CfInfo>

		<section class="_formBlock">
			<h2 class="sectionTitle">这里已经有</h2>
			<ul class="list">
				<li class="item ok">
					<i class="ph-check-circle ph-bold"></i>
					<div class="body">
						<span class="label">第三方应用授权（MiAuth）</span>
						<span class="note"
							>应用用 MiAuth 链接申请访问令牌，用户在授权页确认权限</span
						>
					</div>
					<RouterLink class="link" to="/settings/apps">已授权应用</RouterLink>
				</li>
				<li class="item ok">
					<i class="ph-check-circle ph-bold"></i>
					<div class="body">
						<span class="label">ActivityPub 联邦登录不了，但可以关注</span>
						<span class="note"
							>Fediverse 账号之间通过关注互通，不需要把账号绑定到本站</span
						>
					</div>
				</li>
			</ul>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">上游有、这里没有</h2>
			<ul class="list">
				<li v-for="item in missing" :key="item" class="item">
					<i class="ph-x-circle ph-bold"></i>
					<div class="body">
						<span class="label">{{ item }}</span>
					</div>
				</li>
			</ul>
			<CfInfo class="note">
				这些功能需要实例管理员配置外部 OAuth 应用，并增加账号关联表；首版范围明确不做。
				与其放置点了没反应的表单，这个页面直接说明现状。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish's integration settings bind a Cinderfish account to Discord, GitHub
// and Twitter. The product scope excludes third-party sign-in, so the page
// explains the gap and points at the authorization surface that does exist.
import CfInfo from "@/components/ui/CfInfo.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsIntegration" });

const missing = [
	"Discord 账号绑定与登录",
	"GitHub 账号绑定与登录",
	"Twitter / X 账号绑定与登录",
	"Matrix 房间通知",
];

definePageMetadata({
	title: "集成",
	icon: "ph-plugs-connected ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-integration {
	.note {
		margin-bottom: 12px;
	}

	.sectionTitle {
		font-size: 1.05em;
		font-weight: 700;
		margin: 0 0 12px;
	}

	.list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 10px 12px;
		border-radius: var(--radius);
		background: var(--panelHighlight);

		&.ok > i {
			color: var(--success);
		}

		> i {
			margin-top: 2px;
			color: var(--error);
		}

		.body {
			flex: 1;
			display: flex;
			flex-direction: column;
			gap: 2px;
		}

		.label {
			font-weight: 600;
		}

		.note {
			opacity: 0.8;
			font-size: 0.92em;
		}

		.link {
			color: var(--accent);
			white-space: nowrap;
		}
	}
}
</style>
