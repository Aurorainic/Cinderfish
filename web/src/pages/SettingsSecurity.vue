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
				<CfInput
					v-model="currentPassword"
					class="field"
					type="password"
					autocomplete="current-password"
				>
					<template #label>{{ t("currentPassword") }}</template>
				</CfInput>
				<CfInput
					v-model="newPassword"
					class="field"
					type="password"
					autocomplete="new-password"
				>
					<template #label>{{ t("newPassword") }}</template>
				</CfInput>
				<CfInput
					v-model="newPassword2"
					class="field"
					type="password"
					autocomplete="new-password"
				>
					<template #label>{{ t("newPasswordRetype") }}</template>
				</CfInput>
				<div class="row">
					<CfButton
						primary
						:disabled="passwordBusy || !canChangePassword"
						@click="changePassword()"
					>
						<i class="ph-lock-key ph-bold ph-lg"></i>
						{{ t("changePassword") }}
					</CfButton>
				</div>
				<CfInfo v-if="passwordError != null" warn class="note">
					{{ passwordError }}
				</CfInfo>
				<CfInfo v-if="passwordNotice != null" class="note">
					{{ passwordNotice }}
				</CfInfo>

				<h2 class="sectionTitle spaced">{{ t("regenerateLoginToken") }}</h2>
				<CfInfo class="note">
					{{ t("regenerateLoginTokenDescription") }}
					新令牌会替换当前浏览器保存的凭据；其它设备上已保存的旧令牌将立即失效。
				</CfInfo>
				<div class="row">
					<CfButton
						danger
						:disabled="passwordBusy || currentPassword === ''"
						@click="regenerateToken()"
					>
						<i class="ph-arrows-clockwise ph-bold ph-lg"></i>
						{{ t("regenerateLoginToken") }}
					</CfButton>
				</div>
				<CfInfo v-if="freshToken != null" class="note">
					新令牌：<code class="_monospace">{{ freshToken }}</code>
					（已保存到本浏览器）
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("signinHistory") }}</h2>
				<div v-if="historyLoading" class="empty">加载中…</div>
				<div v-else-if="history.length === 0" class="empty">
					还没有登录记录。
				</div>
				<template v-else>
					<ul class="history">
						<li v-for="entry in history" :key="entry.id" class="historyRow">
							<i
								class="historyIcon"
								:class="entry.success ? 'ph-check-circle ph-bold' : 'ph-warning-circle ph-bold'"
							></i>
							<code class="_monospace historyIp">{{ entry.ip || "—" }}</code>
							<CfTime class="historyTime" :time="entry.createdAt" mode="absolute" />
						</li>
					</ul>
					<div v-if="historyMore" class="row">
						<CfButton :disabled="historyLoading" @click="loadHistory(true)">
							加载更多
						</CfButton>
					</div>
				</template>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">两步验证</h2>
				<template v-if="me.twoFactorEnabled">
					<CfKeyValue class="kv">
						<template #key>身份验证器应用</template>
						<template #value><span class="badge">已启用</span></template>
					</CfKeyValue>
					<CfInfo class="note">
						登录时需要在密码之后输入身份验证器应用生成的 6 位验证码。
						停用需要再次确认当前密码。
					</CfInfo>
					<CfInput
						v-if="confirmingDisable"
						v-model="password"
						class="field"
						type="password"
						autocomplete="current-password"
					>
						<template #label>{{ t("currentPassword") }}</template>
					</CfInput>
					<div class="row">
						<CfButton
							v-if="!confirmingDisable"
							danger
							@click="confirmingDisable = true"
						>
							停用两步验证
						</CfButton>
						<template v-else>
							<CfButton danger :disabled="busy" @click="unregisterTotp()">
								确认停用
							</CfButton>
							<CfButton :disabled="busy" @click="cancelDisable()">
								取消
							</CfButton>
						</template>
					</div>
				</template>

				<template v-else-if="registration == null">
					<CfInfo class="note">
						尚未启用两步验证。安装任意 TOTP 身份验证器应用（如 Aegis、1Password、
						Google Authenticator）后，在下方向导中手动输入密钥即可完成绑定。
					</CfInfo>
					<CfInput
						v-model="password"
						class="field"
						type="password"
						autocomplete="current-password"
					>
						<template #label>{{ t("currentPassword") }}</template>
					</CfInput>
					<div class="row">
						<CfButton
							primary
							:disabled="busy || password === ''"
							@click="startRegistration()"
						>
							{{ t("_2fa.registerTOTP") }}
						</CfButton>
					</div>
				</template>

				<template v-else>
					<CfInfo class="note">
						{{ t("_2fa.step1", { a: "Aegis", b: "1Password" }) }}
						{{ t("_2fa.step2Url") }}
					</CfInfo>
					<CfKeyValue class="kv" :copy="registration.secret">
						<template #key>密钥（手动输入）</template>
						<template #value>
							<code class="_monospace secret">{{ registration.secret }}</code>
						</template>
					</CfKeyValue>
					<CfKeyValue class="kv" :copy="registration.url">
						<template #key>otpauth URL</template>
						<template #value>
							<code class="_monospace url">{{ registration.url }}</code>
						</template>
					</CfKeyValue>
					<CfInput
						v-model="code"
						class="field"
						autocomplete="one-time-code"
						pattern="^[0-9]{6}$"
						:spellcheck="false"
					>
						<template #label>{{ t("_2fa.token") }}</template>
					</CfInput>
					<CfInfo class="note">{{ t("_2fa.step3") }}</CfInfo>
					<div class="row">
						<CfButton primary :disabled="busy || code === ''" @click="finishRegistration()">
							完成设置
						</CfButton>
						<CfButton :disabled="busy" @click="cancelRegistration()">取消</CfButton>
					</div>
				</template>

				<CfInfo v-if="twoFactorError != null" warn class="note">
					{{ twoFactorError }}
				</CfInfo>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish's security page: `X2fa` (TOTP registration and removal), the
// password dialog, `i/signin-history` and `i/regenerate-token`. All four have
// endpoints, so all four are live; the backup-code list stays empty because
// backup codes are not generated.
import { computed, onMounted, ref } from "vue";
import { addStoredAccount, me, updateMe } from "@/account";
import type {
	MeDetailed,
	SigninHistoryEntry,
	TwoFactorRegistration,
} from "@/api/types";
import { api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTime from "@/components/ui/CfTime.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsSecurity" });

const loading = ref(true);
const errorText = ref<string | null>(null);

const password = ref("");
const code = ref("");
const registration = ref<TwoFactorRegistration | null>(null);
const confirmingDisable = ref(false);
const busy = ref(false);
const twoFactorError = ref<string | null>(null);

const currentPassword = ref("");
const newPassword = ref("");
const newPassword2 = ref("");
const passwordBusy = ref(false);
const passwordError = ref<string | null>(null);
const passwordNotice = ref<string | null>(null);
const freshToken = ref<string | null>(null);

const history = ref<SigninHistoryEntry[]>([]);
const historyLoading = ref(false);
const historyMore = ref(false);

const canChangePassword = computed(
	() =>
		currentPassword.value !== "" &&
		newPassword.value !== "" &&
		newPassword.value === newPassword2.value,
);

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

function describeTwoFactorError(error: unknown, fallback: string): string {
	return error instanceof Error && error.message !== ""
		? error.message
		: fallback;
}

async function changePassword(): Promise<void> {
	if (passwordBusy.value) return;
	passwordError.value = null;
	passwordNotice.value = null;
	if (newPassword.value !== newPassword2.value) {
		passwordError.value = "两次输入的新密码不一致。";
		return;
	}
	passwordBusy.value = true;
	try {
		await api("i/change-password", {
			currentPassword: currentPassword.value,
			newPassword: newPassword.value,
		});
		currentPassword.value = "";
		newPassword.value = "";
		newPassword2.value = "";
		passwordNotice.value = "密码已更新。其它设备上的会话不受影响。";
	} catch (error) {
		passwordError.value = describeTwoFactorError(
			error,
			"无法修改密码，请检查当前密码。",
		);
	} finally {
		passwordBusy.value = false;
	}
}

async function regenerateToken(): Promise<void> {
	if (passwordBusy.value) return;
	passwordError.value = null;
	passwordNotice.value = null;
	freshToken.value = null;
	passwordBusy.value = true;
	try {
		const result = await api("i/regenerate-token", {
			password: currentPassword.value,
		});
		freshToken.value = result.token;
		updateMe({ token: result.token });
		if (me.value != null) {
			addStoredAccount({ id: me.value.id, token: result.token });
		}
		currentPassword.value = "";
	} catch (error) {
		passwordError.value = describeTwoFactorError(
			error,
			"无法重新生成令牌，请检查当前密码。",
		);
	} finally {
		passwordBusy.value = false;
	}
}

async function loadHistory(more = false): Promise<void> {
	if (historyLoading.value) return;
	historyLoading.value = true;
	try {
		const last = more ? history.value[history.value.length - 1] : undefined;
		const rows = await api("i/signin-history", {
			limit: 10,
			...(last != null ? { untilId: last.id } : {}),
		});
		history.value = more ? [...history.value, ...rows] : rows;
		historyMore.value = rows.length === 10;
	} catch {
		historyMore.value = false;
	} finally {
		historyLoading.value = false;
	}
}

async function startRegistration(): Promise<void> {
	if (busy.value) return;
	busy.value = true;
	twoFactorError.value = null;
	try {
		registration.value = await api("i/2fa/register", {
			password: password.value,
		});
		password.value = "";
	} catch (error) {
		twoFactorError.value = describeTwoFactorError(
			error,
			"无法开始两步验证设置，请检查当前密码。",
		);
	} finally {
		busy.value = false;
	}
}

async function finishRegistration(): Promise<void> {
	if (busy.value) return;
	busy.value = true;
	twoFactorError.value = null;
	try {
		await api("i/2fa/done", { token: code.value.replace(/\s/g, "") });
		registration.value = null;
		code.value = "";
		await load();
	} catch (error) {
		twoFactorError.value = describeTwoFactorError(
			error,
			"验证码未能通过校验，请重新输入。",
		);
	} finally {
		busy.value = false;
	}
}

function cancelRegistration(): void {
	registration.value = null;
	code.value = "";
	twoFactorError.value = null;
}

async function unregisterTotp(): Promise<void> {
	if (busy.value) return;
	busy.value = true;
	twoFactorError.value = null;
	try {
		await api("i/2fa/unregister", { password: password.value });
		password.value = "";
		confirmingDisable.value = false;
		await load();
	} catch (error) {
		twoFactorError.value = describeTwoFactorError(
			error,
			"无法停用两步验证，请检查当前密码。",
		);
	} finally {
		busy.value = false;
	}
}

function cancelDisable(): void {
	confirmingDisable.value = false;
	password.value = "";
	twoFactorError.value = null;
}

onMounted(() => {
	void load();
	void loadHistory();
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

	.field {
		margin-block-start: 12px;
	}

	.spaced {
		margin-block-start: 24px;
	}

	.history {
		margin: 8px 0 0;
		padding: 0;
		list-style: none;
	}

	.historyRow {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 0;

		&:not(:last-child) {
			border-bottom: 1px solid var(--divider);
		}
	}

	.historyIcon {
		flex: none;
		color: var(--success);

		&.ph-warning-circle {
			color: var(--error);
		}
	}

	.historyIp {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.historyTime {
		margin-inline-start: auto;
		opacity: 0.7;
	}

	.secret,
	.url {
		word-break: break-all;
		user-select: all;
	}
}
</style>
