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
					导入还没有端点：需要先定义文件格式、冲突规则（同 id 的帖子如何处理）与限流，
					所以这里不提供会失败的导入按钮。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish splits this page into per-dataset exports (`i/export-notes`,
// `i/export-following`, …) delivered asynchronously through jobs. Cinderfish
// answers `i/export` synchronously with one JSON document, which is small
// enough for a personal instance and can be opened with any editor. Import is
// deliberately absent until a format and conflict policy exist.
import { ref } from "vue";
import { ApiError, apiDownload } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
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
}
</style>
