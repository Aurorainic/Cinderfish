<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/email-settings.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-email-settings">
				<CfInfo warn>
					需要管理员权限，无法查看邮件配置。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-email-settings">
				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<form v-else class="_formRoot" @submit.prevent="save">
					<section class="_formBlock">
						<h2 class="sectionTitle">{{ t("emailServer") }}</h2>
						<p class="sectionNote">
							当前状态：
							<span v-if="configured" class="badge on">已配置</span>
							<span v-else class="badge">未配置（邮件写入服务日志）</span>
							<span class="source">
								· 来源：{{ stored ? "管理台设置" : "config.toml" }}
							</span>
						</p>

						<CfInput v-model="host" class="_formBlock" :disabled="saving">
							<template #label>SMTP 主机</template>
							<template #caption>留空表示不投递，邮件只写日志。</template>
						</CfInput>
						<CfInput v-model="port" class="_formBlock" type="number" :disabled="saving">
							<template #label>端口</template>
							<template #caption>默认 587；隐式 TLS 常用 465。</template>
						</CfInput>
						<CfInput v-model="username" class="_formBlock" :disabled="saving">
							<template #label>用户名（可选）</template>
						</CfInput>
						<CfInput
							v-model="password"
							class="_formBlock"
							type="password"
							:disabled="saving"
						>
							<template #label>密码</template>
							<template #caption>
								{{ passwordSet ? "已配置；留空保留现有密码。密码不会回传。" : "尚未配置。" }}
							</template>
						</CfInput>
						<CfInput v-model="from" class="_formBlock" :disabled="saving">
							<template #label>发件人（可选）</template>
							<template #caption>留空使用 no-reply@&lt;主机名&gt;。</template>
						</CfInput>
						<CfSelect v-model="tls" class="_formBlock" :disabled="saving">
							<template #label>TLS 模式</template>
							<option value="starttls">STARTTLS（默认，提交端口）</option>
							<option value="tls">隐式 TLS（465）</option>
							<option value="none">不加密（仅测试）</option>
						</CfSelect>

						<CfInfo v-if="formError !== null" warn class="_formBlock">
							{{ formError }}
						</CfInfo>
						<CfInfo v-else-if="saved" class="_formBlock">
							已保存并立即生效，新邮件会走这套设置。
						</CfInfo>

						<CfButton type="submit" primary :disabled="saving">
							{{ saving ? "保存中…" : "保存" }}
						</CfButton>
					</section>
				</form>

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
							<strong>通知邮件</strong>：用户可在「邮箱设置」页勾选提及、回复、关注、
							关注请求四类（<code>i/update</code> 的 <code>emailNotificationTypes</code>），
							事件发生时按用户的选择投递。
						</li>
					</ul>
					<CfInfo class="note">
						这里保存的设置覆盖 <code>config.toml</code> 的 <code>[mail]</code> 段与对应的环境变量，无需重启；密码保存在实例数据库中，接口只返回「已配置」状态。
					</CfInfo>
				</section>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Runtime SMTP configuration. The server wraps the sender in a
// mail.SwitchableSender, so saving this form changes delivery immediately;
// the persisted override lives in instance_meta and wins over config.toml
// (and the CINDERFISH_MAIL_* environment variables) from the next start too.
import { computed, onMounted, ref } from "vue";
import { isAdmin } from "@/account";
import { ApiError, api } from "@/api/client";
import type { MailSettingsInfo } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import MkError from "@/components/global/MkError.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminEmailSettings" });

const admin = computed<boolean>(() => isAdmin());
const loading = ref(false);
const loadError = ref<string | null>(null);
const saving = ref(false);
const saved = ref(false);
const formError = ref<string | null>(null);

const configured = ref(false);
const stored = ref(false);
const passwordSet = ref(false);
const host = ref<string | number | null>("");
const port = ref<string | number | null>(587);
const username = ref<string | number | null>("");
const password = ref("");
const from = ref<string | number | null>("");
const tls = ref<"starttls" | "tls" | "none">("starttls");

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

function apply(info: MailSettingsInfo): void {
	configured.value = info.configured;
	stored.value = info.stored;
	passwordSet.value = info.passwordSet;
	host.value = info.host;
	port.value = info.port;
	username.value = info.username;
	password.value = "";
	from.value = info.from ?? "";
	tls.value = info.tls;
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		apply(await api("admin/email-settings", {}));
	} catch (cause) {
		loadError.value = errorMessage(cause);
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value) return;
	formError.value = null;
	saved.value = false;
	const portNumber = Number(port.value);
	if (!Number.isInteger(portNumber) || portNumber < 1 || portNumber > 65535) {
		formError.value = "端口必须是 1–65535 之间的整数。";
		return;
	}
	saving.value = true;
	try {
		apply(
			await api("admin/email-settings/update", {
				host: String(host.value ?? "").trim(),
				port: portNumber,
				username: String(username.value ?? "").trim(),
				password: password.value.trim(),
				from: String(from.value ?? "").trim(),
				tls: tls.value,
			}),
		);
		saved.value = true;
	} catch (cause) {
		formError.value = errorMessage(cause);
	} finally {
		saving.value = false;
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

	.source {
		margin-inline-start: 6px;
		font-size: 0.85em;
		opacity: 0.65;
	}

	.sectionTitle {
		margin: 0 0 8px;
		font-size: 1.05em;
	}

	.sectionNote {
		margin: 0 0 12px;
		font-size: 0.9em;
	}

	.flows {
		margin: 0;
		padding-inline-start: 1.4em;
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 0.9em;
	}

	.note {
		margin-top: 12px;
	}
}
</style>
