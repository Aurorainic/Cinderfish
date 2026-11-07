<!-- Ported from misskey@4ff2edf:packages/frontend/src/pages/auth.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="600" :margin-min="16" :margin-max="32">
		<div class="cf-auth">
			<div v-if="state === 'error'" class="section">
				<CfInfo warn>{{ t("somethingHappened") }}</CfInfo>
			</div>

			<div v-else-if="!signedIn" class="section">
				<p class="lead">{{ t("_auth.pleaseLogin") }}</p>
				<CfSignin />
			</div>

			<div v-else-if="session == null" class="section loading">
				<CfLoading />
			</div>

			<template v-else>
				<div v-if="state === 'denied'" class="section">
					<h2 class="title">{{ t("_auth.denied") }}</h2>
				</div>

				<div v-else-if="state === 'accepted'" class="section">
					<h2 class="title">
						{{
							session.app.isAuthorized
								? t("_auth.alreadyAuthorized")
								: t("_auth.accepted")
						}}
					</h2>
					<p v-if="session.app.callbackUrl" class="lead">
						{{ t("_auth.callback") }}…
					</p>
					<p v-else class="lead">{{ t("_auth.pleaseGoBack") }}</p>
				</div>

				<div v-else class="section">
					<h2 class="title">
						{{ t("_auth.shareAccess", { name: session.app.name }) }}
					</h2>
					<ul v-if="scopes.length > 0" class="permissions">
						<li v-for="scope in scopes" :key="scope" class="permission">
							<i class="ph-check ph-bold"></i>
							{{ scopeLabel(scope) }}
						</li>
					</ul>
					<CfInfo v-if="errorText != null" warn class="note">{{
						errorText
					}}</CfInfo>
					<div class="footer">
						<button class="_button border" type="button" @click="deny">
							{{ t("cancel") }}
						</button>
						<button
							class="_button primary"
							type="button"
							:disabled="busy"
							@click="accept"
						>
							{{ t("accept") }}
						</button>
					</div>
				</div>
			</template>
		</div>
	</MkSpacer>
</template>

<script lang="ts" setup>
// The approval page of the app/session flow: the client registered with
// app/create, called auth/session/generate and sent the user here with the
// session token. Accepting calls auth/accept, which issues the access token
// the client collects through auth/session/userkey.
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import CfSignin from "@/account/components/CfSignin.vue";
import { signedIn } from "@/account/me";
import { ApiError, api } from "@/api/client";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAuth" });

interface AuthSessionApp {
	id: string;
	name: string;
	callbackUrl: string | null;
	permission: string[];
	isAuthorized: boolean;
}

interface AuthSession {
	id: string;
	app: AuthSessionApp;
	token: string;
}

const route = useRoute();
const token = computed(() => {
	const raw = route.params.token;
	return typeof raw === "string" ? raw : "";
});

const session = ref<AuthSession | null>(null);
const state = ref<"idle" | "accepted" | "denied" | "error">("idle");
const busy = ref(false);
const errorText = ref<string | null>(null);

const scopes = computed<string[]>(() =>
	(session.value?.app.permission ?? []).filter(
		(p): p is string => typeof p === "string",
	),
);

const scopeLabels: Record<string, string> = {
	"read:account": "读取账号信息",
	"write:account": "修改账号信息",
	"read:notes": "读取笔记",
	"write:notes": "发布与删除笔记",
	"read:following": "读取关注关系",
	"write:following": "关注与取消关注",
	"read:drive": "读取云盘",
	"write:drive": "上传与删除文件",
};

function scopeLabel(scope: string): string {
	return scopeLabels[scope] ?? scope;
}

function safeCallback(url: string | null): string | null {
	if (url == null) return null;
	try {
		const parsed = new URL(url);
		if (
			["javascript:", "file:", "data:", "mailto:", "tel:", "vbscript:"].includes(
				parsed.protocol,
			)
		)
			return null;
		return url;
	} catch {
		return null;
	}
}

function accepted(): void {
	state.value = "accepted";
	const target = safeCallback(session.value?.app.callbackUrl ?? null);
	if (target != null && session.value != null) {
		const separator = target.includes("?") ? "&" : "?";
		window.location.assign(`${target}${separator}token=${session.value.token}`);
	}
}

function deny(): void {
	state.value = "denied";
}

async function accept(): Promise<void> {
	if (busy.value || session.value == null) return;
	busy.value = true;
	errorText.value = null;
	try {
		await api("auth/accept", { token: session.value.token });
		accepted();
	} catch (cause) {
		errorText.value =
			cause instanceof ApiError ? cause.message : t("somethingHappened");
	} finally {
		busy.value = false;
	}
}

onMounted(async () => {
	if (!signedIn.value) return;
	try {
		const result = await api("auth/session/show", { token: token.value });
		session.value = {
			id: result.id,
			token: result.token,
			app: result.app,
		};
		// An app that already holds a token is approved without asking again,
		// which is Misskey's behaviour.
		if (result.app.isAuthorized) {
			await api("auth/accept", { token: result.token });
			accepted();
		}
	} catch {
		state.value = "error";
	}
});

definePageMetadata({
	title: "应用授权",
	icon: "ph-app-window ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-auth {
	max-width: 600px;
	margin: 0 auto;

	.section {
		padding: 24px;
		border-radius: var(--radius);
		background: var(--panel);

		@media (max-width: 500px) {
			padding: 16px;
		}
	}

	.title {
		margin: 0 0 16px;
		font-size: 1.15em;
	}

	.lead {
		margin: 0 0 12px;
		opacity: 0.85;
	}

	.permissions {
		list-style: none;
		margin: 0 0 16px;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.permission {
		display: flex;
		align-items: center;
		gap: 8px;

		> i {
			color: var(--success);
		}
	}

	.note {
		margin-bottom: 12px;
	}

	.footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
	}

	.loading {
		display: flex;
		justify-content: center;
	}
}
</style>
