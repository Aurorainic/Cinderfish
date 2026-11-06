<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/federation.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header><MkPageHeader /></template>
		<MkSpacer :content-max="900">
			<div v-if="!adminUser" class="guard">
				<CfInfo warn>
					这个页面只对管理员开放。当前账号没有管理员权限，因此没有请求任何数据。
				</CfInfo>
				<MkA class="home" to="/">返回首页</MkA>
			</div>

			<MkLoading v-else-if="loading" />

			<template v-else-if="loadFailed">
				<MkError @retry="load" />
			</template>

			<div v-else class="cf-admin-federation">
				<section class="panel _panel">
					<h2>联邦模式</h2>
					<p class="current">
						当前：
						<strong>{{ modeTitle(mode) }}</strong>
					</p>

					<div class="modes" role="radiogroup" aria-label="联邦模式">
						<button
							v-for="option in MODE_OPTIONS"
							:key="option.mode"
							type="button"
							class="mode _button"
							:class="{ active: mode === option.mode }"
							:disabled="savingMode"
							role="radio"
							:aria-checked="mode === option.mode"
							@click="changeMode(option.mode)"
						>
							<span class="modeTitle">
								<i :class="option.icon"></i>
								{{ option.title }}
								<i
									v-if="mode === option.mode"
									class="ph-check-circle ph-bold currentIcon"
								></i>
							</span>
							<span class="modeDescription">{{
								option.description
							}}</span>
						</button>
					</div>

					<CfInfo v-if="modeError != null" warn>{{ modeError }}</CfInfo>
				</section>

				<section class="panel _panel">
					<h2>联邦白名单</h2>

					<CfInfo v-if="mode !== 'allowlist'" warn>
						当前联邦模式是「{{ modeTitle(mode) }}」，白名单不生效。只有切到「仅白名单」后，名单才会限制联邦目标；现在仍然可以预先编辑。
					</CfInfo>

					<CfInfo>
						本实例自身的主机名
						<code>{{ ownHost }}</code>
						始终被允许，既不需要也不能加进列表。
					</CfInfo>

					<ul v-if="hosts.length > 0" class="hosts">
						<li v-for="host in hosts" :key="host" class="host">
							<i class="ph-globe ph-bold"></i>
							<span class="hostName">{{ host }}</span>
							<button
								class="_buttonIcon remove"
								:title="t('delete')"
								:aria-label="t('delete')"
								:disabled="removingHost === host"
								@click="removeHost(host)"
							>
								<i class="ph-x ph-bold"></i>
							</button>
						</li>
					</ul>
					<div v-else class="empty _fullinfo">
						<i class="ph-list-checks ph-bold emptyIcon"></i>
						<div>
							{{
								mode === "allowlist"
									? "白名单为空，当前没有任何远程实例被允许联邦。"
									: "白名单为空。"
							}}
						</div>
					</div>

					<form class="addHost" @submit.prevent="addHost">
						<input
							v-model="newHost"
							class="hostInput"
							type="text"
							inputmode="url"
							autocomplete="off"
							spellcheck="false"
							placeholder="example.com"
							:disabled="addingHost"
						/>
						<CfButton type="submit" primary :disabled="addingHost">
							<i class="ph-plus ph-bold ph-lg"></i>
							{{ addingHost ? "添加中…" : "添加主机" }}
						</CfButton>
					</form>
					<p class="caption">
						只填写主机名：全小写，不带
						<code>https://</code>
						前缀和路径；需要非标准端口时写成
						<code>example.com:3000</code>
						。
					</p>

					<CfInfo v-if="hostError != null" warn>{{ hostError }}</CfInfo>
				</section>

				<CfInfo>
					降级说明：Firefish 的联邦页还会列出每个实例的投递状态与
					pubSub 统计（<code>federation/instances</code>），本实例没有这个端点，因此这里只管理模式与白名单。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's admin/federation.vue is a thin wrapper around the public
// about.federation.vue instance browser (`federation/instances`). Cinderfish
// has no instance list and no per-instance statistics; what it does have is the
// live federation switch and the allowlist:
//
//   GET  admin/federation/mode       -> { mode: public | allowlist | none }
//   POST admin/federation/mode       -> same shape, applies to the next request
//   GET  admin/federation/allowlist  -> { hosts: string[] }
//   POST admin/federation/allowlist  -> { hosts } (add, or remove with remove:true)
//
// Both GETs are real GET routes (see GET_ENDPOINTS in api/client.ts), hence
// apiGet instead of api. The allowlist is stored independently of the mode, so
// editing it while the mode is `public`/`none` is allowed but has no effect —
// the page says so instead of disabling the editor.
import { computed, onMounted, ref } from "vue";
import { ApiError, api, apiGet } from "@/api/client";
import type { FederationModeResponse } from "@/api/types";
import { isAdmin } from "@/account";
import { confirm, success } from "@/os";
import { host as ownHost } from "@/config";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import MkError from "@/components/global/MkError.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import { CfButton, CfInfo } from "@/components/ui";

defineOptions({ name: "CfPageAdminFederation" });

type FederationMode = FederationModeResponse["mode"];

interface ModeOption {
	mode: FederationMode;
	title: string;
	icon: string;
	description: string;
}

const MODE_OPTIONS: readonly ModeOption[] = [
	{
		mode: "public",
		title: "公开联邦",
		icon: "ph-globe-hemisphere-west ph-bold ph-lg",
		description:
			"与所有实例联邦。远程账号可以关注本实例，本实例的投递也发往任意实例。",
	},
	{
		mode: "allowlist",
		title: "仅白名单",
		icon: "ph-list-checks ph-bold ph-lg",
		description:
			"仅与白名单内实例联邦。不在名单里的实例的投递一律拒绝，本实例也不会向它们投递。",
	},
	{
		mode: "none",
		title: "关闭联邦",
		icon: "ph-prohibit ph-bold ph-lg",
		description:
			"完全关闭联邦。既不接收也不发送远程投递，实例只保留本地账号。",
	},
];

const adminUser = computed(() => isAdmin());

const loading = ref(true);
const loadFailed = ref(false);

const mode = ref<FederationMode>("public");
const savingMode = ref(false);
const modeError = ref<string | null>(null);

const hosts = ref<string[]>([]);
const newHost = ref("");
const addingHost = ref(false);
const removingHost = ref<string | null>(null);
const hostError = ref<string | null>(null);

/**
 * One DNS label plus an optional port: lowercase alphanumerics and inner
 * hyphens, dot-separated, no scheme and no path. A bare single label
 * (`localhost`) is accepted because a personal instance may run on one.
 */
const HOST_PATTERN =
	/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?::\d{1,5})?$/;

function modeTitle(value: FederationMode): string {
	return MODE_OPTIONS.find((option) => option.mode === value)?.title ?? value;
}

async function load(): Promise<void> {
	loading.value = true;
	loadFailed.value = false;
	try {
		const [modeResponse, allowlistResponse] = await Promise.all([
			apiGet("admin/federation/mode"),
			apiGet("admin/federation/allowlist"),
		]);
		mode.value = modeResponse.mode;
		hosts.value = allowlistResponse.hosts;
	} catch {
		loadFailed.value = true;
	} finally {
		loading.value = false;
	}
}

async function changeMode(next: FederationMode): Promise<void> {
	if (next === mode.value || savingMode.value) return;

	const confirmed = await confirm({
		type: "warning",
		text: `确定把联邦模式切换为「${modeTitle(next)}」吗？切换立即生效，影响之后的所有投递。`,
	});
	if (confirmed.canceled) return;

	savingMode.value = true;
	modeError.value = null;
	try {
		const response = await api("admin/federation/mode", { mode: next });
		mode.value = response.mode;
		success();
	} catch (error) {
		modeError.value = modeErrorText(error);
	} finally {
		savingMode.value = false;
	}
}

function modeErrorText(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.status === 403) {
			return "当前账号没有管理员权限，无法切换联邦模式。";
		}
		if (error.code === "INVALID_PARAM") {
			return "服务器不接受这个模式值，请刷新页面后重试。";
		}
		return error.message;
	}
	return String(error);
}

/** Returns a Chinese explanation, or null when the host is acceptable. */
function validateHost(value: string): string | null {
	if (value === "") return "请输入主机名。";
	if (value.includes("://")) {
		return "请填写主机名而不是 URL：去掉 http:// 或 https:// 前缀。";
	}
	if (value.includes("/")) return "主机名里不能包含路径或结尾的斜杠。";
	if (/\s/.test(value)) return "主机名里不能包含空格。";
	if (value !== value.toLowerCase()) return "主机名必须全部小写。";
	if (!HOST_PATTERN.test(value)) {
		return "这不是一个有效的主机名，示例：misskey.io、social.example.com:3000。";
	}
	return null;
}

async function addHost(): Promise<void> {
	if (addingHost.value) return;

	const raw = newHost.value.trim();
	const invalid = validateHost(raw);
	if (invalid != null) {
		hostError.value = invalid;
		return;
	}
	if (raw === ownHost.toLowerCase()) {
		hostError.value = "这是本实例自己的主机名，它始终被允许，无需加入白名单。";
		return;
	}
	// The endpoint answers 200 + the unchanged list for a duplicate, which would
	// look like a successful add; catch it here instead.
	if (hosts.value.includes(raw)) {
		hostError.value = `${raw} 已经在白名单里了。`;
		return;
	}

	addingHost.value = true;
	hostError.value = null;
	try {
		const response = await api("admin/federation/allowlist", { host: raw });
		hosts.value = response.hosts;
		newHost.value = "";
		success();
	} catch (error) {
		hostError.value = allowlistErrorText(error);
	} finally {
		addingHost.value = false;
	}
}

async function removeHost(value: string): Promise<void> {
	if (removingHost.value != null) return;

	const confirmed = await confirm({
		type: "warning",
		text: `确定把 ${value} 移出白名单吗？`,
	});
	if (confirmed.canceled) return;

	removingHost.value = value;
	hostError.value = null;
	try {
		const response = await api("admin/federation/allowlist", {
			host: value,
			remove: true,
		});
		hosts.value = response.hosts;
		success();
	} catch (error) {
		hostError.value = allowlistErrorText(error);
	} finally {
		removingHost.value = null;
	}
}

function allowlistErrorText(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.status === 403) {
			return "当前账号没有管理员权限，无法修改白名单。";
		}
		if (error.code === "INVALID_PARAM") {
			return "服务器拒绝了这条主机名，请检查格式后重试。";
		}
		return error.message;
	}
	return String(error);
}

onMounted(() => {
	if (adminUser.value) void load();
});

definePageMetadata({
	title: "联邦",
	icon: "ph-planet ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.guard {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 12px;

	> .home {
		color: var(--accent);
	}
}

.cf-admin-federation {
	display: flex;
	flex-direction: column;
	gap: 1.5em;

	> .panel {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		background: var(--panel);

		> h2 {
			margin: 0;
			font-size: 1em;
		}

		> .current {
			margin: 0;
			color: var(--fgTransparentWeak);

			> strong {
				color: var(--fg);
			}
		}

		> .modes {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
			gap: 12px;

			> .mode {
				display: flex;
				flex-direction: column;
				gap: 6px;
				padding: 12px;
				text-align: start;
				background: var(--bg);
				border: solid 1px var(--divider);
				border-radius: var(--radius);
				transition:
					border-color 0.2s,
					background 0.2s;

				&:hover:not(:disabled) {
					border-color: var(--accent);
				}

				&.active {
					border-color: var(--accent);
					background: var(--accentedBg);
				}

				&:disabled {
					opacity: 0.6;
					cursor: default;
				}

				> .modeTitle {
					display: flex;
					align-items: center;
					gap: 6px;
					font-weight: bold;

					> .currentIcon {
						margin-inline-start: auto;
						color: var(--accent);
					}
				}

				> .modeDescription {
					font-size: 0.85em;
					color: var(--fgTransparentWeak);
					line-height: 1.5;
				}
			}
		}

		> .hosts {
			display: flex;
			flex-direction: column;
			margin: 0;
			padding: 0;
			list-style: none;
			border: solid 1px var(--divider);
			border-radius: var(--radius);
			overflow: hidden;

			> .host {
				display: flex;
				align-items: center;
				gap: 8px;
				padding: 10px 12px;

				& + .host {
					border-block-start: solid 1px var(--divider);
				}

				> .hostName {
					flex: 1;
					min-inline-size: 0;
					overflow: hidden;
					text-overflow: ellipsis;
					white-space: nowrap;
				}

				> .remove {
					inline-size: 28px;
					block-size: 28px;
					color: var(--fgTransparentWeak);

					&:hover {
						color: var(--error);
					}

					&:disabled {
						opacity: 0.4;
					}
				}
			}
		}

		> .empty {
			color: var(--fgTransparentWeak);
		}

		> .addHost {
			display: flex;
			align-items: center;
			gap: 8px;

			> .hostInput {
				flex: 1;
				min-inline-size: 0;
				padding: 8px 12px;
				block-size: 38px;
				background: var(--bg);
				border: solid 1px var(--inputBorder);
				border-radius: 6px;
				color: var(--fg);

				&:focus {
					outline: none;
					border-color: var(--accent);
				}

				&:disabled {
					opacity: 0.6;
				}
			}
		}

		> .caption {
			margin: 0;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
			line-height: 1.5;
		}
	}
}

.emptyIcon {
	display: block;
	margin: 0 auto 8px auto;
	font-size: 32px;
	opacity: 0.5;
}
</style>
