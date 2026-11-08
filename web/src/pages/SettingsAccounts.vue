<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/accounts.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-accounts">
				<CfInfo class="_formBlock">
					浏览器会为每个账号保存一份登录凭据。切换账号会重新读取该账号的资料，
					不会影响其它账号的登录状态。
				</CfInfo>

				<CfButton primary :disabled="signingIn" @click="addOpen = !addOpen">
					<i class="ph-user-plus ph-bold"></i>
					添加账号
				</CfButton>

				<form v-if="addOpen" class="addCard" @submit.prevent="addAccount">
					<h2 class="title">用另一个账号登录</h2>
					<p class="caption _caption">
						只支持本实例账号。登录成功后，原来的账号会保留在下面的列表里。
					</p>
					<CfInput
						v-model="username"
						class="field"
						autocomplete="username"
						:spellcheck="false"
						required
					>
						<template #label>用户名</template>
						<template #prefix>@</template>
					</CfInput>
					<CfInput
						v-model="password"
						class="field"
						type="password"
						autocomplete="current-password"
						required
					>
						<template #label>密码</template>
					</CfInput>
					<CfInput
						v-if="needsToken"
						v-model="token"
						class="field"
						autocomplete="one-time-code"
						pattern="^[0-9]{6}$"
						:spellcheck="false"
						required
					>
						<template #label>两步验证码</template>
					</CfInput>
					<CfInfo v-if="addError !== null" warn class="field">
						{{ addError }}
					</CfInfo>
					<div class="actions">
						<CfButton type="submit" primary :disabled="signingIn">
							{{ signingIn ? "登录中…" : "登录并保存" }}
						</CfButton>
						<CfButton :disabled="signingIn" @click="closeAdd">
							取消
						</CfButton>
					</div>
				</form>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="entries.length === 0">
					当前浏览器没有保存任何账号凭据。
				</CfInfo>
				<ul v-else class="accounts">
					<li
						v-for="entry in entries"
						:key="entry.id"
						class="account"
						:class="{ current: entry.current }"
					>
						<MkAvatar class="avatar" :user="entry.user" />
						<div class="body">
							<div class="name">
								<MkUserName :user="entry.user" :nowrap="true" />
								<span v-if="entry.current" class="badge">当前</span>
							</div>
							<span class="acct _monospace">@{{ acct(entry.user) }}</span>
						</div>
						<div class="buttons">
							<CfButton
								v-if="!entry.current"
								mini
								:disabled="switchingTo !== null || entry.unavailable"
								@click="switchTo(entry)"
							>
								{{ switchingTo === entry.id ? "切换中…" : "切换" }}
							</CfButton>
							<CfButton
								mini
								:disabled="entry.current"
								@click="remove(entry)"
							>
								移除
							</CfButton>
						</div>
					</li>
				</ul>

				<CfInfo v-if="actionError !== null" warn class="_formBlock">
					{{ actionError }}
				</CfInfo>

				<CfInfo class="_formBlock">
					移除只删除本浏览器保存的凭据，不会注销服务器上的令牌；
					要彻底吊销某个应用的访问权限，请到<CfA class="_link" to="/settings/apps">应用</CfA>页撤销。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish resolves the stored account ids through `users/show({ userIds })`.
// Cinderfish's users/show takes one id at a time, so each stored token is
// resolved with `POST /api/i` instead: that also confirms the token is still
// valid and gives the avatar and display name in one call.
import { computed, onMounted, ref } from "vue";
import { ApiError } from "@/api/client";
import {
	fetchMe,
	me,
	signIn,
	switchAccount,
	readStoredAccounts,
	removeStoredAccount,
} from "@/account";
import type { MeDetailed, StoredAccount } from "@/account";
import MkAvatar from "@/components/global/MkAvatar.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkUserName from "@/components/global/MkUserName.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { acct } from "@/filters/user";

defineOptions({ name: "CfPageSettingsAccounts" });

interface Entry {
	id: string;
	user: MeDetailed;
	current: boolean;
	unavailable: boolean;
}

const entries = ref<Entry[]>([]);
const loading = ref(false);
const switchingTo = ref<string | null>(null);
const actionError = ref<string | null>(null);

const addOpen = ref(false);
const username = ref("");
const password = ref("");
const token = ref("");
const needsToken = ref(false);
const signingIn = ref(false);
const addError = ref<string | null>(null);

const currentId = computed(() => me.value?.id ?? null);

async function load(): Promise<void> {
	loading.value = true;
	actionError.value = null;
	const stored: StoredAccount[] = readStoredAccounts();
	const rows = await Promise.all(
		stored.map(async (account) => {
			if (currentId.value === account.id) {
				const user = me.value;
				if (user != null) {
					return { id: account.id, user, current: true, unavailable: false };
				}
			}
			try {
				const user = await fetchMe(account.token);
				return { id: account.id, user, current: false, unavailable: false };
			} catch {
				return {
					id: account.id,
					user: {
						...placeholderUser(account.id),
					} as MeDetailed,
					current: currentId.value === account.id,
					unavailable: true,
				};
			}
		}),
	);
	entries.value = rows;
	loading.value = false;
}

function placeholderUser(id: string): MeDetailed {
	return {
		id,
		username: "unknown",
		name: null,
		avatarUrl: null,
	} as unknown as MeDetailed;
}

async function switchTo(entry: Entry): Promise<void> {
	if (switchingTo.value !== null || entry.unavailable) return;
	switchingTo.value = entry.id;
	actionError.value = null;
	try {
		await switchAccount(entry.id);
		location.reload();
	} catch (error) {
		actionError.value =
			error instanceof ApiError ? error.message : String(error);
	} finally {
		switchingTo.value = null;
	}
}

function remove(entry: Entry): void {
	if (entry.current) return;
	removeStoredAccount(entry.id);
	entries.value = entries.value.filter((row) => row.id !== entry.id);
}

async function addAccount(): Promise<void> {
	if (signingIn.value) return;
	signingIn.value = true;
	addError.value = null;
	try {
		const outcome = await signIn(
			username.value.trim().replace(/^@/, ""),
			password.value,
			needsToken.value ? token.value : undefined,
		);
		if ("twoFactorRequired" in outcome) {
			needsToken.value = true;
			token.value = "";
			addError.value = "该账号已启用两步验证，请输入身份验证器应用中的 6 位验证码。";
			return;
		}
		username.value = "";
		password.value = "";
		token.value = "";
		needsToken.value = false;
		addOpen.value = false;
		await load();
		location.reload();
	} catch (error) {
		addError.value =
			error instanceof ApiError ? error.message : String(error);
	} finally {
		signingIn.value = false;
	}
}

function closeAdd(): void {
	addOpen.value = false;
	addError.value = null;
	password.value = "";
	token.value = "";
	needsToken.value = false;
}

onMounted(() => {
	void load();
});
</script>

<style lang="scss" scoped>
.addCard {
	margin: 12px 0;
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.title {
		margin: 0 0 4px;
		font-size: 1em;
	}

	.caption {
		margin: 0 0 12px;
	}

	.field {
		margin-bottom: 12px;
	}

	.actions {
		display: flex;
		gap: 8px;
	}
}

.accounts {
	list-style: none;
	margin: 12px 0 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.account {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 10px 12px;
	background: var(--panel);
	border-radius: var(--radius);

	&.current {
		outline: 2px solid var(--accent, #86b30044);
	}

	.avatar {
		width: 44px;
		height: 44px;
		flex: none;
	}

	.body {
		flex: 1;
		min-width: 0;
	}

	.name {
		display: flex;
		align-items: center;
		gap: 8px;
		font-weight: 600;
	}

	.badge {
		padding: 1px 6px;
		border-radius: 999px;
		background: var(--accented-bg, rgba(0, 0, 0, 0.08));
		font-size: 0.72em;
		font-weight: 400;
		opacity: 0.8;
	}

	.acct {
		font-size: 0.82em;
		color: var(--fgTransparentWeak);
	}

	.buttons {
		display: flex;
		gap: 6px;
		flex: none;
	}
}

@media (max-width: 520px) {
	.account {
		flex-wrap: wrap;

		.buttons {
			width: 100%;
		}
	}
}
</style>
