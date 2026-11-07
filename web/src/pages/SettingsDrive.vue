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
				<CfKeyValue class="kv">
					<template #key>{{ t("capacity") }}</template>
					<template #value>
						{{
							capacity === 0
								? "未设置配额（无限制）"
								: bytes(capacity, 1)
						}}
					</template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>{{ t("inUse") }}</template>
					<template #value>{{ bytes(usedBytes, 1) }}</template>
				</CfKeyValue>
				<CfKeyValue class="kv">
					<template #key>{{ t("driveFilesCount") }}</template>
					<template #value>{{ fileCount }}</template>
				</CfKeyValue>
				<CfInfo class="note">
					数据来自 <code>drive</code> 端点。容量为 0 表示本实例没有配置配额，
					上传只受单文件大小限制。
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
					关闭时浏览器会在上传前把 JPEG / WebP 重编码到最长边 2048、质量 0.85；
					打开时上传原始字节。其它类型（PNG、GIF、SVG 等）始终按原样上传。
				</template>
			</CfSwitch>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("drive") }}</h2>
			<div class="kvRow">
				<span class="kvKey">{{ t("alwaysMarkSensitive") }}</span>
				<CfSwitch
					:model-value="me?.alwaysMarkNsfw ?? false"
					@update:model-value="saveDefaultSensitive"
				/>
			</div>
			<div class="kvRow">
				<span class="kvKey">{{ t("enableAutoSensitive") }}</span>
				<CfSwitch
					:model-value="me?.autoSensitive ?? false"
					@update:model-value="saveAutoSensitive"
				/>
			</div>
			<CfInfo v-if="profileErrorText != null" class="note" warn>
				{{ profileErrorText }}
			</CfInfo>
			<CfInfo v-else class="note">
				这两项由 <code>i/update</code> 写入个人资料；网盘统计图表
				（<code>MkChart</code> / chart.js）仍未提供。
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
//   * `drive` answers capacity/usage/count; capacity 0 means this instance has
//     no quota configured (the UI says so instead of drawing a fake meter);
//   * no chart library → the per-user-drive histogram is dropped;
//   * `i/update` writes `alwaysMarkNsfw` / `autoSensitive`; both are real switches;
//   * `drive/folders/show` exists, so the configured folder can be resolved to a
//     name, and the folder picker reuses the wave-2 `MkDrive` in select mode.
import { computed, onMounted, ref } from "vue";
import { me, updateProfile } from "@/account";
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

const loading = ref(true);
const errorText = ref<string | null>(null);
const capacity = ref(0);
const usedBytes = ref(0);
const fileCount = ref(0);
const profileErrorText = ref<string | null>(null);

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
	capacity.value = 0;
	try {
		const usage = await api("drive");
		capacity.value = usage.capacity;
		usedBytes.value = usage.usage;
		fileCount.value = usage.filesCount;
	} catch (error) {
		errorText.value = describe(error);
	} finally {
		loading.value = false;
	}
}

async function saveDefaultSensitive(value: boolean): Promise<void> {
	profileErrorText.value = null;
	try {
		await updateProfile({ alwaysMarkNsfw: value });
	} catch (error) {
		profileErrorText.value = describe(error);
	}
}

async function saveAutoSensitive(value: boolean): Promise<void> {
	profileErrorText.value = null;
	try {
		await updateProfile({ autoSensitive: value });
	} catch (error) {
		profileErrorText.value = describe(error);
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
