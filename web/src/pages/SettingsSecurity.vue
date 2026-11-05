<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/security.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-security">
		<div v-if="loading" class="state"><CfLoading /></div>
		<div v-else-if="errorText != null" class="state">
			<CfInfo warn>{{ errorText }}</CfInfo>
			<CfButton class="retry" inline @click="load()">{{ t("retry") }}</CfButton>
		</div>
		<div v-else-if="me == null" class="state">
			<CfInfo>{{ t("signinRequired") }}</CfInfo>
		</div>
		<template v-else>
			<section class="_formBlock">
				<h2 class="sectionTitle">会话信息</h2>
				<CfKeyValue class="kv">
					<template #key>{{ t("username") }}</template>
					<template #value>@{{ me.username }}</template>
				</CfKeyValue>
				<CfKeyValue class="kv" :copy="me.id">
					<template #key">用户 ID</template>
					<template #value><code class="_monospace">{{ me.id }}</code></template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>{{ t("createdAt") }}</template>
					<template #value>
						<CfTime :time="me.createdAt" mode="absolute" />
					</template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>已登录天数</template>
					<template #value>{{ me.loggedInDays }}</template>
				</CfKeyValue>
				<CfKeyValue v-if="me.email != null" class="kv">
					<template #key>{{ t("email") }}</template>
					<template #value>
						{{ me.email }}
						<span v-if="me.emailVerified" class="badge">
							{{ t("emailVerified") }}
						</span>
					</template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>角色</template>
					<template #value>
						<template v-if="me.isAdmin">{{ t("administrator") }}</template>
						<template v-else-if="me.isModerator">{{ t("moderator") }}</template>
						<template v-else>普通用户</template>
					</template>
				</CfKeyValue>
				<CfInfo class="note">
					会话凭据由账号层保管（<code>@/account</code>），页面不读取也不显示 token。
					登录信息只反映当前会话的账号资料，服务端没有会话列表端点。
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("password") }}</h2>
				<div class="row">
					<CfButton primary disabled>
						<i class="ph-lock-key ph-bold ph-lg"></i>
						{{ t("changePassword") }}
					</CfButton>
					<CfButton danger disabled>
						<i class="ph-arrows-clockwise ph-bold ph-lg"></i>
						{{ t("regenerateLoginToken") }}
					</CfButton>
				</div>
				<CfInfo class="note">
					修改密码所需的 <code>i/change-password</code>、重新生成令牌所需的
					<code>i/regenerate-token</code> 在本实例<strong>未提供</strong>，因此两个按钮保持禁用。
					{{ t("regenerateLoginTokenDescription") }}
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("signinHistory") }}</h2>
				<div class="empty">
					登录历史端点 <code>i/signin-history</code> <strong>未提供</strong>，没有可显示的记录。
				</div>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">两步验证</h2>
				<CfKeyValue class="kv">
					<template #key>{{ t("securityKey") }}</template>
					<template #value>{{ t("notSet") }}</template>
				</CfKeyValue>
				<CfInfo class="note">
					两步验证与安全密钥（<code>i/2fa/*</code>、<code>i/security-keys/*</code>）
					在本实例<strong>未提供</strong>；备份码余量字段存在但不可操作。
				</CfInfo>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish's security page: password change dialog, X2fa, `i/signin-history`
// pagination and `i/regenerate-token`. None of the four endpoints exist in
// Cinderfish (docs/research/2026-11-04-misskey-api-coverage.md), so this page
// reports the session's real state (from `POST /api/i`) and marks every missing
// capability as unavailable instead of rendering dead buttons that look live.
import { onMounted, ref } from "vue";
import { me, updateMe } from "@/account";
import type { MeDetailed } from "@/api/types";
import { api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTime from "@/components/ui/CfTime.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsSecurity" });

const loading = ref(true);
const errorText = ref<string | null>(null);

async function load(): Promise<void> {
	loading.value = true;
	errorText.value = null;
	try {
		const fresh = await api("i");
		updateMe(fresh as Partial<MeDetailed>);
	} catch (error) {
		errorText.value =
			error instanceof Error && error.message !== ""
				? error.message
				: t("somethingHappened");
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("security"), icon: "ph-lock ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-security {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.state {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12px;
	}

	.kv {
		margin-block-start: 8px;
	}

	.badge {
		margin-inline-start: 8px;
		padding: 1px 6px;
		border-radius: var(--radius);
		background: var(--accentedBg);
		color: var(--accent);
		font-size: 0.8em;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 12px;
	}

	.empty {
		margin-block-start: 8px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.note {
		margin-block-start: 12px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}
}
</style>
