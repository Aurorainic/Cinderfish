<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/security.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="760" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="wrap">
				<CfInfo warn>
					需要管理员权限。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="wrap">
				<section class="invites">
					<h2 class="sectionTitle">邀请码</h2>
					<p class="sectionNote">
						注册模式为「仅限邀请」时，新用户必须凭这里的邀请码注册。切到
						<RouterLink class="_link" to="/admin/settings">实例设置</RouterLink>
						修改注册模式。
					</p>

					<form class="createCard" @submit.prevent="createInvites">
						<label class="countField">
							<span class="label">数量</span>
							<input
								v-model.number="count"
								class="nativeInput"
								type="number"
								min="1"
								max="100"
								:disabled="creating"
							/>
						</label>
						<label class="countField">
							<span class="label">有效期（可选）</span>
							<input
								v-model="expiresLocal"
								class="nativeInput"
								type="datetime-local"
								:disabled="creating"
							/>
							<span class="caption">留空表示永久有效。</span>
						</label>
						<CfInfo v-if="formError !== null" warn class="field">
							{{ formError }}
						</CfInfo>
						<div class="actions">
							<CfButton type="submit" primary :disabled="creating">
								{{ creating ? "生成中…" : "生成邀请码" }}
							</CfButton>
						</div>
					</form>

					<div v-if="createdCodes.length > 0" class="createdCodes">
						<CfInfo class="note">
							已生成 {{ createdCodes.length }} 个邀请码，点击复制发给受邀人：
						</CfInfo>
						<ul class="codeList">
							<li v-for="code in createdCodes" :key="code" class="codeRow">
								<code class="code">{{ code }}</code>
								<CfButton mini @click="copyCode(code)">复制</CfButton>
							</li>
						</ul>
					</div>

					<div class="filterRow">
						<select v-model="filterType" class="nativeSelect" :disabled="loading">
							<option value="all">全部</option>
							<option value="unused">未使用</option>
							<option value="used">已使用</option>
							<option value="expired">已过期</option>
						</select>
						<CfButton mini :disabled="loading" @click="load()">刷新</CfButton>
					</div>

					<CfLoading v-if="loading && list.length === 0" />
					<MkError v-else-if="loadError !== null" @retry="load()" />
					<CfInfo v-else-if="list.length === 0">
						{{ emptyText }}
					</CfInfo>
					<ul v-else class="list">
						<li v-for="item in list" :key="item.id" class="item">
							<div class="body">
								<div class="head">
									<code class="code">{{ item.code }}</code>
									<span v-if="item.used" class="badge used">已使用</span>
									<span
										v-else-if="isExpired(item)"
										class="badge expired"
										>已过期</span
									>
									<span v-else class="badge ok">未使用</span>
								</div>
								<div class="meta">
									<span>
										创建于
										<MkTime :time="item.createdAt" mode="absolute" />
									</span>
									<span
										v-if="item.createdBy != null"
										>· {{ item.createdBy.name ?? item.createdBy.username }}</span
									>
									<span v-if="item.expiresAt !== null">
										· 有效至
										<MkTime :time="item.expiresAt" mode="absolute" />
									</span>
									<span v-if="item.usedBy != null">
										· 使用者
										<RouterLink
											class="_link"
											:to="`/@${item.usedBy.username}`"
											>{{ item.usedBy.name ?? item.usedBy.username }}</RouterLink
										>
									</span>
								</div>
							</div>
							<div class="itemActions">
								<CfButton mini @click="copyCode(item.code)">复制</CfButton>
								<CfButton
									mini
									:disabled="deletingId === item.id"
									@click="remove(item)"
								>
									{{ deletingId === item.id ? "删除中…" : "删除" }}
								</CfButton>
							</div>
						</li>
					</ul>
					<CfInfo v-if="actionError !== null" warn class="actionError">
						{{ actionError }}
					</CfInfo>
				</section>

				<section class="signinAudit">
					<h2 class="sectionTitle">登录审计</h2>
					<p class="sectionNote">
						所有账号的登录记录，含失败尝试。成功登录也会出现在账号自己的设置页里。
					</p>
					<div class="filterRow">
						<CfButton mini :disabled="auditLoading" @click="loadAudit()">刷新</CfButton>
					</div>
					<CfLoading v-if="auditLoading && audit.length === 0" />
					<MkError v-else-if="auditError !== null" @retry="loadAudit()" />
					<CfInfo v-else-if="audit.length === 0">还没有登录记录。</CfInfo>
					<ul v-else class="list">
						<li v-for="entry in audit" :key="entry.id" class="item auditRow">
							<div class="body">
								<div class="head">
									<RouterLink class="user" :to="`/@${entry.user.username}`"
										>@{{ entry.user.username }}</RouterLink
									>
									<span :class="['badge', entry.success ? 'ok' : 'expired']">{{
										entry.success ? "成功" : "失败"
									}}</span>
									<code class="code">{{ entry.ip || "未知地址" }}</code>
								</div>
								<div class="meta">
									<MkTime :time="entry.createdAt" mode="absolute" />
								</div>
							</div>
						</li>
					</ul>
				</section>

				<section class="ipBans">
					<h2 class="sectionTitle">IP 封禁</h2>
					<p class="sectionNote">
						被封锁的地址无法登录或注册（浏览不受影响）。规则可以是单个 IP 或 CIDR
						网段；到期后自动失效。
					</p>
					<form class="createCard" @submit.prevent="createBan">
						<label class="countField">
							<span class="label">IP 或网段</span>
							<input
								v-model="banIP"
								class="nativeInput"
								type="text"
								placeholder="192.0.2.7 或 198.51.100.0/24"
								:disabled="banCreating"
							/>
						</label>
						<label class="countField">
							<span class="label">原因（可选）</span>
							<input
								v-model="banReason"
								class="nativeInput"
								type="text"
								:disabled="banCreating"
							/>
						</label>
						<label class="countField">
							<span class="label">有效期（可选）</span>
							<input
								v-model="banExpiresLocal"
								class="nativeInput"
								type="datetime-local"
								:disabled="banCreating"
							/>
							<span class="caption">留空表示永久封禁。</span>
						</label>
						<CfInfo v-if="banFormError !== null" warn class="field">
							{{ banFormError }}
						</CfInfo>
						<div class="actions">
							<CfButton type="submit" primary :disabled="banCreating">
								{{ banCreating ? "封禁中…" : "封禁" }}
							</CfButton>
						</div>
					</form>

					<CfLoading v-if="banLoading && bans.length === 0" />
					<MkError v-else-if="banError !== null" @retry="loadBans()" />
					<CfInfo v-else-if="bans.length === 0">没有 IP 封禁规则。</CfInfo>
					<ul v-else class="list">
						<li v-for="ban in bans" :key="ban.id" class="item">
							<div class="body">
								<div class="head">
									<code class="code">{{ ban.ip }}</code>
									<span v-if="isBanExpired(ban)" class="badge expired">已过期</span>
									<span v-else class="badge ok">生效中</span>
								</div>
								<div class="meta">
									<span v-if="ban.reason !== ''">{{ ban.reason }}</span>
									<span>
										创建于
										<MkTime :time="ban.createdAt" mode="absolute" />
									</span>
									<span v-if="ban.createdBy !== null"
										>· {{ ban.createdBy.username }}</span
									>
									<span v-if="ban.expiresAt !== null">
										· 至
										<MkTime :time="ban.expiresAt" mode="absolute" />
									</span>
								</div>
							</div>
							<div class="itemActions">
								<CfButton
									mini
									:disabled="banDeletingId === ban.id"
									@click="removeBan(ban)"
								>
									{{ banDeletingId === ban.id ? "删除中…" : "删除" }}
								</CfButton>
							</div>
						</li>
					</ul>
					<CfInfo v-if="banActionError !== null" warn class="actionError">
						{{ banActionError }}
					</CfInfo>
				</section>

				<AdminUnavailable
					class="rest"
					:title="rest.title"
					:summary="rest.summary"
					:available="rest.available"
					:missing="rest.missing"
				/>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's admin/security page manages bot protection, email validation
// and IP logging, none of which this backend has. What it does have is
// invite-code management (Misskey keeps that under admin/invite/*) plus two
// Cinderfish extensions: IP bans and a cross-account signin audit. The
// AdminUnavailable block at the bottom records what is still absent
// (captcha support, automatic bans after repeated failures).
import { computed, onMounted, ref, watch } from "vue";
import { ApiError, api } from "@/api/client";
import type { AdminSigninEntry, IPBan } from "@/api/types";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { confirm } from "@/os";
import { copyToClipboard } from "@/scripts/copy-to-clipboard";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminSecurity" });

interface InviteCodeRow {
	id: string;
	code: string;
	createdAt: string;
	expiresAt: string | null;
	usedAt: string | null;
	used: boolean;
	createdBy: { username: string; name?: string | null } | null;
	usedBy: { username: string; name?: string | null } | null;
}

const admin = computed<boolean>(() => isAdmin());

const list = ref<InviteCodeRow[]>([]);
const loading = ref(false);
const loadError = ref<string | null>(null);
const actionError = ref<string | null>(null);
const deletingId = ref<string | null>(null);
const filterType = ref<"all" | "unused" | "used" | "expired">("all");

const count = ref(1);
const expiresLocal = ref("");
const creating = ref(false);
const formError = ref<string | null>(null);
const createdCodes = ref<string[]>([]);

const audit = ref<AdminSigninEntry[]>([]);
const auditLoading = ref(false);
const auditError = ref<string | null>(null);

const bans = ref<IPBan[]>([]);
const banLoading = ref(false);
const banError = ref<string | null>(null);
const banActionError = ref<string | null>(null);
const banDeletingId = ref<string | null>(null);
const banIP = ref("");
const banReason = ref("");
const banExpiresLocal = ref("");
const banCreating = ref(false);
const banFormError = ref<string | null>(null);

const rest = {
	title: "其余安全设置",
	summary:
		"上游的安全页还集中管理机器人防护与邮件验证。这些依赖外部服务，当前没有实现；登录限流按 IP 在后端内置，登录记录见上方审计区。",
	available: [
		{ label: "注册模式", note: "open / invite / admin 三档，运行期可切换", to: "/admin/settings" },
		{ label: "封禁用户", note: "在用户列表里冻结/解冻账号", to: "/admin/users" },
		{ label: "两步验证", note: "由每个账号在安全设置里自行开启", to: "/settings/security" },
	],
	missing: ["机器人防护（hCaptcha / reCAPTCHA）", "重复失败后的自动封禁"],
};

function errorText(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

const emptyText = computed<string>(() => {
	switch (filterType.value) {
		case "unused":
			return "没有未使用的邀请码。生成一个发给受邀人吧。";
		case "used":
			return "还没有被使用的邀请码。";
		case "expired":
			return "没有已过期的邀请码。";
		default:
			return "还没有邀请码。生成一个发给受邀人吧。";
	}
});

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		list.value = await api<InviteCodeRow[]>("admin/invite/list", {
			limit: 100,
			type: filterType.value,
		});
	} catch (cause) {
		loadError.value = errorText(cause);
		list.value = [];
	} finally {
		loading.value = false;
	}
}

function isExpired(item: InviteCodeRow): boolean {
	return (
		!item.used &&
		item.expiresAt !== null &&
		new Date(item.expiresAt).getTime() < Date.now()
	);
}

async function createInvites(): Promise<void> {
	if (creating.value) return;
	formError.value = null;
	const n = Number(count.value);
	if (!Number.isInteger(n) || n < 1 || n > 100) {
		formError.value = "数量必须是 1–100 之间的整数。";
		return;
	}
	let expiresAt: string | null = null;
	if (expiresLocal.value !== "") {
		const parsed = new Date(expiresLocal.value);
		if (Number.isNaN(parsed.getTime())) {
			formError.value = "有效期格式不正确。";
			return;
		}
		if (parsed.getTime() <= Date.now()) {
			formError.value = "有效期必须晚于现在。";
			return;
		}
		expiresAt = parsed.toISOString();
	}
	creating.value = true;
	try {
		const created = await api<InviteCodeRow[]>(
			"admin/invite/create",
			expiresAt === null
				? { count: n }
				: { count: n, expiresAt },
		);
		createdCodes.value = created.map((row) => row.code);
		await load();
	} catch (cause) {
		formError.value = errorText(cause);
	} finally {
		creating.value = false;
	}
}

async function copyCode(code: string): Promise<void> {
	await copyToClipboard(code);
}

async function remove(item: InviteCodeRow): Promise<void> {
	if (deletingId.value !== null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除邀请码 ${item.code} 后，它将无法再被使用。确定删除吗？`,
	});
	if (canceled) return;
	deletingId.value = item.id;
	actionError.value = null;
	try {
		await api("admin/invite/delete", { inviteId: item.id });
		list.value = list.value.filter((row) => row.id !== item.id);
		createdCodes.value = createdCodes.value.filter((code) => code !== item.code);
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		deletingId.value = null;
	}
}

async function loadAudit(): Promise<void> {
	if (!admin.value) return;
	auditLoading.value = true;
	auditError.value = null;
	try {
		audit.value = await api<AdminSigninEntry[]>("admin/signin-history", {
			limit: 100,
		});
	} catch (cause) {
		auditError.value = errorText(cause);
		audit.value = [];
	} finally {
		auditLoading.value = false;
	}
}

async function loadBans(): Promise<void> {
	if (!admin.value) return;
	banLoading.value = true;
	banError.value = null;
	try {
		bans.value = await api<IPBan[]>("admin/ip-bans/list", { limit: 100 });
	} catch (cause) {
		banError.value = errorText(cause);
		bans.value = [];
	} finally {
		banLoading.value = false;
	}
}

function isBanExpired(ban: IPBan): boolean {
	return (
		ban.expiresAt !== null && new Date(ban.expiresAt).getTime() < Date.now()
	);
}

async function createBan(): Promise<void> {
	if (banCreating.value) return;
	banFormError.value = null;
	const rule = banIP.value.trim();
	if (rule === "") {
		banFormError.value = "IP 地址或网段不能为空。";
		return;
	}
	let expiresAt: string | null = null;
	if (banExpiresLocal.value !== "") {
		const parsed = new Date(banExpiresLocal.value);
		if (Number.isNaN(parsed.getTime())) {
			banFormError.value = "有效期格式不正确。";
			return;
		}
		if (parsed.getTime() <= Date.now()) {
			banFormError.value = "有效期必须晚于现在。";
			return;
		}
		expiresAt = parsed.toISOString();
	}
	banCreating.value = true;
	try {
		await api<IPBan>(
			"admin/ip-bans/create",
			expiresAt === null
				? { ip: rule, reason: banReason.value.trim() }
				: { ip: rule, reason: banReason.value.trim(), expiresAt },
		);
		banIP.value = "";
		banReason.value = "";
		banExpiresLocal.value = "";
		await loadBans();
	} catch (cause) {
		banFormError.value = errorText(cause);
	} finally {
		banCreating.value = false;
	}
}

async function removeBan(ban: IPBan): Promise<void> {
	if (banDeletingId.value !== null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除 ${ban.ip} 的封禁规则后，该地址可以重新登录与注册。确定删除吗？`,
	});
	if (canceled) return;
	banDeletingId.value = ban.id;
	banActionError.value = null;
	try {
		await api("admin/ip-bans/delete", { id: ban.id });
		bans.value = bans.value.filter((row) => row.id !== ban.id);
	} catch (cause) {
		banActionError.value = errorText(cause);
	} finally {
		banDeletingId.value = null;
	}
}

onMounted(() => {
	void load();
	void loadAudit();
	void loadBans();
});

// Changing the type filter re-queries immediately; the 刷新 button is for
// picking up rows created elsewhere.
watch(filterType, () => {
	void load();
});

definePageMetadata({ title: "安全设置", icon: "ph-shield-check ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}

.signinAudit,
.ipBans {
	margin-top: 24px;
}

.auditRow .user {
	font-weight: 600;
}

.sectionTitle {
	margin: 0 0 4px;
	font-size: 1.05em;
}

.sectionNote {
	margin: 0 0 12px;
	font-size: 0.85em;
	opacity: 0.7;
}

.createCard {
	margin-bottom: 16px;
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.countField {
		display: flex;
		flex-direction: column;
		gap: 4px;
		margin-bottom: 12px;

		.label {
			font-size: 0.85em;
			opacity: 0.8;
		}

		.caption {
			font-size: 0.78em;
			opacity: 0.6;
		}
	}

	.nativeInput {
		padding: 8px 10px;
		border: 1px solid var(--divider);
		border-radius: 6px;
		background: var(--bg);
		color: var(--fg);
		font: inherit;
		width: 240px;
		max-width: 100%;
	}

	.field {
		margin-bottom: 12px;
	}

	.actions {
		display: flex;
		gap: 8px;
	}
}

.createdCodes {
	margin-bottom: 16px;

	.note {
		margin-bottom: 8px;
	}

	.codeList {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.codeRow {
		display: flex;
		align-items: center;
		gap: 8px;

		.code {
			flex: 1;
			min-width: 0;
			padding: 6px 10px;
			background: var(--bg);
			border-radius: 6px;
			font-size: 0.9em;
			overflow-wrap: anywhere;
		}
	}
}

.filterRow {
	display: flex;
	align-items: center;
	gap: 8px;
	margin-bottom: 12px;

	.nativeSelect {
		padding: 6px 10px;
		border: 1px solid var(--divider);
		border-radius: 6px;
		background: var(--panel);
		color: var(--fg);
		font: inherit;
	}
}

.list {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.item {
	display: flex;
	align-items: flex-start;
	gap: 12px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.body {
		flex: 1;
		min-width: 0;
	}

	.head {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.code {
		padding: 2px 8px;
		background: var(--bg);
		border-radius: 6px;
		font-size: 0.9em;
		overflow-wrap: anywhere;
	}

	.badge {
		padding: 1px 6px;
		border-radius: 999px;
		font-size: 0.72em;

		&.ok {
			background: var(--bg);
			opacity: 0.75;
		}

		&.used,
		&.expired {
			background: var(--bg);
			opacity: 0.55;
		}
	}

	.meta {
		margin-top: 6px;
		font-size: 0.78em;
		opacity: 0.6;
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.itemActions {
		display: flex;
		gap: 8px;
		flex-shrink: 0;
	}
}

.actionError {
	margin-top: 12px;
}

.rest {
	margin-top: 24px;
}
</style>
