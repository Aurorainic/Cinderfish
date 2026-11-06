<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/import-export.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-import-export">
				<section class="_formBlock card">
					<h2 class="title">导出账号数据</h2>
					<p class="caption">
						下载一个 JSON 文件，包含你的资料、全部帖子（含私密与关注者可见）、关注与粉丝列表、
						收藏以及网盘文件的元数据。文件里不含别人的帖子，也不含登录 token。
					</p>
					<p class="caption">
						这是 Cinderfish 自己的备份格式（<code>cinderfish-account-export@1</code>），
						不是 Misskey/Firefish 的 zip；对应的导入端点尚未实现。
					</p>
					<CfButton primary :disabled="exporting" @click="exportData">
						<i class="ph-download-simple ph-bold"></i>
						{{ exporting ? "导出中…" : "导出（JSON）" }}
					</CfButton>
					<CfInfo v-if="error !== null" warn class="feedback">
						{{ error }}
					</CfInfo>
					<CfInfo v-else-if="savedAs !== null" class="feedback">
						已下载 {{ savedAs }}（{{ savedSize }}）。
					</CfInfo>
				</section>

				<CfInfo warn>
					导入只接受本页导出的 <code>cinderfish-account-export@1</code> JSON。
					笔记保留原 id、冲突跳过；关注与收藏只在目标用户/笔记已存在时写入。
					导入是增量操作，不会删除任何现有数据。
				</CfInfo>

				<section class="_formBlock card">
					<h2 class="title">导入账号数据</h2>
					<p class="caption">
						选择一份之前导出的 JSON。文件会先上传到你的云盘，再解析导入；
						建议在空账号或测试账号上使用。
					</p>
					<input
						ref="fileInput"
						type="file"
						accept="application/json,.json"
						class="hiddenInput"
						@change="importData"
					/>
					<CfButton
						primary
						:disabled="importing"
						@click="fileInput?.click()"
					>
						<i class="ph-upload-simple ph-bold"></i>
						{{ importing ? "导入中…" : "选择 JSON 并导入" }}
					</CfButton>
					<CfInfo v-if="importError !== null" warn class="feedback">
						{{ importError }}
					</CfInfo>
					<CfInfo v-else-if="importResult !== null" class="feedback">
						导入完成：笔记 {{ importResult.notes }} 条（跳过 {{ importResult.skipped }}），
						关注 {{ importResult.followings }} 条（跳过 {{ importResult.skippedFollowings }}），
						收藏 {{ importResult.favorites }} 条（跳过 {{ importResult.skippedFavorites }}）。
					</CfInfo>
				</section>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish splits this page into per-dataset exports (`i/export-notes`,
// `i/export-following`, …) delivered asynchronously through jobs. Cinderfish
// answers `i/export` synchronously with one JSON document, which is small
// enough for a personal instance and can be opened with any editor. `i/import`
// reads that same format back: notes keep their ids when free, followings and
// favorites are restored only when their targets already exist.
import { ref } from "vue";
import { ApiError, api, apiDownload } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { ImportSummary } from "@/api/types";
import { uploadFile } from "@/scripts/upload";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";

defineOptions({ name: "CfPageSettingsImportExport" });

const exporting = ref(false);
const error = ref<string | null>(null);
const savedAs = ref<string | null>(null);
const savedSize = ref("");

const fileInput = ref<HTMLInputElement | null>(null);
const importing = ref(false);
const importError = ref<string | null>(null);
const importResult = ref<ImportSummary | null>(null);

async function importData(event: Event): Promise<void> {
	const input = event.target as HTMLInputElement;
	const file = input.files?.[0];
	input.value = "";
	if (file == null || importing.value) return;
	importing.value = true;
	importError.value = null;
	importResult.value = null;
	try {
		const uploaded = await uploadFile(file, { name: file.name });
		importResult.value = await api("i/import", { fileId: uploaded.id });
	} catch (cause) {
		importError.value =
			cause instanceof ApiError ? cause.message : String(cause);
	} finally {
		importing.value = false;
	}
}

function formatSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

async function exportData(): Promise<void> {
	if (exporting.value) return;
	exporting.value = true;
	error.value = null;
	savedAs.value = null;
	try {
		const { blob, filename } = await apiDownload(ENDPOINTS.iExport);
		const name = filename ?? "cinderfish-export.json";
		const url = URL.createObjectURL(blob);
		const link = document.createElement("a");
		link.href = url;
		link.download = name;
		link.rel = "noopener";
		document.body.appendChild(link);
		link.click();
		link.remove();
		// Revoke on the next task so the browser has started the download.
		window.setTimeout(() => URL.revokeObjectURL(url), 5000);
		savedAs.value = name;
		savedSize.value = formatSize(blob.size);
	} catch (cause) {
		error.value = cause instanceof ApiError ? cause.message : String(cause);
	} finally {
		exporting.value = false;
	}
}
</script>

<style lang="scss" scoped>
.card {
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.title {
		margin: 0 0 6px;
		font-size: 1em;
	}

	.caption {
		margin: 0 0 10px;
		font-size: 0.88em;
		opacity: 0.8;
	}

	.feedback {
		margin-top: 10px;
	}

	.hiddenInput {
		display: none;
	}
}
</style>
