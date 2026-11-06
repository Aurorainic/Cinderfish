<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/email-settings.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-email-settings">
				<CfInfo warn>
					需要管理员权限，无法查看邮件设置。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>

			<div v-else class="cf-admin-email-settings">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError != null" warn>
					{{ loadError }}
					<CfButton inline @click="load">{{ t("retry") }}</CfButton>
				</CfInfo>

				<template v-else>
					<section class="_formBlock">
						<h2 class="sectionTitle">{{ t("emailServer") }}</h2>
						<CfKeyValue>
							<template #key>状态</template>
							<template #value>
								<span v-if="configured" class="badge on">已配置</span>
								<span v-else class="badge">未配置</span>
							</template>
						</CfKeyValue>
						<CfKeyValue>
							<template #key>SMTP 主机</template>
							<template #value>
								<code class="_monospace">{{ host || "（未设置）" }}</code>
							</template>
						</CfKeyValue>
						<CfInfo class="note">
							<template v-if="configured">
								验证邮件与密码重置链接会经由这台服务器发出。端口、账号、密码、
								发件人与 TLS 模式都在实例的
								<code>config.toml</code> 的 <code>[mail]</code> 段里，不在本页面编辑——
								这些是启动配置，改完需要重启服务。
							</template>
							<template v-else>
								没有配置 SMTP 主机，验证邮件与密码重置链接不会真的投递，
								而是整封写进服务日志（<code>[mail] host</code> 为空时的日志模式）。
								要让邮件真正发出，请在 <code>config.toml</code> 的
								<code>[mail]</code> 段填写 <code>host</code>、<code>from</code> 等字段后重启。
							</template>
						</CfInfo>
					</section>

					<section class="_formBlock">
						<h2 class="sectionTitle">可用的邮件流程</h2>
						<ul class="flows">
							<li>
								<strong>邮箱验证</strong>：<code>i/update-email</code> 改地址后发送验证链接，
								<code>verify-email</code> 核销；<code>i/email-verification</code> 可重发。
							</li>
							<li>
								<strong>密码重置</strong>：<code>request-reset-password</code> 恒回 204，
								<code>reset-password</code> 用链接里的 token 设置新密码。
							</li>
							<li>
								<strong>通知邮件</strong>：尚未实现——<code>i/update</code> 不存储
								<code>emailNotificationTypes</code>，所以实例里没有可发的通知邮件。
							</li>
						</ul>
					</section>

					<section class="_formBlock">
						<h2 class="sectionTitle">环境变量</h2>
						<CfInfo class="note">
							这些值也可以用 <code>CINDERFISH_MAIL_HOST</code>、
							<code>CINDERFISH_MAIL_PORT</code>、<code>CINDERFISH_MAIL_USERNAME</code>、
							<code>CINDERFISH_MAIL_PASSWORD</code>、<code>CINDERFISH_MAIL_FROM</code>、
							<code>CINDERFISH_MAIL_TLS</code> 覆盖配置文件。
						</CfInfo>
					</section>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's admin/email-settings edits SMTP credentials at runtime. Cinderfish
// reads them from config.toml at startup only, so this page reports the state
// and explains where the settings live instead of rendering fields that would
// silently do nothing. `admin/meta` exposes the host (public knowledge, it shows
// up in every message's Received header) and nothing else.
import { computed, onMounted, ref } from "vue";
import { isAdmin } from "@/account";
import { ApiError, api } from "@/api/client";
import type { AdminInstanceMeta } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminEmailSettings" });

const admin = computed<boolean>(() => isAdmin());
const loading = ref(false);
const loadError = ref<string | null>(null);
const host = ref("");

const configured = computed(() => host.value !== "");

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		const meta: AdminInstanceMeta = await api("admin/meta", {});
		host.value = meta.emailServer ?? "";
	} catch (cause) {
		loadError.value = errorMessage(cause);
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("emailServer"), icon: "ph-envelope-simple ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-admin-email-settings {
	.badge {
		padding: 1px 8px;
		border-radius: 999px;
		background: var(--bg);
		font-size: 0.8em;
		opacity: 0.75;
	}

	.badge.on {
		color: var(--success, #3ba55d);
		opacity: 1;
	}

	.note {
		margin-top: 12px;
	}

	code {
		font-family: monospace;
		background: var(--bg);
		padding: 1px 4px;
		border-radius: 4px;
	}

	.flows {
		margin: 0;
		padding-inline-start: 20px;
		font-size: 0.9em;
		line-height: 1.8;
	}
}
</style>
