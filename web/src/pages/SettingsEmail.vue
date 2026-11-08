<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/email.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-email">
		<CfInfo v-if="me == null">{{ t("signinRequired") }}</CfInfo>
		<template v-else>
			<section class="_formBlock">
				<h2 class="_sectionTitle">{{ t("emailAddress") }}</h2>
				<CfKeyValue v-if="hasEmail">
					<template #key>当前地址</template>
					<template #value>
						<span class="address">{{ me.email }}</span>
						<span v-if="me.emailVerified" class="badge verified">
							{{ t("emailVerified") }}
						</span>
						<span v-else class="badge">{{ t("unverified") }}</span>
					</template>
				</CfKeyValue>
				<CfInfo v-else class="note">{{ t("emailNotConfiguredWarning") }}</CfInfo>
				<CfInfo v-if="hasEmail && !me.emailVerified" class="note">
					{{ t("verificationEmailSent") }}
				</CfInfo>

				<CfInput
					v-model="draft"
					class="field"
					type="email"
					autocomplete="email"
					:spellcheck="false"
				>
					<template #label>{{
						hasEmail ? "更换为其它地址" : "填写邮箱地址"
					}}</template>
				</CfInput>
				<div class="_row">
					<CfButton primary :disabled="busy || !canSave" @click="save()">
						<i class="ph-floppy-disk ph-bold ph-lg"></i>
						{{ t("save") }}
					</CfButton>
					<CfButton
						v-if="hasEmail && !me.emailVerified"
						:disabled="busy"
						@click="resend()"
					>
						<i class="ph-paper-plane-tilt ph-bold ph-lg"></i>
						重新发送验证邮件
					</CfButton>
				</div>
				<CfInfo v-if="errorText != null" warn class="note">{{ errorText }}</CfInfo>
				<CfInfo v-if="notice != null" class="note">{{ notice }}</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="_sectionTitle">{{ t("emailNotification") }}</h2>
				<CfInfo v-if="!me.emailVerified" warn class="note">
					邮箱还没验证，下面的开关不会发出任何邮件。请先完成验证。
				</CfInfo>

				<CfSwitch
					:model-value="me.receiveAnnouncementEmail"
					:disabled="savingMail"
					class="_formBlock"
					@update:model-value="onMailSwitch('receiveAnnouncementEmail', $event)"
				>
					接收实例公告邮件
				</CfSwitch>

				<CfSwitch
					v-for="item in mailTypes"
					:key="item.type"
					:model-value="hasMailType(item.type)"
					:disabled="savingMail"
					class="_formBlock"
					@update:model-value="onMailType(item.type, $event)"
				>
					{{ item.label }}
				</CfSwitch>

				<CfInfo class="note">
					回复、提及、关注与关注请求可以同时发邮件；转发、引用与反应只在站内通知。
				</CfInfo>
				<CfInfo v-if="mailError != null" warn class="note">
					{{ mailError }}
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="_sectionTitle">{{ t("password") }}</h2>
				<CfInfo class="note">
					{{ t("_forgotPassword.enterEmail") }}
					忘记密码可以在登录页发起重置，见
					<RouterLink to="/reset-password">重置密码</RouterLink>。
				</CfInfo>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish's settings/email.vue saves on blur and asks for the password in a
// dialog. The port keeps that shape: an explicit save button, a password
// dialog, then `i/update-email`, which stores the address unverified and mails
// a link. The two notification switches in Firefish write fields Cinderfish
// does not store, so they are described instead of rendered.
import { computed, ref } from "vue";
import { me, updateMe, updateProfile } from "@/account";
import { ApiError, api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import { inputText } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsEmail" });

const draft = ref("");
const busy = ref(false);
const savingMail = ref(false);
const mailError = ref<string | null>(null);
const errorText = ref<string | null>(null);
const notice = ref<string | null>(null);

const hasEmail = computed(
	() => me.value?.email != null && me.value.email !== "",
);
const canSave = computed(() => {
	const next = draft.value.trim();
	return next !== "" && next !== (me.value?.email ?? "");
});

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function save(): Promise<void> {
	if (me.value == null || busy.value) return;
	const address = draft.value.trim();
	if (address === "") return;

	const { canceled, result } = await inputText({
		title: t("password"),
		type: "password",
		autocomplete: "current-password",
	});
	if (canceled) return;

	busy.value = true;
	errorText.value = null;
	notice.value = null;
	try {
		const fresh = await api("i/update-email", {
			password: result ?? "",
			email: address,
		});
		updateMe({ email: fresh.email, emailVerified: fresh.emailVerified });
		draft.value = "";
		notice.value = t("verificationEmailSent");
	} catch (cause) {
		errorText.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

// The four reasons the backend mails about (see
// docs/research/api-contract/auth-users.md §POST /api/i/update).
type MailType = "mention" | "reply" | "follow" | "receiveFollowRequest";

const mailTypes: { type: MailType; label: string }[] = [
	{ type: "mention", label: "被提及时" },
	{ type: "reply", label: "被回复时" },
	{ type: "follow", label: "被关注时" },
	{ type: "receiveFollowRequest", label: "收到关注请求时" },
];

function hasMailType(type: MailType): boolean {
	return (me.value?.emailNotificationTypes ?? []).includes(type);
}

async function saveMail(patch: {
	emailNotificationTypes?: string[];
	receiveAnnouncementEmail?: boolean;
}): Promise<void> {
	if (savingMail.value) return;
	savingMail.value = true;
	mailError.value = null;
	try {
		await updateProfile(patch);
	} catch (cause) {
		mailError.value = errorMessage(cause);
	} finally {
		savingMail.value = false;
	}
}

function onMailSwitch(key: "receiveAnnouncementEmail", value: boolean | null): void {
	if (value == null) return;
	void saveMail({ [key]: value });
}

function onMailType(type: MailType, value: boolean | null): void {
	if (value == null) return;
	const current = me.value?.emailNotificationTypes ?? [];
	const next = value
		? [...current.filter((t) => t !== type), type]
		: current.filter((t) => t !== type);
	void saveMail({ emailNotificationTypes: next });
}

async function resend(): Promise<void> {
	if (me.value == null || busy.value) return;
	busy.value = true;
	errorText.value = null;
	notice.value = null;
	try {
		await api("i/email-verification", { email: me.value.email ?? undefined });
		notice.value = t("verificationEmailSent");
	} catch (cause) {
		errorText.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

definePageMetadata({ title: t("email"), icon: "ph-envelope-simple-open ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-email {
	.address {
		overflow-wrap: anywhere;
	}

	.badge {
		margin-left: 8px;
		padding: 1px 8px;
		border-radius: 999px;
		background: var(--bg);
		font-size: 0.75em;
		opacity: 0.7;
	}

	.badge.verified {
		color: var(--success, #3ba55d);
		opacity: 1;
	}

	.note {
		margin-top: 12px;
	}
}
</style>
