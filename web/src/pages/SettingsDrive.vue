<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/drive.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-drive">
		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("usageAmount") }}</h2>

			<div v-if="loading" class="state"><CfLoading /></div>
			<div v-else-if="errorText != null" class="state">
				<CfInfo warn>{{ errorText }}</CfInfo>
				<CfButton class="retry" inline @click="loadUsage()">
					{{ t("retry") }}
				</CfButton>
			</div>
			<template v-else>
				<div v-if="fileCount === 0" class="empty">{{ t("emptyDrive") }}</div>
				<CfKeyValue v-else class="kv">
					<template #key>{{ t("inUse") }}</template>
					<template #value>{{ bytes(usedBytes, 1) }}</template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>{{ t("driveFilesCount") }}</template>
					<template #value>
						{{ truncated ? `≥ ${fileCount}` : fileCount }}
					</template>
				</CfKeyValue>
				<CfInfo class="note">
					容量与配额来自 <code>drive</code> 端点，本实例<strong>未提供</strong>，
					所以没有容量条；上面的用量是按 <code>drive/files</code> 逐页汇总的真实大小。
					<template v-if="truncated">
						文件过多，只统计了前 {{ scannedPages * 100 }} 个。
					</template>
				</CfInfo>
			</template>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("uploadFolder") }}</h2>
			<div class="row">
				<CfButton @click="openFolderPicker()">
					<i class="ph-folder-notch-open ph-bold ph-lg"></i>
					{{ folderLoading ? "…" : folderName ?? t("none") }}
				</CfButton>
				<CfButton v-if="uploadFolder != null" inline @click="clearFolder()">
					{{ t("remove") }}
				</CfButton>
			</div>
			<p class="caption">
				上传文件与新建文件夹会默认落在这个文件夹；留空则使用网盘根目录。
			</p>
			<CfInfo v-if="folderErrorText != null" class="note" :warn="true">
				{{ folderErrorText }}
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("clientSettings") }}</h2>
			<CfSwitch v-model="keepOriginalUploading" class="_formBlock">
				{{ t("keepOriginalUploading") }}
				<template #caption>
					{{ t("keepOriginalUploadingDescription") }}
					当前上传管线直接把原文件写入存储，不做压缩，因此该偏好已保存但暂不改变上传结果。
				</template>
			</CfSwitch>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("drive") }}</h2>
			<div class="kvRow">
				<span class="kvKey">{{ t("alwaysMarkSensitive") }}</span>
				<span class="kvValue">{{ me?.alwaysMarkNsfw ? t("yes") : t("no") }}</span>
			</div>
			<div class="kvRow">
				<span class="kvKey">{{ t("enableAutoSensitive") }}</span>
				<span class="kvValue">{{ me?.autoSensitive ? t("yes") : t("no") }}</span>
			</div>
			<CfInfo class="note">
				这两项由 <code>i/update</code> 写入，但后端只接受 name / description / avatarId /
				followersVisibility / followingVisibility，所以这里只读显示，修改<strong>未提供</strong>。
				网盘统计图表（<code>MkChart</code> / chart.js）同样未提供。
			</CfInfo>
		</section>

		<CfModal
			v-if="pickerShowing"
			:close-on-backdrop="true"
			@closed="pickerShowing = false"
		>
			<div class="_popup _shadow picker" @click.stop>
				<div class="pickerHeader">
					<span>{{ t("uploadFolder") }}</span>
					<button class="_button closeButton" @click="pickerShowing = false">
						<i class="ph-x ph-bold ph-lg"></i>
					</button>
				</div>
				<div class="pickerBody">
					<MkDrive
						:multiple="false"
						select="folder"
						@selected="onFolderSelected"
					/>
				</div>
			</div>
		</CfModal>
	</div>
</template>

<script lang="ts" setup>
// Firefish reads `drive` for capacity/usage and shows an MkChart histogram, then
// lets the user pick a default upload folder and toggles the profile's NSFW
// flags. Cinderfish's contract (docs/research/api-contract/drive-media.md):
//   * no `drive` endpoint → capacity/quota is unavailable; usage is summed from
//     `drive/files` pages instead of being fabricated;
//   * no chart library → the per-user-drive histogram is dropped;
//   * `i/update` ignores `alwaysMarkNsfw` / `autoSensitive` → read-only display;
//   * `drive/folders/show` exists, so the configured folder can be resolved to a
//     name, and the folder picker reuses the wave-2 `MkDrive` in select mode.
import { computed, onMounted, ref } from "vue";
import { me } from "@/account";
import type { DriveFile, DriveFolder } from "@/api/types";
import { api } from "@/api/client";
import MkDrive from "@/components/MkDrive.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import bytes from "@/filters/bytes";
import { t } from "@/i18n";
import CfModal from "@/os/CfModal.vue";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsDrive" });

const MAX_PAGES = 10;
const PAGE_SIZE = 100;

const loading = ref(true);
const errorText = ref<string | null>(null);
const usedBytes = ref(0);
const fileCount = ref(0);
const scannedPages = ref(0);
const truncated = ref(false);

const folderLoading = ref(false);
const folderName = ref<string | null>(null);
const folderErrorText = ref<string | null>(null);
const pickerShowing = ref(false);

const uploadFolder = computed<string | null>(
	() => defaultStore.reactiveState.uploadFolder.value,
);

const keepOriginalUploading = computed<boolean>({
	get: () => defaultStore.reactiveState.keepOriginalUploading.value,
	set: (value) => defaultStore.set("keepOriginalUploading", value),
});

function describe(error: unknown): string {
	return error instanceof Error && error.message !== ""
		? error.message
		: t("somethingHappened");
}

async function loadUsage(): Promise<void> {
	loading.value = true;
	errorText.value = null;
	usedBytes.value = 0;
	fileCount.value = 0;
	scannedPages.value = 0;
	truncated.value = false;

	try {
		let untilId: string | undefined;
		for (let page = 0; page < MAX_PAGES; page++) {
			const files = await api("drive/files", {
				limit: PAGE_SIZE,
				...(untilId == null ? {} : { untilId }),
			});
			scannedPages.value = page + 1;
			if (files.length === 0) break;
			usedBytes.value += files.reduce((sum, file) => sum + file.size, 0);
			fileCount.value += files.length;
			if (files.length < PAGE_SIZE) break;
			untilId = files[files.length - 1]?.id;
			if (page === MAX_PAGES - 1) truncated.value = true;
			if (untilId == null) break;
		}
	} catch (error) {
		errorText.value = describe(error);
	} finally {
		loading.value = false;
	}
}

async function loadFolderName(folderId: string | null): Promise<void> {
	folderName.value = null;
	folderErrorText.value = null;
	if (folderId == null) return;
	folderLoading.value = true;
	try {
		const folder = await api("drive/folders/show", { folderId });
		folderName.value = folder.name;
	} catch (error) {
		// A folder deleted from another tab leaves a dangling preference: drop it
		// instead of showing an error the user cannot act on.
		if (
			error != null &&
			typeof error === "object" &&
			"code" in error &&
			(error as { code?: string }).code === "NO_SUCH_FOLDER"
		) {
			defaultStore.set("uploadFolder", null);
		} else {
			folderErrorText.value = describe(error);
		}
	} finally {
		folderLoading.value = false;
	}
}

function openFolderPicker(): void {
	pickerShowing.value = true;
}

function onFolderSelected(target: DriveFile | DriveFolder): void {
	if (!("parentId" in target)) return;
	defaultStore.set("uploadFolder", target.id);
	folderName.value = target.name;
	pickerShowing.value = false;
}

function clearFolder(): void {
	defaultStore.set("uploadFolder", null);
	folderName.value = null;
}

onMounted(() => {
	void loadUsage();
	void loadFolderName(uploadFolder.value);
});

definePageMetadata({ title: t("drive"), icon: "ph-cloud ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-drive {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.state {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12px;
		margin-block-start: 12px;
	}

	.empty {
		margin-block-start: 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.kv {
		margin-block-start: 8px;
	}

	.kvRow {
		display: flex;
		align-items: baseline;
		gap: 12px;
		margin-block-start: 12px;

		> .kvKey {
			flex: 0 0 35%;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
		}

		> .kvValue {
			font-size: 0.9em;
		}
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 12px;
	}

	.caption {
		margin: 8px 0 0;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.note {
		margin-block-start: 12px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}

	.picker {
		display: flex;
		flex-direction: column;
		inline-size: min(800px, calc(100vw - 32px));
		block-size: min(560px, calc(100vh - 64px));
		background: var(--panel);
		border-radius: var(--radius);
		overflow: hidden;

		> .pickerHeader {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: 12px 16px;
			border-block-end: solid 0.5px var(--divider);
			font-weight: bold;

			> .closeButton {
				padding: 4px;
				border-radius: var(--radius);
				cursor: pointer;
			}
		}

		> .pickerBody {
			flex: 1;
			min-block-size: 0;
			overflow: auto;
		}
	}
}
</style>
