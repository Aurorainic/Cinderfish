<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/overview.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="1000" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-overview">
				<CfInfo warn>
					需要管理员权限，无法查看概况。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-overview">
				<CfLoading v-if="loading" />
				<template v-else>
					<div class="grid">
						<section class="card _panel">
							<h2 class="card-title">
								<i class="ph-info ph-bold"></i>
								实例
							</h2>
							<CfInfo v-if="metaError != null" warn class="error">
								无法读取实例元数据：{{ describeError(metaError) }}
								<CfButton inline class="retry" @click="load">
									{{ t("retry") }}
								</CfButton>
							</CfInfo>
							<dl v-else-if="meta != null" class="facts">
								<div class="fact">
									<dt>{{ t("instanceName") }}</dt>
									<dd>{{ meta.name }}</dd>
								</div>
								<div class="fact">
									<dt>{{ t("version") }}</dt>
									<dd class="_monospace">{{ meta.version }}</dd>
								</div>
								<div class="fact">
									<dt>{{ t("description") }}</dt>
									<dd
										:class="{
											unset:
												meta.description == null ||
												meta.description === '',
										}"
									>
										{{ meta.description || "未设置" }}
									</dd>
								</div>
							</dl>
						</section>

						<section class="card _panel">
							<h2 class="card-title">
								<i class="ph-planet ph-bold"></i>
								{{ t("federation") }}
							</h2>
							<CfInfo v-if="modeError != null" warn class="error">
								无法读取联邦模式：{{ describeError(modeError) }}
								<CfButton inline class="retry" @click="load">
									{{ t("retry") }}
								</CfButton>
							</CfInfo>
							<dl v-else class="facts">
								<div class="fact">
									<dt>当前模式</dt>
									<dd>
										<span v-if="mode != null" class="mode">{{
											modeLabel
										}}</span>
										<span v-else class="unset">读取中…</span>
										<code v-if="mode != null" class="code">{{
											mode
										}}</code>
									</dd>
								</div>
								<div class="fact">
									<dt>白名单主机</dt>
									<dd v-if="mode === 'allowlist'">
										<template v-if="allowlistError != null">
											<span class="errorText"
												>读取失败：{{
													describeError(allowlistError)
												}}</span
											>
										</template>
										<template v-else-if="allowlistCount == null">
											读取中…
										</template>
										<template v-else>
											{{ allowlistCount }} 个主机
										</template>
									</dd>
									<dd v-else class="unset">未启用</dd>
								</div>
							</dl>
							<CfA class="card-link" to="/admin/federation">
								管理联邦设置
								<i class="ph-caret-right ph-bold"></i>
							</CfA>
						</section>

						<section class="card _panel users-card">
							<h2 class="card-title">
								<i class="ph-users ph-bold"></i>
								用户列表（前 5 位）
							</h2>
							<CfInfo v-if="usersError != null" warn class="error">
								无法读取用户列表：{{ describeError(usersError) }}
								<CfButton inline class="retry" @click="load">
									{{ t("retry") }}
								</CfButton>
							</CfInfo>
							<CfInfo
								v-else-if="users.length === 0"
								warn
								class="error"
							>
								这个实例还没有本地用户。
							</CfInfo>
							<div v-else class="user-list">
								<CfA
									v-for="user in users"
									:key="user.id"
									class="user"
									:to="userPage(user)"
								>
									<MkAvatar
										class="avatar"
										:user="user"
										:disable-link="true"
									/>
									<div class="user-body">
										<div class="user-name">
											<MkUserName
												:user="user"
												:nowrap="true"
											/>
											<span
												v-if="user.isAdmin"
												class="badge admin"
												>{{ t("administrator") }}</span
											>
											<span
												v-if="user.isSuspended"
												class="badge suspended"
												>已冻结</span
											>
										</div>
										<div class="user-sub">
											<span class="_monospace"
												>@{{ acct(user) }}</span
											>
											<span class="sep">·</span>
											<MkTime
												:time="user.createdAt"
												mode="absolute"
											/>
										</div>
									</div>
								</CfA>
							</div>
							<p class="caption _caption">
								后端按「管理员优先、注册时间升序」返回，没有总数与排序参数，
								所以这里是列表的前 5 位，不是「最新 5 位」。
							</p>
							<CfA class="card-link" to="/admin/users">
								用户管理
								<i class="ph-caret-right ph-bold"></i>
							</CfA>
						</section>
					</div>

					<CfInfo class="notice">
						Firefish 概览里的统计图表、活跃用户、热力图、队列、AP
						请求和实例统计都依赖本实例没有的端点（见第四波施工契约 §2），
						这里只展示 <code>meta</code>、<code>admin/federation/mode</code>、
						<code>admin/federation/allowlist</code> 与
						<code>admin/show-users</code> 实际返回的值。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's dashboard is a grid of `MkFolder` widgets fed by charts, queues,
// federation stats and admin/server-info. None of those endpoints exist on
// Cinderfish, so each widget is replaced by a fact card that reads one of the
// four sources the backend really serves. Nothing is estimated: when a call
// fails the card says so and offers a retry instead of showing a blank.
import { computed, onMounted, ref } from "vue";
import { ApiError, api, apiGet } from "@/api/client";
import type {
	AllowlistResponse,
	FederationModeResponse,
	MetaLite,
	UserDetailed,
} from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

type FederationMode = FederationModeResponse["mode"];

const MODE_LABELS: Record<FederationMode, string> = {
	public: "公开",
	allowlist: "白名单",
	none: "关闭",
};

const admin = computed<boolean>(() => isAdmin());

const meta = ref<MetaLite | null>(null);
const metaError = ref<unknown>(null);

const mode = ref<FederationMode | null>(null);
const modeError = ref<unknown>(null);
const allowlistCount = ref<number | null>(null);
const allowlistError = ref<unknown>(null);

const users = ref<UserDetailed[]>([]);
const usersError = ref<unknown>(null);

const loading = ref(true);

const modeLabel = computed<string>(() =>
	mode.value == null ? "" : MODE_LABELS[mode.value],
);

function describeError(cause: unknown): string {
	if (cause instanceof ApiError) {
		return cause.code != null
			? `${cause.message}（${cause.code}）`
			: cause.message;
	}
	return cause instanceof Error ? cause.message : String(cause);
}

async function loadMeta(): Promise<void> {
	metaError.value = null;
	try {
		meta.value = await api("meta", { detail: false });
	} catch (cause) {
		meta.value = null;
		metaError.value = cause;
	}
}

async function loadFederation(): Promise<void> {
	modeError.value = null;
	allowlistError.value = null;
	allowlistCount.value = null;
	mode.value = null;

	try {
		const response = await apiGet("admin/federation/mode");
		mode.value = response.mode;
	} catch (cause) {
		modeError.value = cause;
		return;
	}

	// The allowlist only describes anything while the mode is `allowlist`; the
	// contract asks for 「未启用」 rather than a stale host count otherwise.
	if (mode.value !== "allowlist") return;

	try {
		const response: AllowlistResponse =
			await apiGet("admin/federation/allowlist");
		allowlistCount.value = response.hosts.length;
	} catch (cause) {
		allowlistError.value = cause;
	}
}

async function loadUsers(): Promise<void> {
	usersError.value = null;
	try {
		// No `sort` parameter exists: the backend returns administrators first,
		// then accounts by ascending registration time.
		users.value = await api("admin/show-users", { limit: 5 });
	} catch (cause) {
		users.value = [];
		usersError.value = cause;
	}
}

async function load(): Promise<void> {
	if (!admin.value) return;

	loading.value = true;
	await Promise.all([loadMeta(), loadFederation(), loadUsers()]);
	loading.value = false;
}

onMounted(() => {
	if (admin.value) void load();
	else loading.value = false;
});

definePageMetadata({
	title: t("overview"),
	icon: "ph-gauge ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-admin-overview {
	display: flex;
	flex-direction: column;
	gap: 16px;

	> .grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: 16px;
	}

	> .notice {
		font-size: 0.85em;
		flex-wrap: wrap;
	}
}

.card {
	display: flex;
	flex-direction: column;
	gap: 12px;
	padding: 16px;

	> .error {
		flex-wrap: wrap;
	}

	> .card-title {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 0;
		font-size: 1em;
		font-weight: bold;

		> i {
			color: var(--accent);
		}
	}

	> .facts {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin: 0;
		> .fact {
			display: flex;
			flex-direction: column;
			gap: 2px;

			> dt {
				font-size: 0.8em;
				color: var(--fgTransparentWeak);
			}

			> dd {
				margin: 0;
				word-break: break-word;

				&.unset {
					color: var(--fgTransparentWeak);
				}

				> .mode {
					font-weight: bold;
					margin-inline-end: 6px;
				}

				> .code {
					color: var(--fgTransparentWeak);
					font-size: 0.85em;
				}
			}
		}
	}

	> .card-link {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-top: auto;
		font-size: 0.9em;
		color: var(--accent);
	}

	&.users-card {
		grid-column: 1 / -1;
	}
}

.user-list {
	display: flex;
	flex-direction: column;
	gap: 4px;

	> .user {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px;
		border-radius: var(--radius);

		&:hover,
		&:focus-visible {
			text-decoration: none;
			background: var(--panelHighlight);
		}

		> .avatar {
			inline-size: 36px;
			block-size: 36px;
			flex-shrink: 0;
		}

		> .user-body {
			min-width: 0;
			flex: 1;

			> .user-name {
				display: flex;
				align-items: center;
				gap: 6px;
				flex-wrap: wrap;
				font-weight: bold;
			}

			> .user-sub {
				display: flex;
				flex-wrap: wrap;
				gap: 6px;
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}
		}
	}
}

.badge {
	font-size: 0.72em;
	font-weight: normal;
	padding: 1px 8px;
	border-radius: 999px;
	line-height: 1.5;
	white-space: nowrap;

	&.admin {
		background: var(--accentedBg);
		color: var(--accent);
	}

	&.suspended {
		color: var(--error);
		border: solid 1px currentColor;
	}
}

.caption {
	margin: 0;
}

.errorText {
	color: var(--error);
}

.retry {
	margin-inline-start: 8px;
}
</style>
