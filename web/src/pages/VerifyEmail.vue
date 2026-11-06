<!-- Ported from firefish@b425f61:packages/client/src/pages/verify-email.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-verify-email">
		<CfLoading v-if="state === 'working'" />
		<template v-else>
			<CfInfo v-if="state === 'done'">
				<i class="ph-check-circle ph-bold ph-lg"></i>
				电子邮件地址已验证。
			</CfInfo>
			<CfInfo v-else-if="state === 'signed-out'" warn>
				{{ t("signinRequired") }}
				验证链接与账号绑定，请先用收到邮件的账号登录，再重新打开链接。
			</CfInfo>
			<CfInfo v-else warn>{{ errorText }}</CfInfo>

			<div class="actions">
				<CfButton v-if="state === 'signed-out'" @click="goSignin()">
					去登录
				</CfButton>
				<CfButton primary @click="goHome()">{{ t("ok") }}</CfButton>
			</div>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish opens this page from the mail link and POSTs the code. The code is
// bound to the account that requested it, so the page needs a session: a
// visitor without one is told to sign in instead of losing the code.
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { me } from "@/account";
import { ApiError, api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import { updateMe } from "@/account";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageVerifyEmail" });

type State = "working" | "done" | "failed" | "signed-out";

const route = useRoute();
const router = useRouter();
const state = ref<State>("working");
const errorText = ref<string>("验证链接无效或已过期。");

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function verify(): Promise<void> {
	const code = String(route.params.code ?? "");
	if (me.value == null) {
		state.value = "signed-out";
		return;
	}
	if (code === "") {
		state.value = "failed";
		errorText.value = "链接缺少验证码，请重新打开邮件里的链接。";
		return;
	}
	try {
		await api("verify-email", { code });
		updateMe({ emailVerified: true });
		state.value = "done";
	} catch (cause) {
		errorText.value = errorMessage(cause);
		state.value = "failed";
	}
}

function goHome(): void {
	void router.push("/");
}

function goSignin(): void {
	void router.push("/");
}

onMounted(() => {
	void verify();
});

definePageMetadata({ title: "Verify email", icon: "ph-envelope-simple ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-verify-email {
	max-width: 520px;
	margin: 40px auto;
	padding: 0 16px;

	.actions {
		display: flex;
		gap: 8px;
		margin-top: 16px;
	}
}
</style>
