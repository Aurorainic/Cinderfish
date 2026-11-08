<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/apps.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-apps">
				<CfInfo class="_formBlock">
					这里列出通过 MiAuth 或 OAuth 授权过的第三方应用。撤销后，该应用手中的令牌立即失效，
					需要重新授权才能继续访问你的账号。当前浏览器会话所用的令牌不在列表里，退出登录才会失效。
				</CfInfo>

				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<CfInfo v-else-if="apps.length === 0">
					还没有授权过任何应用。在第三方客户端里登录并确认权限后，应用会出现在这里。
				</CfInfo>
				<div v-else class="apps">
					<article v-for="app in apps" :key="app.id" class="app">
						<img
							v-if="app.iconUrl !== null"
							class="icon"
							:src="resolveMediaUrl(app.iconUrl)"
							alt=""
						/>
						<div v-else class="icon placeholder">
							<i class="ph-app-window ph-bold ph-lg"></i>
						</div>
						<div class="body">
							<h2 class="name">{{ app.name || "未命名应用" }}</h2>
							<p v-if="app.description !== ''" class="description">
								{{ app.description }}
							</p>
							<dl class="facts">
								<div>
									<dt>授权时间</dt>
									<dd><MkTime :time="app.createdAt" mode="absolute" /></dd>
								</div>
								<div>
									<dt>最近使用</dt>
									<dd>
										<MkTime
											v-if="app.lastUsedAt !== null"
											:time="app.lastUsedAt"
											mode="absolute"
										/>
										<span v-else>从未使用</span>
									</dd>
								</div>
							</dl>
							<details v-if="app.permission.length > 0" class="permissions">
								<summary>权限（{{ app.permission.length }}）</summary>
								<ul>
									<li
										v-for="scope in app.permission"
										:key="scope"
										class="_monospace"
									>
										{{ scope }}
									</li>
								</ul>
							</details>
						</div>
						<CfButton
							class="revoke"
							:disabled="revokingId === app.id"
							@click="revoke(app)"
						>
							{{ revokingId === app.id ? "撤销中…" : "撤销" }}
						</CfButton>
					</article>
				</div>

				<CfInfo v-if="actionError !== null" warn class="_formBlock">
					{{ actionError }}
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Upstream renders `i/apps` through `MkPagination` and revokes with
// `i/revoke-token`. The backend returns an unpaginated array, so this page
// loads it once and reloads after a revoke instead of carrying a pagination
// component that would only ever see one page.
import { onMounted, ref } from "vue";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
import type { AuthorizedApp } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkError from "@/components/global/MkError.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { confirm } from "@/os";

defineOptions({ name: "CfPageSettingsApps" });

const apps = ref<AuthorizedApp[]>([]);
const loading = ref(false);
const loadError = ref<string | null>(null);
const actionError = ref<string | null>(null);
const revokingId = ref<string | null>(null);

function errorText(error: unknown): string {
	if (error instanceof ApiError) return error.message;
	return String(error);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		apps.value = await api("i/apps", { sort: "+lastUsedAt" });
	} catch (error) {
		loadError.value = errorText(error);
	} finally {
		loading.value = false;
	}
}

async function revoke(app: AuthorizedApp): Promise<void> {
	if (revokingId.value !== null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `撤销「${app.name || "未命名应用"}」后，它需要重新授权才能访问你的账号。确定撤销吗？`,
	});
	if (canceled) return;
	revokingId.value = app.id;
	actionError.value = null;
	try {
		await api("i/revoke-token", { tokenId: app.id });
		apps.value = apps.value.filter((item) => item.id !== app.id);
	} catch (error) {
		actionError.value = errorText(error);
	} finally {
		revokingId.value = null;
	}
}

onMounted(() => {
	void load();
});
</script>

<style lang="scss" scoped>
.apps {
	display: flex;
	flex-direction: column;
	gap: 12px;
}

.app {
	display: flex;
	align-items: flex-start;
	gap: 12px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.icon {
		width: 44px;
		height: 44px;
		border-radius: 8px;
		object-fit: cover;
		flex: none;
	}

	.icon.placeholder {
		display: grid;
		place-items: center;
		background: var(--bg);
		opacity: 0.7;
	}

	.body {
		flex: 1;
		min-width: 0;
	}

	.name {
		margin: 0;
		font-size: 1em;
	}

	.description {
		margin: 4px 0 0;
		font-size: 0.88em;
		color: var(--fgTransparentWeak);
		overflow-wrap: anywhere;
	}

	.facts {
		display: flex;
		flex-wrap: wrap;
		gap: 6px 20px;
		margin: 8px 0 0;
		font-size: 0.82em;

		dt,
		dd {
			display: inline;
			margin: 0;
		}

		dt {
			color: var(--fgTransparentWeak);
			margin-right: 6px;
		}
	}

	.permissions {
		margin-top: 8px;
		font-size: 0.82em;

		summary {
			cursor: pointer;
			color: var(--fgTransparentWeak);
		}

		ul {
			margin: 6px 0 0;
			padding-left: 20px;
			color: var(--fgTransparentWeak);
		}
	}

	.revoke {
		flex: none;
	}
}

@media (max-width: 520px) {
	.app {
		flex-wrap: wrap;
	}

	.revoke {
		width: 100%;
	}
}
</style>
