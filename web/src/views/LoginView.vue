// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkSignin.vue
// (form structure) and pages/welcome.entrance.a.vue (card panel look).
//
// Dropped versus the original: passwordless login, security keys, TOTP,
// password reset (Cinderfish has no mailer), the avatar preview and the
// host suffix (single-instance). The session store and error-id mapping are
// Cinderfish's own and stay unchanged.
<template>
	<form class="signin" @submit.prevent="submit">
		<MkInfo v-if="error" warn>{{ error }}</MkInfo>
		<div class="normal-signin">
			<MkInput
				v-model="username"
				class="block"
				:placeholder="'用户名'"
				type="text"
				pattern="^[a-zA-Z0-9_]+$"
				:spellcheck="false"
				autocomplete="username"
				required
				data-cy-signin-username
			>
				<template #prefix>@</template>
			</MkInput>
			<MkInput
				v-model="password"
				class="block"
				:placeholder="'密码'"
				type="password"
				autocomplete="current-password"
				required
				data-cy-signin-password
			>
				<template #prefix>#</template>
			</MkInput>
			<MkButton
				class="block submit"
				type="submit"
				primary
				rounded
				:disabled="busy"
				>{{ busy ? "登录中…" : "登录" }}</MkButton
			>
		</div>
		<p class="switch">
			还没有账号？<MkA to="/signup">注册</MkA>
		</p>
	</form>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import MkA from "../components/ui/MkA.vue";
import MkButton from "../components/ui/MkButton.vue";
import MkInput from "../components/ui/MkInput.vue";
import MkInfo from "../components/ui/MkInfo.vue";
import { ApiError } from "../api/client";
import { signin } from "../stores/session";
import { usePageTitle } from "../composables/use-page-title";

const router = useRouter();
const route = useRoute();

const username = ref("");
const password = ref("");
const error = ref("");
const busy = ref(false);

usePageTitle("登录");

// MkSignin.vue's loginFailed(): map the server's error ids to concrete copy.
// The backend sends id-only errors, so err.message would just be the HTTP
// status text.
const LOGIN_ERROR_COPY: Record<string, string> = {
	"6cc579cc-885d-43d8-95c2-b8c7fc963280": "用户不存在。",
	"932c904e-9460-45b7-9ce6-7ed33be7eb2c": "密码不正确。",
	"e03a5f46-d309-4865-9b69-56282d94e1eb": "该账号已被停用。",
	"22d05606-fbcf-421a-a2db-b32610dcfd1b": "尝试次数过多，请稍后再试。",
};

function loginErrorText(err: unknown): string {
	if (err instanceof ApiError) {
		const mapped = LOGIN_ERROR_COPY[err.id];
		if (mapped) return mapped;
	}
	return "登录失败，请稍后再试。";
}

async function submit() {
	if (busy.value) return;
	error.value = "";
	if (!username.value.trim() || !password.value) {
		error.value = "请输入用户名和密码。";
		return;
	}
	busy.value = true;
	try {
		await signin(username.value.trim(), password.value);
		const redirect =
			typeof route.query.redirect === "string" ? route.query.redirect : "/";
		await router.push(redirect);
	} catch (err) {
		error.value = loginErrorText(err);
	} finally {
		busy.value = false;
	}
}
</script>

<style scoped>
.signin > * {
	margin-block-end: 16px;
}

.signin .block {
	margin-block: 0 16px;
	margin-inline: 0;
}

.signin .submit {
	margin-block: 16px 0;
	margin-inline: auto;
}

.signin .switch {
	margin: 16px 0 0;
	font-size: 0.9em;
	text-align: center;
	color: var(--fgTransparentWeak);
}

.signin .switch a {
	color: var(--accent);
	font-weight: bold;
	text-decoration: none;
}
</style>
