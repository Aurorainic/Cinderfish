<!-- Ported from firefish@b425f61:packages/client/src/pages/oauth.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
		<div class="cf-oauth">
			<div v-if="paramError != null" class="section">
				<CfInfo warn>{{ paramError }}</CfInfo>
			</div>

			<div v-else-if="!signedIn" class="section">
				<h2 class="title">登录后授权</h2>
				<CfSignin />
			</div>

			<div v-else-if="internalState === 'denied'" class="section">
				<h2 class="title">{{ t("_auth.denied") }}</h2>
				<p v-if="clientHost != null" class="description">
					应用在 {{ clientHost }}，你可以关闭这个页面。
				</p>
			</div>

			<div v-else-if="internalState === 'accepted'" class="section">
				<h2 class="title">已授权</h2>
				<p class="description">{{ t("_auth.callback") }}…</p>
			</div>

			<div v-else class="section">
				<h2 class="title">
					{{ t("_auth.shareAccess", { name: clientName }) }}
				</h2>
				<p class="description">{{ t("_auth.permissionAsk") }}</p>
				<ul class="permissions">
					<li v-for="scope in scopes" :key="scope" class="permission">
						<i :class="`ph-${iconFor(scope)} ph-bold ph-xl`"></i>
						{{ scopeLabel(scope) }}
					</li>
				</ul>
				<p v-if="clientHost != null" class="description">
					授权后浏览器会跳回 <code>{{ clientHost }}</code
					>。
				</p>
				<div v-if="errorText != null" class="error" role="alert">
					{{ errorText }}
				</div>
				<div class="footer">
					<button class="_button border" type="button" @click="deny">
						{{ t("cancel") }}
					</button>
					<button
						class="_button primary"
						type="button"
						:disabled="internalState === 'waiting'"
						@click="accept"
					>
						{{ t("accept") }}
					</button>
				</div>
			</div>
		</div>
	</MkSpacer>
</template>

<script lang="ts" setup>
// /oauth/authorize is served by the backend, which hands the browser the SPA
// when the request carries no credential; this page is that consent screen.
// Approving calls POST /api/oauth/authorize with the signed-in token and then
// navigates to the redirect the server builds (code + state + iss).
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import CfSignin from "@/account/components/CfSignin.vue";
import { signedIn } from "@/account/me";
import { permissionIcon } from "@/account/miauth";
import { ApiError, api } from "@/api/client";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageOAuth" });

const route = useRoute();

function first(value: unknown): string {
	if (typeof value === "string") return value;
	if (Array.isArray(value) && typeof value[0] === "string") return value[0];
	return "";
}

const clientID = computed(() => first(route.query.client_id));
const redirectURI = computed(() => first(route.query.redirect_uri));
const responseType = computed(() => first(route.query.response_type));
const codeChallenge = computed(() => first(route.query.code_challenge));
const codeChallengeMethod = computed(() =>
	first(route.query.code_challenge_method),
);
const internalState = ref<"idle" | "waiting" | "accepted" | "denied">("idle");
const errorText = ref<string | null>(null);

const scopes = computed<string[]>(() => {
	const raw = route.query.scope;
	const list: string[] = [];
	for (const value of Array.isArray(raw) ? raw : raw == null ? [] : [raw]) {
		if (typeof value !== "string") continue;
		for (const scope of value.split(/\s+/)) {
			if (scope !== "") list.push(scope);
		}
	}
	return list.length > 0 ? list : ["read"];
});

const paramError = computed<string | null>(() => {
	if (clientID.value === "") return "授权链接缺少 client_id。";
	if (redirectURI.value === "") return "授权链接缺少 redirect_uri。";
	if (responseType.value !== "code") return "只支持 code 授权方式。";
	if (codeChallenge.value === "") return "授权链接缺少 code_challenge（需要 PKCE）。";
	if (codeChallengeMethod.value !== "S256")
		return "code_challenge_method 必须是 S256。";
	return null;
});

const clientName = computed(() => clientHost.value ?? clientID.value);
const clientHost = computed<string | null>(() => {
	try {
		return new URL(clientID.value).host;
	} catch {
		return null;
	}
});

const scopeLabels: Record<string, string> = {
	read: "读取账号与内容",
	write: "以你的名义发帖、修改资料",
	follow: "关注与取消关注",
	upload: "上传文件到云盘",
};

function scopeLabel(scope: string): string {
	return scopeLabels[scope] ?? scope;
}

function iconFor(scope: string): string {
	return scope === "read" ? "eye" : permissionIcon(scope);
}

function deny(): void {
	internalState.value = "denied";
}

async function accept(): Promise<void> {
	if (internalState.value === "waiting") return;
	internalState.value = "waiting";
	errorText.value = null;
	try {
		const result = await api("oauth/authorize", {
			client_id: clientID.value,
			redirect_uri: redirectURI.value,
			response_type: "code",
			code_challenge: codeChallenge.value,
			code_challenge_method: "S256",
			state: first(route.query.state),
			scope: scopes.value,
		});
		internalState.value = "accepted";
		window.location.assign(result.redirect);
	} catch (cause) {
		internalState.value = "idle";
		errorText.value =
			cause instanceof ApiError ? cause.message : t("somethingHappened");
	}
}

definePageMetadata({
	title: "应用授权",
	icon: "ph-shield-check ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-oauth {
	max-width: 800px;
	margin: 0 auto;

	.section {
		padding: 32px;
		border-radius: var(--radius);
		background: var(--panel);

		@media (max-width: 500px) {
			padding: 16px;
		}
	}

	.title {
		margin: 0 0 16px;
		font-size: 1.2em;
	}

	.description {
		margin: 0 0 12px;
		color: var(--fgTransparentWeak);
	}

	.permissions {
		list-style: none;
		margin: 0 0 16px;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.permission {
		display: flex;
		align-items: center;
		gap: 10px;

		> i {
			color: var(--accent);
		}
	}

	.error {
		color: var(--error);
		margin-bottom: 12px;
	}

	.footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}
}
</style>
