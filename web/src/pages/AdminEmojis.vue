<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/emojis.vue (AGPL-3.0-only) -->
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

			<div v-else class="cf-admin-emojis">
				<div class="toolbar">
					<FormInput
						v-model="query"
						class="search"
						type="search"
						:debounce="true"
					>
						<template #prefix
							><i
								class="ph-magnifying-glass ph-bold ph-lg"
							></i
						></template>
						<template #label>{{ t("search") }}</template>
					</FormInput>
					<CfButton class="addButton" primary @click="openForm">
						<i class="ph-plus ph-bold ph-lg"></i>
						添加表情
					</CfButton>
				</div>

				<form
					v-if="formOpen"
					class="form _panel"
					@submit.prevent="submit"
				>
					<div class="formHeader">
						<h2>添加自定义表情</h2>
						<button
							type="button"
							class="_buttonIcon close"
							:aria-label="t('close')"
							@click="closeForm"
						>
							<i class="ph-x ph-bold"></i>
						</button>
					</div>

					<FormInput
						v-model="nameInput"
						class="_formBlock"
						:disabled="saving"
					>
						<template #label>名称</template>
						<template #caption
							>笔记里写成
							<code>:名称:</code>
							。只能使用字母、数字和下划线，与后端校验一致。</template
						>
					</FormInput>

					<FormInput
						v-model="aliasInput"
						class="_formBlock"
						:disabled="saving"
					>
						<template #label>别名</template>
						<template #caption
							>用逗号分隔，可以留空。每个别名同样只能使用字母、数字和下划线。</template
						>
					</FormInput>

					<FormInput
						v-model="categoryInput"
						class="_formBlock"
						:disabled="saving"
					>
						<template #label>类别（可选）</template>
						<template #caption
							>表情选择器里的分组名，留空表示不分组。</template
						>
					</FormInput>

					<div class="fileBlock">
						<div class="fileInfo">
							<span class="fileLabel">图片</span>
							<span v-if="selectedFile == null" class="fileEmpty"
								>尚未选择</span
							>
							<span v-else class="fileName">
								<i class="ph-file-image ph-bold"></i>
								{{ selectedFile.name }}
							</span>
						</div>
						<CfButton :disabled="saving" @click="pickFile">
							<i class="ph-image ph-bold ph-lg"></i>
							{{ selectedFile == null ? "选择图片" : "重新选择" }}
						</CfButton>
					</div>

					<CfInfo v-if="formError != null" warn>{{ formError }}</CfInfo>

					<div class="actions">
						<CfButton :disabled="saving" @click="closeForm">
							取消
						</CfButton>
						<CfButton type="submit" primary :disabled="saving">
							{{ saving ? "添加中…" : "添加" }}
						</CfButton>
					</div>
				</form>

				<CfPagination
					ref="paginationComponent"
					:key="paginationKey"
					:pagination="pagination"
				>
					<template #empty>
						<div class="_fullinfo">
							<i class="ph-smiley ph-bold emptyIcon"></i>
							<div>{{ emptyText }}</div>
						</div>
					</template>

					<template #default="{ items }">
						<div class="grid">
							<div
								v-for="emoji in emojis(items)"
								:key="emoji.id"
								class="emoji _panel"
							>
								<img
									class="img"
									:src="resolveMediaUrl(emoji.url)"
									:alt="emoji.name"
									loading="lazy"
								/>
								<div class="body">
									<div class="name _monospace" :title="emoji.name">
										:{{ emoji.name }}:
									</div>
									<div class="meta">
										<span
											v-if="
												emoji.category != null &&
												emoji.category !== ''
											"
											class="category"
											>{{ emoji.category }}</span
										>
										<span
											v-if="aliasesText(emoji) !== ''"
											class="aliases"
											:title="aliasesText(emoji)"
											>{{ aliasesText(emoji) }}</span
										>
									</div>
								</div>
								<button
									class="_buttonIcon delete"
									:title="t('delete')"
									:aria-label="t('delete')"
									:disabled="deletingId === emoji.id"
									@click="removeEmoji(emoji)"
								>
									<i class="ph-trash ph-bold"></i>
								</button>
							</div>
						</div>
					</template>
				</CfPagination>

				<CfInfo v-if="deleteError != null" warn class="deleteError">
					{{ deleteError }}
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish renders a local/remote tab pair plus bulk editing through
// `admin/emoji/list-remote`, `set-category-bulk`, `add-aliases-bulk`,
// `delete-bulk` and the zip import/export pair. None of those endpoints exist
// here, so this page only uses the three that do:
//
//   admin/emoji/list   cursor pagination, `query` filters by name/alias
//   admin/emoji/add    name + a drive file the admin owns
//   admin/emoji/delete by id
//
// Consequences, all deliberate:
//   * no bulk selection — there is no bulk endpoint to drive it;
//   * no edit dialog — `admin/emoji/update` does not exist, so a wrong name can
//     only be fixed by deleting and re-adding;
//   * no remote tab / zip import — those endpoints are absent.
//
// The add form is an inline card instead of `emoji-edit-dialog.vue` so this page
// keeps its port to a single file (the admin component directory belongs to
// another workstream of this wave).
import { computed, ref, watch } from "vue";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
import type { AdminEmoji, DriveFile } from "@/api/types";
import { isAdmin } from "@/account";
import { confirm, success } from "@/os";
import { selectFiles } from "@/scripts/select-file";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { CfButton, CfInfo, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import FormInput from "@/components/form/input.vue";

defineOptions({ name: "CfPageAdminEmojis" });

/** internal/httpapi/emojis.go: emojiNamePattern */
const NAME_PATTERN = /^[a-zA-Z0-9_]+$/;

const adminUser = computed(() => isAdmin());

const query = ref("");
const appliedQuery = ref("");
const paginationKey = ref(0);

const pagination = computed<Paging>(() => ({
	endpoint: "admin/emoji/list",
	limit: 30,
	params: { query: appliedQuery.value === "" ? null : appliedQuery.value },
}));

interface EmojiPaginationHandle {
	reload: () => Promise<AdminEmoji[]>;
	removeItem: (finder: (item: { id: string }) => boolean) => boolean;
}

const paginationComponent = ref<EmojiPaginationHandle | null>(null);

const formOpen = ref(false);
const nameInput = ref("");
const aliasInput = ref("");
const categoryInput = ref("");
const selectedFile = ref<DriveFile | null>(null);
const saving = ref(false);
const formError = ref<string | null>(null);

const deletingId = ref<string | null>(null);
const deleteError = ref<string | null>(null);

const emptyText = computed(() =>
	appliedQuery.value === ""
		? t("noCustomEmojis")
		: `没有匹配「${appliedQuery.value}」的表情。`,
);

// The search box is debounced by CfInput; re-mounting CfPagination is how the
// new `query` reaches the endpoint, because usePagination resolves params only
// when it loads a page.
watch(query, (value) => {
	const next = value.trim();
	if (next === appliedQuery.value) return;
	appliedQuery.value = next;
	deleteError.value = null;
	paginationKey.value += 1;
});

// CfPagination's slot payload is typed as `{ id: string }[]`; the endpoint is
// admin/emoji/list, so re-typing it here is safe and keeps the template narrow.
function emojis(items: readonly { id: string }[]): AdminEmoji[] {
	return items as AdminEmoji[];
}

function aliasesText(emoji: AdminEmoji): string {
	return (emoji.aliases ?? []).map((alias) => `:${alias}:`).join(" ");
}

function parseAliases(value: string): string[] {
	return value
		.split(/[,，\s]+/)
		.map((alias) => alias.trim())
		.filter((alias) => alias !== "");
}

function openForm(): void {
	formOpen.value = true;
	formError.value = null;
}

function resetForm(): void {
	formOpen.value = false;
	formError.value = null;
	nameInput.value = "";
	aliasInput.value = "";
	categoryInput.value = "";
	selectedFile.value = null;
}

function closeForm(): void {
	// The submit path resets the form itself; while a request is in flight the
	// cancel button is disabled, so this guard only stops stray Escape-ish clicks.
	if (saving.value) return;
	resetForm();
}

async function pickFile(event: MouseEvent): Promise<void> {
	const files = await selectFiles({
		multiple: false,
		label: "表情图片",
		src: event,
	});
	if (files.length === 0) return;
	selectedFile.value = files[0];
	formError.value = null;
}

/**
 * Client-side mirror of the server's own checks, so a rejected name is caught
 * before the request instead of coming back as INVALID_PARAM.
 */
function validateForm(): { name: string; aliases: string[] } | null {
	const name = nameInput.value.trim();
	if (name === "") {
		formError.value = "请填写表情名称。";
		return null;
	}
	if (!NAME_PATTERN.test(name)) {
		formError.value = "名称只能使用字母、数字和下划线（a-zA-Z0-9_）。";
		return null;
	}

	const aliases = parseAliases(aliasInput.value);
	const badAlias = aliases.find((alias) => !NAME_PATTERN.test(alias));
	if (badAlias != null) {
		formError.value = `别名「${badAlias}」只能使用字母、数字和下划线。`;
		return null;
	}

	if (selectedFile.value == null) {
		formError.value = "请先从网盘选择图片，或者上传一张新图片。";
		return null;
	}

	return { name, aliases };
}

async function submit(): Promise<void> {
	if (saving.value) return;
	formError.value = null;

	const valid = validateForm();
	if (valid == null) return;
	const file = selectedFile.value;
	if (file == null) return;

	saving.value = true;
	try {
		await api("admin/emoji/add", {
			name: valid.name,
			fileId: file.id,
			category:
				categoryInput.value.trim() === ""
					? null
					: categoryInput.value.trim(),
			aliases: valid.aliases,
		});
		resetForm();
		success();
		await paginationComponent.value?.reload();
	} catch (error) {
		formError.value = addErrorText(error);
	} finally {
		saving.value = false;
	}
}

function addErrorText(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.code === "EMOJI_ALREADY_EXISTS") {
			return "这个名称或别名已经被占用了，换一个名字再试。";
		}
		if (error.code === "NO_SUCH_FILE") {
			return "选中的文件不在管理员网盘里，或者已经被删除。";
		}
		if (error.status === 403) {
			return "当前账号没有管理员权限，无法添加表情。请重新登录后再试。";
		}
		return error.message;
	}
	return String(error);
}

async function removeEmoji(emoji: AdminEmoji): Promise<void> {
	if (deletingId.value != null) return;

	const confirmed = await confirm({
		type: "warning",
		text: `删除 :${emoji.name}: 之后，已经发送的笔记里它只会显示成纯文本。确定删除吗？`,
	});
	if (confirmed.canceled) return;

	deletingId.value = emoji.id;
	deleteError.value = null;
	try {
		await api("admin/emoji/delete", { id: emoji.id });
		paginationComponent.value?.removeItem((item) => item.id === emoji.id);
		success();
	} catch (error) {
		deleteError.value = deleteErrorText(error);
	} finally {
		deletingId.value = null;
	}
}

function deleteErrorText(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.code === "NO_SUCH_EMOJI") {
			return "这个表情已经不存在了，列表可能已经过期。";
		}
		if (error.status === 403) {
			return "当前账号没有管理员权限，无法删除表情。";
		}
		return error.message;
	}
	return String(error);
}

definePageMetadata({
	title: "自定义表情",
	icon: "ph-smiley ph-bold ph-lg",
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

.cf-admin-emojis {
	> .toolbar {
		display: flex;
		align-items: flex-end;
		gap: 12px;
		margin-block-end: 1.5em;

		> .search {
			flex: 1;
			min-inline-size: 0;
		}

		> .addButton {
			flex-shrink: 0;
		}
	}

	> .form {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;
		margin-block-end: 1.5em;
		background: var(--panel);

		> .formHeader {
			display: flex;
			align-items: center;
			justify-content: space-between;

			> h2 {
				margin: 0;
				font-size: 1em;
			}
		}

		> .fileBlock {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 12px;
			padding: 10px 12px;
			border: solid 1px var(--divider);
			border-radius: var(--radius);

			> .fileInfo {
				display: flex;
				align-items: center;
				gap: 8px;
				min-inline-size: 0;

				> .fileLabel {
					font-size: 0.85em;
					color: var(--fgTransparentWeak);
				}

				> .fileEmpty {
					color: var(--fgTransparentWeak);
				}

				> .fileName {
					display: inline-flex;
					align-items: center;
					gap: 6px;
					overflow: hidden;
					text-overflow: ellipsis;
					white-space: nowrap;
				}
			}
		}

		> .actions {
			display: flex;
			justify-content: flex-end;
			gap: 8px;
		}
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
		gap: 12px;
		margin-block: var(--margin);

		> .emoji {
			position: relative;
			display: flex;
			align-items: center;
			min-inline-size: 0;
			padding: 11px;
			background: var(--panel);
			border: solid 1px var(--divider);
			border-radius: var(--radius);

			> .img {
				flex-shrink: 0;
				inline-size: 42px;
				block-size: 42px;
				object-fit: contain;
			}

			> .body {
				min-inline-size: 0;
				padding-inline-start: 8px;
				white-space: nowrap;
				overflow: hidden;

				> .name {
					overflow: hidden;
					text-overflow: ellipsis;
				}

				> .meta {
					display: flex;
					flex-direction: column;
					font-size: 0.75em;
					color: var(--fgTransparentWeak);

					> .category,
					> .aliases {
						overflow: hidden;
						text-overflow: ellipsis;
					}
				}
			}

			> .delete {
				position: absolute;
				inset-block-start: 6px;
				inset-inline-end: 6px;
				inline-size: 26px;
				block-size: 26px;
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

	> .deleteError {
		margin-block-start: 12px;
	}
}

.emptyIcon {
	display: block;
	margin: 0 auto 8px auto;
	font-size: 32px;
	opacity: 0.5;
}
</style>
