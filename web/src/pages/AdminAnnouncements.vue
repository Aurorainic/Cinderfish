<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/announcements.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-announcements">
				<CfInfo warn>
					需要管理员权限，无法管理公告。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-announcements">
				<form class="createCard" @submit.prevent="submit">
					<h2 class="title">{{ editingId === null ? "新建公告" : "编辑公告" }}</h2>
					<CfInput v-model="title" class="field" :disabled="saving">
						<template #label>标题</template>
						<template #caption>最多 256 个字符。</template>
					</CfInput>
					<CfTextarea v-model="text" class="field" tall :disabled="saving">
						<template #label>正文</template>
						<template #caption
							>按纯文本显示，保留换行；最多 10000 个字符。</template
						>
					</CfTextarea>
					<CfInput
						v-model="imageUrl"
						class="field"
						type="url"
						:disabled="saving"
					>
						<template #label>图片 URL（可选）</template>
						<template #caption
							>http(s) 地址或站点内的绝对路径。</template
						>
					</CfInput>
					<label class="expiryField">
						<span class="label">有效期（可选）</span>
						<input
							v-model="expiresLocal"
							class="nativeInput"
							type="datetime-local"
							:disabled="saving"
						/>
						<span class="caption"
							>留空表示永久；到期后公开列表自动隐藏，管理端仍可见。</span
						>
					</label>

					<CfInfo v-if="formError !== null" warn class="field">
						{{ formError }}
					</CfInfo>
					<div class="actions">
						<CfButton type="submit" primary :disabled="saving">
							{{ saving ? "保存中…" : editingId === null ? "发布" : "保存" }}
						</CfButton>
						<CfButton
							v-if="editingId !== null"
							:disabled="saving"
							@click="cancelEdit()"
						>
							取消
						</CfButton>
					</div>
				</form>

				<CfLoading v-if="loading && list.length === 0" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<CfInfo v-else-if="list.length === 0">
					还没有公告。发布后，所有访客都能在「公告」页看到。
				</CfInfo>
				<ul v-else class="list">
					<li v-for="item in list" :key="item.id" class="item">
						<div class="body">
							<div class="head">
								<h3 class="title">{{ item.title }}</h3>
								<span v-if="isExpired(item)" class="badge expired">已过期</span>
							</div>
							<p class="text">{{ item.text }}</p>
							<div class="meta">
								<MkTime :time="item.createdAt" mode="absolute" />
								<span v-if="item.expiresAt !== null">
									· 有效至
									<MkTime :time="item.expiresAt" mode="absolute" />
								</span>
							</div>
						</div>
						<div class="itemActions">
							<CfButton mini :disabled="saving" @click="beginEdit(item)">
								编辑
							</CfButton>
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
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's admin announcements page edits rows through
// `admin/announcements/{create,update,delete}`; all three exist now, and the
// same form switches between create and edit modes.
//
// Expiry uses the native datetime-local input because CfInput has no such
// type; the value is converted to Unix milliseconds on submit.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { Announcement } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { confirm } from "@/os";

defineOptions({ name: "CfPageAdminAnnouncements" });

const admin = computed<boolean>(() => isAdmin());

const list = ref<Announcement[]>([]);
const loading = ref(false);
const loadError = ref<string | null>(null);
const actionError = ref<string | null>(null);
const deletingId = ref<string | null>(null);
const editingId = ref<string | null>(null);

const title = ref<string | number | null>("");
const text = ref("");
const imageUrl = ref<string | number | null>("");
const expiresLocal = ref("");
const saving = ref(false);
const formError = ref<string | null>(null);

function errorText(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		list.value = await api("admin/announcements/list", { limit: 50 });
	} catch (cause) {
		loadError.value = errorText(cause);
		list.value = [];
	} finally {
		loading.value = false;
	}
}

function isExpired(item: Announcement): boolean {
	return item.expiresAt !== null && new Date(item.expiresAt).getTime() < Date.now();
}

function resetForm(): void {
	editingId.value = null;
	title.value = "";
	text.value = "";
	imageUrl.value = "";
	expiresLocal.value = "";
	formError.value = null;
}

function cancelEdit(): void {
	resetForm();
}

function beginEdit(item: Announcement): void {
	editingId.value = item.id;
	title.value = item.title;
	text.value = item.text;
	imageUrl.value = item.imageUrl ?? "";
	expiresLocal.value =
		item.expiresAt === null ? "" : toLocalInput(item.expiresAt);
	formError.value = null;
}

function toLocalInput(iso: string): string {
	const date = new Date(iso);
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

async function submit(): Promise<void> {
	if (saving.value) return;
	formError.value = null;
	const titleText = String(title.value ?? "").trim();
	const bodyText = text.value.trim();
	if (titleText === "" || bodyText === "") {
		formError.value = "标题和正文都不能为空。";
		return;
	}
	let expiresAt: number | null = null;
	if (expiresLocal.value !== "") {
		const parsed = new Date(expiresLocal.value).getTime();
		if (!Number.isFinite(parsed)) {
			formError.value = "有效期格式不正确。";
			return;
		}
		if (parsed <= Date.now()) {
			formError.value = "有效期必须晚于现在。";
			return;
		}
		expiresAt = parsed;
	}
	const image = String(imageUrl.value ?? "").trim();
	saving.value = true;
	try {
		if (editingId.value === null) {
			await api("admin/announcements/create", {
				title: titleText,
				text: bodyText,
				...(image !== "" ? { imageUrl: image } : {}),
				...(expiresAt !== null ? { expiresAt } : {}),
			});
		} else {
			await api("admin/announcements/update", {
				id: editingId.value,
				title: titleText,
				text: bodyText,
				imageUrl: image === "" ? null : image,
				expiresAt,
			});
		}
		resetForm();
		await load();
	} catch (cause) {
		formError.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

async function remove(item: Announcement): Promise<void> {
	if (deletingId.value !== null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除公告「${item.title}」后，用户将立即看不到它。确定删除吗？`,
	});
	if (canceled) return;
	deletingId.value = item.id;
	actionError.value = null;
	try {
		await api("admin/announcements/delete", { id: item.id });
		list.value = list.value.filter((row) => row.id !== item.id);
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		deletingId.value = null;
	}
}

onMounted(() => {
	void load();
});
</script>

<style lang="scss" scoped>
.createCard {
	margin-bottom: 20px;
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.title {
		margin: 0 0 8px;
		font-size: 1em;
	}

	.field {
		margin-bottom: 12px;
	}

	.expiryField {
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
		background: var(--panel);
		color: var(--fg);
		font: inherit;
	}

	.actions {
		display: flex;
		gap: 8px;
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
	}

	.title {
		margin: 0;
		font-size: 1em;
	}

	.badge.expired {
		padding: 1px 6px;
		border-radius: 999px;
		background: var(--bg);
		font-size: 0.72em;
		opacity: 0.7;
	}

	.text {
		margin: 6px 0 0;
		font-size: 0.9em;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
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
</style>
