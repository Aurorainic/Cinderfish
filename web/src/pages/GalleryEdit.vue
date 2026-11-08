<!-- Ported from firefish@b425f61:packages/client/src/pages/gallery/edit.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="[]" :tabs="[]" :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-gallery-editor">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<template v-else>
					<CfInput v-model="title" class="_formBlock">
						<template #label>{{ t("title") }}</template>
					</CfInput>

					<CfTextarea
						v-model="description"
						class="_formBlock"
						:placeholder="t('optional')"
					>
						<template #label>{{ t("description") }}</template>
						<template #caption>
							{{ description.length }} / {{ DESCRIPTION_MAX }}
						</template>
					</CfTextarea>

					<div class="_formBlock filesBlock">
						<div class="label">{{ t("attachFile") }}</div>
						<div v-if="files.length > 0" class="files">
							<div
								v-for="file in files"
								:key="file.id"
								class="file"
								:style="fileStyle(file)"
							>
								<div class="name">{{ file.name }}</div>
								<button
									v-tooltip="t('remove')"
									class="remove _button"
									:aria-label="t('remove')"
									@click="removeFile(file)"
								>
									<i class="ph-x ph-bold ph-lg"></i>
								</button>
							</div>
						</div>
						<CfButton
							:disabled="saving || files.length >= FILES_MAX"
							@click="pickFiles"
						>
							<i class="ph-plus ph-bold ph-lg"></i>
							{{ t("attachFile") }}
						</CfButton>
						<div class="caption">
							{{ files.length }} / {{ FILES_MAX }}
						</div>
					</div>

					<CfSwitch v-model="isSensitive" class="_formBlock">
						{{ t("markAsSensitive") }}
					</CfSwitch>

					<div class="actions _formBlock">
						<CfButton
							primary
							:disabled="
								saving ||
								title.trim() === '' ||
								files.length === 0
							"
							@click="save"
						>
							<i class="ph-floppy-disk-back ph-bold ph-lg"></i>
							{{ isEdit ? t("save") : t("publish") }}
						</CfButton>
						<CfButton
							v-if="isEdit"
							danger
							:disabled="saving || deleting"
							@click="remove"
						>
							<i class="ph-trash ph-bold ph-lg"></i>
							{{ t("delete") }}
						</CfButton>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's gallery editor: title, a 500-character description, the attached
// drive files in order, the sensitive flag and create / save / delete. The port
// keeps Firefish's `selectFiles` path (the drive picker or an upload), which is
// the same helper the channels editor and the post form use. `fileIds` is sent
// in list order — the backend preserves it and drops ids the caller does not
// own. An empty description is sent as null so `gallery/posts/update` clears a
// stored value instead of leaving it untouched.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { router } from "@/router";
import { definePageMetadata } from "@/scripts/page-metadata";
import { selectFiles } from "@/scripts/select-file";

defineOptions({ name: "CfPageGalleryEdit" });

/** Firefish's textarea cap. */
const DESCRIPTION_MAX = 500;
/** gallery/posts/create's fileIds cap. */
const FILES_MAX = 32;

const route = useRoute();

const postId = computed<string | null>(() => {
	const value = route.params.postId;
	return typeof value === "string" && value !== "" ? value : null;
});
const isEdit = computed(() => postId.value != null);

const loading = ref(false);
const loadError = ref<string | null>(null);
const saving = ref(false);
const deleting = ref(false);

const title = ref("");
const description = ref("");
const isSensitive = ref(false);
const files = ref<DriveFile[]>([]);

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

function fileStyle(file: DriveFile): Record<string, string> {
	const url = file.thumbnailUrl ?? file.url;
	return url === "" ? {} : { backgroundImage: `url(${url})` };
}

async function load(): Promise<void> {
	if (postId.value == null) return;
	loading.value = true;
	loadError.value = null;
	try {
		const post = await api(ENDPOINTS.galleryPostsShow, {
			postId: postId.value,
		});
		title.value = post.title;
		description.value = post.description ?? "";
		isSensitive.value = post.isSensitive;
		files.value = [...post.files];
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function pickFiles(event: MouseEvent): Promise<void> {
	const picked = await selectFiles({
		multiple: true,
		label: t("attachFile"),
		src: event,
	});
	if (picked.length === 0) return;
	const known = new Set(files.value.map((file) => file.id));
	const added = picked.filter((file) => !known.has(file.id));
	files.value = [...files.value, ...added].slice(0, FILES_MAX);
}

function removeFile(file: DriveFile): void {
	files.value = files.value.filter((item) => item.id !== file.id);
}

async function save(): Promise<void> {
	const trimmedTitle = title.value.trim();
	if (
		saving.value ||
		trimmedTitle === "" ||
		files.value.length === 0
	) {
		return;
	}
	const trimmedDescription = description.value.trim();
	const descriptionParam =
		trimmedDescription === "" ? null : trimmedDescription;
	const fileIds = files.value.map((file) => file.id);
	saving.value = true;
	try {
		if (postId.value != null) {
			const updated = await api(ENDPOINTS.galleryPostsUpdate, {
				postId: postId.value,
				title: trimmedTitle,
				description: descriptionParam,
				fileIds,
				isSensitive: isSensitive.value,
			});
			os.success();
			await router.push(`/gallery/${updated.id}`);
		} else {
			const created = await api(ENDPOINTS.galleryPostsCreate, {
				title: trimmedTitle,
				description: descriptionParam,
				fileIds,
				isSensitive: isSensitive.value,
			});
			os.success();
			await router.push(`/gallery/${created.id}`);
		}
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function remove(): Promise<void> {
	if (postId.value == null || deleting.value) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteConfirm"),
	});
	if (canceled) return;
	deleting.value = true;
	try {
		await api(ENDPOINTS.galleryPostsDelete, { postId: postId.value });
		await router.push("/gallery");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		deleting.value = false;
	}
}

// The textarea has no maxlength prop, so the cap is applied on the bound value.
watch(description, (value) => {
	if (value.length > DESCRIPTION_MAX) {
		description.value = value.slice(0, DESCRIPTION_MAX);
	}
});

watch(postId, () => void load(), { immediate: true });

definePageMetadata(
	computed(() => ({
		title: isEdit.value ? t("edit") : t("postToGallery"),
		icon: "ph-pencil ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-gallery-editor {
	.filesBlock {
		> .label {
			margin-block-end: 8px;
			font-weight: bold;
		}

		> .files {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
			margin-block-end: 12px;
		}

		> .caption {
			margin-block-start: 6px;
			font-size: 0.82em;
			color: var(--fgTransparentWeak);
		}
	}

	.file {
		position: relative;
		flex: 0 0 200px;
		block-size: 140px;
		background-color: var(--panelHighlight);
		background-position: center;
		background-repeat: no-repeat;
		background-size: contain;
		border: solid 1px var(--divider);
		border-radius: var(--radius);
		overflow: hidden;

		> .name {
			position: absolute;
			inset-block-start: 8px;
			inset-inline-start: 8px;
			max-inline-size: calc(100% - 48px);
			padding: 4px 8px;
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
			font-size: 0.82em;
			background: var(--panel);
			border-radius: 4px;
		}

		> .remove {
			position: absolute;
			inset-block-start: 8px;
			inset-inline-end: 8px;
			padding: 4px;
			color: #fff;
			background: rgba(0, 0, 0, 0.5);
			border-radius: 4px;
		}
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
}
</style>
