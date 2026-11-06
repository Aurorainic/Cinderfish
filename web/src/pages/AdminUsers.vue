<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/users.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-users">
				<CfInfo warn>
					需要管理员权限，无法查看用户列表。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-users">
				<div class="toolbar">
					<div class="summary">
						<span class="page">第 {{ pageNumber }} 页</span>
						<span class="hint">每页 {{ LIMIT }} 位本地账号</span>
					</div>
					<CfButton primary :disabled="creating" @click="toggleCreateForm">
						<i class="ph-user-plus ph-bold"></i>
						创建账号
					</CfButton>
				</div>

				<form
					v-if="createOpen"
					class="create-card"
					@submit.prevent="submitCreate"
				>
					<h2 class="title">创建本地账号</h2>
					<p class="caption">
						注册模式为「管理员创建」时，这是添加用户的唯一入口。
					</p>
					<CfInput
						v-model="createUsername"
						class="field"
						:disabled="creating"
						autocomplete="off"
						:spellcheck="false"
						required
					>
						<template #label>用户名</template>
						<template #prefix>@</template>
						<template #caption
							>字母、数字或下划线，最多 20 个字符。</template
						>
					</CfInput>
					<CfInput
						v-model="createPassword"
						class="field"
						type="password"
						:disabled="creating"
						autocomplete="new-password"
						required
					>
						<template #label>初始密码</template>
						<template #caption
							>密码规则由服务端校验，创建后请转交给该用户。</template
						>
					</CfInput>
					<CfInfo v-if="createError != null" warn class="create-error">
						{{ createError }}
					</CfInfo>
					<div class="create-actions">
						<CfButton type="submit" primary :disabled="creating">
							{{ creating ? "创建中…" : "创建账号" }}
						</CfButton>
						<CfButton :disabled="creating" @click="closeCreateForm">
							取消
						</CfButton>
					</div>
				</form>

				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError != null" @retry="load" />
				<CfInfo v-else-if="users.length === 0" warn class="empty">
					<span v-if="offset === 0">这个实例还没有本地用户。</span>
					<span v-else>这一页没有用户，请返回上一页。</span>
				</CfInfo>
				<div v-else class="user-list">
					<AdminUserRow
						v-for="user in users"
						:key="user.id"
						:user="user"
						:busy="busyUserId === user.id"
						:can-moderate="canModerate(user)"
						@toggle="toggleSuspend"
					/>
				</div>

				<div v-if="!loading" class="pager">
					<CfButton :disabled="offset === 0" @click="goPrev">
						上一页
					</CfButton>
					<span class="status"
						>第 {{ pageNumber }} 页 · 本页 {{ users.length }} 位</span
					>
					<CfButton
						:disabled="!hasNext || loadError != null"
						@click="goNext"
					>
						下一页
					</CfButton>
				</div>

				<CfInfo class="notice">
					后端 <code>admin/show-users</code> 只接受 <code>limit</code> 与
					<code>offset</code>，没有排序、状态、来源筛选、搜索和总数，所以这里只有上一页/下一页，
					下一页依据「本页恰好取满 {{ LIMIT }} 位」判断。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's admin user list is a filtered grid driven by `MkPagination` with
// `sort` / `state` / `origin` / username / host parameters. Cinderfish's
// `admin/show-users` accepts only `limit` and `offset` (contract §2, §5), so:
//   * the three select filters and the two search inputs are gone rather than
//     drawn as dead controls;
//   * pagination is manual offset paging (30 rows per page, previous/next);
//   * each row gains the suspend toggle, which upstream keeps on the
//     user-info page — this port has `admin/suspend-user` but no per-user
//     admin detail endpoint worth a second round-trip;
//   * account creation is an inline form instead of two `os.inputText` dialogs
//     so the server error lands next to the fields and the form works on a
//     phone; it is the only way to add a user when registration is admin-only.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { UserDetailed } from "@/api/types";
import { isAdmin, me } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import AdminUserRow from "@/components/admin/AdminUserRow.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

const LIMIT = 30;

const admin = computed<boolean>(() => isAdmin());
const users = ref<UserDetailed[]>([]);
const offset = ref(0);
const loading = ref(true);
const loadError = ref<unknown>(null);
const busyUserId = ref<string | null>(null);

const createOpen = ref(false);
const createUsername = ref<string | number | null>("");
const createPassword = ref<string | number | null>("");
const creating = ref(false);
const createError = ref<string | null>(null);

const pageNumber = computed<number>(() => Math.floor(offset.value / LIMIT) + 1);
// No total count is available, so a full page is the only "there may be more"
// signal the backend gives.
const hasNext = computed<boolean>(
	() => users.value.length === LIMIT && loadError.value == null,
);

function describeError(cause: unknown): string {
	if (cause instanceof ApiError) {
		return cause.code != null
			? `${cause.message}（${cause.code}）`
			: cause.message;
	}
	return cause instanceof Error ? cause.message : String(cause);
}

function canModerate(user: UserDetailed): boolean {
	return me.value != null && user.id !== me.value.id;
}

async function load(): Promise<void> {
	if (!admin.value) return;

	loading.value = true;
	loadError.value = null;
	try {
		users.value = await api("admin/show-users", {
			limit: LIMIT,
			offset: offset.value,
		});
	} catch (cause) {
		users.value = [];
		loadError.value = cause;
	} finally {
		loading.value = false;
	}
}

function goPrev(): void {
	if (offset.value === 0) return;
	offset.value = Math.max(0, offset.value - LIMIT);
	void load();
}

function goNext(): void {
	if (!hasNext.value) return;
	offset.value += LIMIT;
	void load();
}

function toggleCreateForm(): void {
	createOpen.value = !createOpen.value;
	if (!createOpen.value) {
		createError.value = null;
	}
}

function closeCreateForm(): void {
	if (creating.value) return;
	createOpen.value = false;
	createError.value = null;
}

async function submitCreate(): Promise<void> {
	if (creating.value) return;

	createError.value = null;
	const username = String(createUsername.value ?? "").trim();
	const password = String(createPassword.value ?? "");

	if (username === "") {
		createError.value = "用户名不能为空。";
		return;
	}
	if (password === "") {
		createError.value = "密码不能为空。";
		return;
	}

	creating.value = true;
	try {
		await api("admin/accounts/create", { username, password });
		createUsername.value = "";
		createPassword.value = "";
		createOpen.value = false;
		os.success(`已创建账号 @${username}。`);
		await load();
	} catch (cause) {
		createError.value = describeError(cause);
	} finally {
		creating.value = false;
	}
}

async function toggleSuspend(user: UserDetailed): Promise<void> {
	if (busyUserId.value != null || !canModerate(user)) return;

	const suspend = !user.isSuspended;
	const confirmed = await os.confirm({
		type: "warning",
		text: suspend ? t("suspendConfirm") : t("unsuspendConfirm"),
	});
	if (confirmed.canceled) return;

	busyUserId.value = user.id;
	try {
		if (suspend) {
			await api("admin/suspend-user", { userId: user.id });
		} else {
			await api("admin/unsuspend-user", { userId: user.id });
		}
		users.value = users.value.map((item) =>
			item.id === user.id ? { ...item, isSuspended: suspend } : item,
		);
	} catch (cause) {
		void os.alert({ type: "error", text: describeError(cause) });
	} finally {
		busyUserId.value = null;
	}
}

onMounted(load);

definePageMetadata({
	title: t("users"),
	icon: "ph-users ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-admin-users {
	display: flex;
	flex-direction: column;
	gap: 16px;

	> .toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		flex-wrap: wrap;

		> .summary {
			display: flex;
			flex-direction: column;
			gap: 2px;

			> .page {
				font-weight: bold;
			}

			> .hint {
				font-size: 0.85em;
				color: var(--fgTransparentWeak);
			}
		}
	}

	> .create-card {
		padding: 16px;
		background: var(--panel);
		border-radius: var(--radius);
		border: solid 1px var(--divider);

		> .title {
			margin: 0 0 4px;
			font-size: 1em;
		}

		> .caption {
			margin: 0 0 12px;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}

		> .field {
			margin-bottom: 12px;
		}

		> .create-error {
			margin-bottom: 12px;
		}

		> .create-actions {
			display: flex;
			gap: 8px;
			flex-wrap: wrap;
		}
	}

	> .user-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	> .empty {
		justify-content: center;
	}

	> .pager {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 12px;
		flex-wrap: wrap;

		> .status {
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}
	}

	> .notice {
		font-size: 0.85em;
		flex-wrap: wrap;
	}
}

@media (max-width: 480px) {
	.cf-admin-users {
		> .toolbar {
			align-items: flex-start;
		}
	}
}
</style>
