// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/signup.vue and
// components/MkSignupDialog.vue (form structure), welcome.entrance.a.vue
// (card panel look).
//
// Dropped versus the original: email confirmation, captcha, ToS checkbox and
// the available-username poll fallback UI — Cinderfish has no mailer/captcha
// and exposes username/available only. The session store, the availability
// check and the 503 (admin-only registration) error mapping are Cinderfish's
// own and stay unchanged.
<template>
	<form class="signup" @submit.prevent="submit">
		<MkInfo v-if="instance.meta?.disableRegistration">
			本实例未开放自助注册。如果管理员已发放邀请码，请在下方填写。
		</MkInfo>
		<div class="fields">
			<MkInput
				v-model="username"
				class="block"
				:placeholder="'用户名'"
				type="text"
				pattern="^[a-zA-Z0-9_]+$"
				:spellcheck="false"
				autocomplete="username"
				required
				data-cy-signup-username
				@change="scheduleAvailabilityCheck"
			>
				<template #prefix>@</template>
				<template #suffix>@{{ host }}</template>
			</MkInput>
			<MkInput
				v-model="password"
				class="block"
				:placeholder="'至少 8 位'"
				type="password"
				autocomplete="new-password"
				required
				data-cy-signup-password
			>
				<template #prefix>#</template>
				<template #caption>
					<span
						v-if="availability === 'free'"
						class="hint ok"
					>用户名可用</span>
					<span
						v-else-if="availability === 'taken'"
						class="hint bad"
					>用户名已被占用</span>
					<span
						v-else-if="availability === 'checking'"
						class="hint"
					>检查中…</span>
				</template>
			</MkInput>
			<MkInput
				v-model="invitationCode"
				class="block"
				:placeholder="'邀请码（可选）'"
				type="text"
				:spellcheck="false"
			/>
			<MkInfo v-if="error" warn>{{ error }}</MkInfo>
			<MkButton
				class="block submit"
				type="submit"
				primary
				rounded
				:disabled="busy || closed"
				>{{ busy ? "创建中…" : "注册" }}</MkButton
			>
		</div>
		<p class="switch">
			已有账号？<MkA to="/login">登录</MkA>
		</p>
	</form>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import MkA from "../components/ui/MkA.vue";
import MkButton from "../components/ui/MkButton.vue";
import MkInput from "../components/ui/MkInput.vue";
import MkInfo from "../components/ui/MkInfo.vue";
import { usePageTitle } from "../composables/use-page-title";
import { instance, loadInstance } from "../stores/instance";
import { signup } from "../stores/session";

const router = useRouter();

const username = ref("");
const password = ref("");
const invitationCode = ref("");
const error = ref("");
const busy = ref(false);
const availability = ref<"unknown" | "checking" | "free" | "taken">("unknown");

// Cinderfish is a single-host instance; Firefish shows the local host as the
// username suffix.
const host = typeof window !== "undefined" ? window.location.hostname : "";

const closed = computed(
	() =>
		instance.meta !== null &&
		instance.meta.disableRegistration &&
		!invitationCode.value,
);

usePageTitle("创建账号");

let checkTimer: ReturnType<typeof setTimeout> | undefined;

function scheduleAvailabilityCheck() {
	availability.value = "unknown";
	clearTimeout(checkTimer);
	const name = username.value.trim();
	if (name.length < 2) return;
	checkTimer = setTimeout(async () => {
		availability.value = "checking";
		try {
			const res = await api.usernameAvailable(name);
			availability.value = res.available ? "free" : "taken";
		} catch {
			availability.value = "unknown";
		}
	}, 400);
}

async function submit() {
	if (busy.value) return;
	error.value = "";
	const name = username.value.trim();
	if (!name || !password.value) {
		error.value = "请输入用户名和密码。";
		return;
	}
	if (availability.value === "taken") {
		error.value = "这个用户名已被占用。";
		return;
	}
	busy.value = true;
	try {
		await signup(name, password.value, invitationCode.value.trim());
		await router.push("/");
	} catch (err) {
		// The backend has no email field and no captcha; a 503 here means the
		// admin registration mode (Firefish shows its "closed" copy for this).
		if (err instanceof ApiError && err.status === 503) {
			error.value = "本实例仅限管理员创建账号。";
		} else {
			error.value = "注册失败，请检查输入后重试。";
		}
	} finally {
		busy.value = false;
	}
}

onMounted(loadInstance);
</script>

<style scoped>
.signup > * {
	margin-block-end: 16px;
}

.signup .block {
	margin-block: 0 16px;
	margin-inline: 0;
}

.signup .submit {
	margin-block: 16px 0;
	margin-inline: auto;
}

.signup .hint {
	color: var(--fgTransparentWeak);
}

.signup .hint.ok {
	color: var(--accent);
}

.signup .hint.bad {
	color: #eb6f92;
}

.signup .switch {
	margin: 16px 0 0;
	font-size: 0.9em;
	text-align: center;
	color: var(--fgTransparentWeak);
}

.signup .switch a {
	color: var(--accent);
	font-weight: bold;
	text-decoration: none;
}
</style>
