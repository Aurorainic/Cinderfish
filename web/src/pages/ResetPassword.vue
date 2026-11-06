<!-- Ported from firefish@b425f61:packages/client/src/pages/reset-password.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-reset-password">
		<h1 class="pageTitle">{{ t("resetPassword") }}</h1>

		<template v-if="token != null">
			<p class="lead">设置一个新密码。设置成功后旧的密码立即失效。</p>
			<CfInput
				v-model="password"
				class="field"
				type="password"
				autocomplete="new-password"
			>
				<template #label>{{ t("newPassword") }}</template>
			</CfInput>
			<CfInput
				v-model="password2"
				class="field"
				type="password"
				autocomplete="new-password"
			>
				<template #label>{{ t("newPasswordRetype") }}</template>
			</CfInput>
			<div class="actions">
				<CfButton primary :disabled="busy || !canSubmit" @click="submit()">
					{{ t("resetPassword") }}
				</CfButton>
			</div>
			<CfInfo v-if="errorText != null" warn class="note">{{ errorText }}</CfInfo>
		</template>

		<template v-else>
			<p class="lead">
				{{ t("_forgotPassword.enterEmail") }}
			</p>
			<CfInput v-model="username" class="field" autocomplete="username">
				<template #label>{{ t("username") }}</template>
			</CfInput>
			<CfInput v-model="email" class="field" type="email" autocomplete="email">
				<template #label>{{ t("emailAddress") }}</template>
			</CfInput>
			<div class="actions">
				<CfButton primary :disabled="busy || !canRequest" @click="request()">
					<i class="ph-paper-plane-tilt ph-bold ph-lg"></i>
					发送重置链接
				</CfButton>
				<CfButton @click="goHome">{{ t("cancel") }}</CfButton>
			</div>
			<CfInfo v-if="errorText != null" warn class="note">{{ errorText }}</CfInfo>
			<CfInfo v-if="notice != null" class="note">{{ notice }}</CfInfo>
			<p class="hint">{{ t("_forgotPassword.ifNoEmail") }}</p>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish renders two modes from one route: with a token it sets the new
// password, without one it opens the "forgot password" dialog. The dialog is
// inlined here as a form so the flow works without a session.
//
// `request-reset-password` always answers 204, so the page must not promise
// that a mail was sent — the notice says "if the details match".
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ApiError, api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageResetPassword" });

const route = useRoute();
const router = useRouter();

const token = computed<string | null>(() => {
	const raw = route.params.token;
	return typeof raw === "string" && raw !== "" ? raw : null;
});

const password = ref("");
const password2 = ref("");
const username = ref("");
const email = ref("");
const busy = ref(false);
const errorText = ref<string | null>(null);
const notice = ref<string | null>(null);

const canSubmit = computed(
	() => password.value !== "" && password.value === password2.value,
);
const canRequest = computed(
	() => username.value.trim() !== "" && email.value.trim() !== "",
);

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function submit(): Promise<void> {
	if (!canSubmit.value || busy.value || token.value == null) return;
	if (password.value !== password2.value) {
		errorText.value = t("passwordNotMatched");
		return;
	}
	busy.value = true;
	errorText.value = null;
	try {
		await api("reset-password", { token: token.value, password: password.value });
		await router.push("/");
	} catch (cause) {
		errorText.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

async function request(): Promise<void> {
	if (!canRequest.value || busy.value) return;
	busy.value = true;
	errorText.value = null;
	notice.value = null;
	try {
		await api("request-reset-password", {
			username: username.value.trim().replace(/^@/, ""),
			email: email.value.trim(),
		});
		notice.value =
			"如果用户名与邮箱匹配，重置链接已经发出，请检查邮箱（也包括垃圾邮件箱）。";
	} catch (cause) {
		errorText.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

function goHome(): void {
	void router.push("/");
}

definePageMetadata({ title: t("resetPassword"), icon: "ph-lock-key ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-reset-password {
	max-width: 480px;
	margin: 40px auto;
	padding: 0 16px;

	.pageTitle {
		margin: 0 0 12px;
		font-size: 1.25em;
	}

	.lead {
		margin: 0 0 16px;
		font-size: 0.9em;
		opacity: 0.8;
	}

	.field {
		margin-bottom: 12px;
	}

	.actions {
		display: flex;
		gap: 8px;
		margin-top: 4px;
	}

	.note {
		margin-top: 12px;
	}

	.hint {
		margin-top: 16px;
		font-size: 0.8em;
		opacity: 0.6;
	}
}
</style>
