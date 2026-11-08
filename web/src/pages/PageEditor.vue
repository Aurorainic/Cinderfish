<!-- Ported from firefish@b425f61:packages/client/src/pages/page-editor/page-editor.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="[]" :tabs="[]" :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<CfLoading v-if="loading" />
			<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
				{{ loadError }}
			</CfInfo>

			<!-- The /view-source route loads the page read-only and prints the
			     stored document instead of mounting the editor. -->
			<div v-else-if="readonly" class="_formRoot cf-page-source">
				<CfInfo class="_formBlock">
					Read-only source view. The document below is exactly what the
					server stores, including any dynamic blocks, variables and
					AiScript that a Firefish client wrote.
				</CfInfo>
				<div class="_formBlock">
					<div class="sourceLabel">Page source</div>
					<pre class="source">{{ sourceJson }}</pre>
				</div>
			</div>

			<template v-else>
				<div class="toolbar _formBlock">
					<CfButton v-if="isExisting" link :to="viewPath">
						<i class="ph-arrow-square-out ph-bold ph-lg"></i>
						View page
					</CfButton>
					<CfButton primary :disabled="saving" @click="save">
						<i class="ph-floppy-disk-back ph-bold ph-lg"></i>
						{{ t("save") }}
					</CfButton>
					<CfButton
						v-if="isExisting"
						:disabled="saving || deleting"
						@click="duplicate"
					>
						<i class="ph-copy ph-bold ph-lg"></i>
						{{ t("duplicate") }}
					</CfButton>
					<CfButton
						v-if="isExisting"
						danger
						:disabled="saving || deleting"
						@click="remove"
					>
						<i class="ph-trash ph-bold ph-lg"></i>
						{{ t("delete") }}
					</CfButton>
				</div>

				<CfTab v-model="tab" :tabs="tabItems" class="tabs" />

				<!-- settings -->
				<div v-if="tab === 'settings'" class="_formRoot">
					<CfInput v-model="title" class="_formBlock">
						<template #label>{{ t("title") }}</template>
						<template #caption>{{ title.length }} / 256</template>
					</CfInput>

					<CfInput v-model="name" class="_formBlock">
						<template #label>URL slug</template>
						<template #prefix>/@{{ meUsername }}/pages/</template>
						<template #caption>
							Unique per account, up to 256 characters. Spaces and
							<code>: / ? # [ ] @ ! $ &amp; ' ( ) * + , ; = \ %</code>
							are not allowed.
						</template>
					</CfInput>

					<CfInput v-model="summary" class="_formBlock">
						<template #label>Summary</template>
						<template #caption>Optional; shown in page previews.</template>
					</CfInput>

					<CfSelect
						:model-value="font"
						class="_formBlock"
						@update:model-value="onFont"
					>
						<template #label>Font</template>
						<option value="sans-serif">Sans-serif</option>
						<option value="serif">Serif</option>
					</CfSelect>

					<CfSwitch v-model="alignCenter" class="_formBlock">
						Center content
					</CfSwitch>

					<CfSwitch v-model="hideTitleWhenPinned" class="_formBlock">
						Hide title when pinned
					</CfSwitch>

					<div class="_formBlock">
						<div class="fieldLabel">Eye-catching image</div>
						<div v-if="eyeCatchingImage != null" class="eyeCatch">
							<MkImgWithBlurhash
								:src="resolveMediaUrl(eyeCatchingImage.url)"
								:hash="eyeCatchingImage.blurhash"
								:alt="eyeCatchingImage.name"
								:cover="false"
							/>
							<CfButton @click="removeEyeCatch">
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("remove") }}
							</CfButton>
						</div>
						<CfButton v-else @click="pickEyeCatch">
							<i class="ph-plus ph-bold ph-lg"></i>
							Set image
						</CfButton>
					</div>
				</div>

				<!-- contents -->
				<div v-else-if="tab === 'contents'" class="_formRoot">
					<CfInfo class="_formBlock">
						Only text, section, image and note blocks are offered,
						matching the pinned upstream editor. Dynamic blocks written
						by Firefish are preserved on save and shown as removable
						placeholders; the variables and AiScript tabs are not
						offered because this editor always sends
						<code>script: ''</code> and <code>variables: []</code>.
					</CfInfo>
					<PageBlockEditor
						:blocks="content"
						:attached-files="attachedFiles"
						:depth="1"
						@file-picked="onFilePicked"
					/>
				</div>
			</template>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's page editor with the dynamic tabs removed. The pinned upstream
// client's editor offers exactly the four static block types and always sends
// `script: ''` and `variables: []`; the backend stores content/variables/script
// opaquely, so a Firefish-authored dynamic page still round-trips (and is shown
// in the read-only /view-source route). The draggable reordering is replaced by
// explicit up/down buttons because vue-draggable-plus is outside the port
// contract, and `crypto.randomUUID()` supplies the local block keys instead of
// the `uuid` package.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile, Page, PageBlock, PageFont } from "@/api/types";
import { me } from "@/account";
import MkImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";
import PageBlockEditor from "@/components/page/PageBlockEditor.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import type { CfTabItem } from "@/components/ui/CfTab.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTab from "@/components/ui/CfTab.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { router } from "@/router";
import { definePageMetadata } from "@/scripts/page-metadata";
import { selectFile } from "@/scripts/select-file";

defineOptions({ name: "CfPagePageEditor" });

const route = useRoute();

function routeParam(key: string): string {
	const value = route.params[key];
	return typeof value === "string" ? value : "";
}

const pageId = computed<string>(() => routeParam("initPageId"));
const sourceUser = computed<string>(() => routeParam("initUser"));
const sourceName = computed<string>(() => routeParam("initPageName"));

const isExisting = computed(() => pageId.value !== "");
const readonly = computed(
	() => sourceUser.value !== "" && sourceName.value !== "",
);

const loading = ref(false);
const loadError = ref<string | null>(null);
const saving = ref(false);
const deleting = ref(false);

const tab = ref<"settings" | "contents">("settings");

const title = ref("");
const name = ref("");
const summary = ref("");
const font = ref<PageFont>("sans-serif");
const alignCenter = ref(false);
const hideTitleWhenPinned = ref(false);
const eyeCatchingImageId = ref<string | null>(null);
const eyeCatchingImage = ref<DriveFile | null>(null);
const content = ref<PageBlock[]>([]);
const attachedFiles = ref<DriveFile[]>([]);
/** Stored originals, shown by the read-only source view and never edited. */
const variables = ref<unknown[]>([]);
const script = ref("");
const current = ref<Page | null>(null);

const meUsername = computed(() => me.value?.username ?? "…");

const tabItems = computed<CfTabItem[]>(() => [
	{ value: "settings", label: "Settings", icon: "ph-gear-six" },
	{ value: "contents", label: "Contents", icon: "ph-sticker" },
]);

const viewPath = computed<string>(() => {
	const loaded = current.value;
	if (loaded == null) return "/pages";
	return `/@${loaded.user.username}/pages/${loaded.name}`;
});

const sourceJson = computed(() =>
	JSON.stringify(
		{
			content: content.value,
			variables: variables.value,
			script: script.value,
		},
		null,
		2,
	),
);

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

function normalizeBlock(value: unknown): PageBlock | null {
	if (value == null || typeof value !== "object" || Array.isArray(value)) {
		return null;
	}
	const block = value as PageBlock;
	if (typeof block.type !== "string") return null;
	if (block.type === "section") {
		const children = Array.isArray(block.children) ? block.children : [];
		block.children = children
			.map(normalizeBlock)
			.filter((child): child is PageBlock => child != null);
	} else if (block.type === "text") {
		if (typeof block.text !== "string") block.text = "";
	} else if (block.type === "image") {
		if (typeof block.fileId !== "string") block.fileId = "";
	} else if (block.type === "note") {
		if (typeof block.note !== "string") block.note = "";
	}
	return block;
}

function normalizeContent(value: unknown): PageBlock[] {
	if (!Array.isArray(value)) return [];
	return value
		.map(normalizeBlock)
		.filter((block): block is PageBlock => block != null);
}

function applyPage(loaded: Page): void {
	current.value = loaded;
	title.value = loaded.title;
	name.value = loaded.name;
	summary.value = loaded.summary ?? "";
	font.value = loaded.font === "serif" ? "serif" : "sans-serif";
	alignCenter.value = loaded.alignCenter;
	hideTitleWhenPinned.value = loaded.hideTitleWhenPinned;
	eyeCatchingImageId.value = loaded.eyeCatchingImageId;
	eyeCatchingImage.value = loaded.eyeCatchingImage;
	content.value = normalizeContent(loaded.content);
	attachedFiles.value = [...loaded.attachedFiles];
	variables.value = loaded.variables;
	script.value = loaded.script;
}

function resetNew(): void {
	current.value = null;
	title.value = "";
	name.value = String(Date.now());
	summary.value = "";
	font.value = "sans-serif";
	alignCenter.value = false;
	hideTitleWhenPinned.value = false;
	eyeCatchingImageId.value = null;
	eyeCatchingImage.value = null;
	content.value = [{ id: crypto.randomUUID(), type: "text", text: "" }];
	attachedFiles.value = [];
	variables.value = [];
	script.value = "";
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		if (isExisting.value) {
			applyPage(
				await api(ENDPOINTS.pagesShow, { pageId: pageId.value }),
			);
		} else if (readonly.value) {
			applyPage(
				await api(ENDPOINTS.pagesShow, {
					name: sourceName.value,
					username: sourceUser.value,
				}),
			);
		} else {
			resetNew();
		}
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

function onFont(value: string | null): void {
	font.value = value === "serif" ? "serif" : "sans-serif";
}

function createParams(): {
	title: string;
	name: string;
	summary: string | null;
	eyeCatchingImageId: string | null;
	font: PageFont;
	alignCenter: boolean;
	hideTitleWhenPinned: boolean;
	content: PageBlock[];
	variables: unknown[];
	script: string;
} {
	const trimmedSummary = summary.value.trim();
	return {
		title: title.value.trim(),
		name: name.value.trim(),
		summary: trimmedSummary === "" ? null : trimmedSummary,
		eyeCatchingImageId: eyeCatchingImageId.value,
		font: font.value,
		alignCenter: alignCenter.value,
		hideTitleWhenPinned: hideTitleWhenPinned.value,
		content: content.value,
		// Upstream's editor sends these empty; the backend treats both as
		// opaque and the viewer never evaluates them.
		variables: [],
		script: "",
	};
}

async function save(): Promise<void> {
	if (saving.value) return;
	if (title.value.trim() === "" || name.value.trim() === "") {
		os.toast("Title and slug are required.", "error");
		return;
	}
	saving.value = true;
	try {
		if (isExisting.value) {
			await api(ENDPOINTS.pagesUpdate, {
				pageId: pageId.value,
				...createParams(),
			});
			os.success();
			// Reload so the header link follows a renamed slug.
			await load();
		} else {
			const created = await api(ENDPOINTS.pagesCreate, createParams());
			os.success();
			await router.replace(`/pages/edit/${created.id}`);
		}
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function duplicate(): Promise<void> {
	if (!isExisting.value || saving.value) return;
	saving.value = true;
	try {
		const base = createParams();
		const created = await api(ENDPOINTS.pagesCreate, {
			...base,
			title: `${base.title} - copy`,
			name: `${base.name}-copy`,
		});
		os.success();
		await router.replace(`/pages/edit/${created.id}`);
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function remove(): Promise<void> {
	if (!isExisting.value || deleting.value) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteConfirm"),
	});
	if (canceled) return;
	deleting.value = true;
	try {
		await api(ENDPOINTS.pagesDelete, { pageId: pageId.value });
		os.success();
		await router.replace("/pages");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		deleting.value = false;
	}
}

async function pickEyeCatch(): Promise<void> {
	const file = await selectFile({ label: "Eye-catching image" });
	if (file == null) return;
	eyeCatchingImageId.value = file.id;
	eyeCatchingImage.value = file;
}

function removeEyeCatch(): void {
	eyeCatchingImageId.value = null;
	eyeCatchingImage.value = null;
}

function onFilePicked(file: DriveFile): void {
	if (!attachedFiles.value.some((item) => item.id === file.id)) {
		attachedFiles.value.push(file);
	}
}

watch([pageId, sourceUser, sourceName], () => void load(), { immediate: true });

definePageMetadata(
	computed(() => ({
		title: readonly.value
			? "Page source"
			: isExisting.value
				? t("edit")
				: "New page",
		icon: "ph-pencil ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.toolbar {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: 8px;
}

.tabs {
	margin-block: var(--margin);
}

.fieldLabel {
	margin-block-end: 8px;
	font-weight: bold;
}

.eyeCatch {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 8px;

	img {
		display: block;
		max-inline-size: 100%;
		max-block-size: 240px;
		object-fit: contain;
	}
}

.cf-page-source {
	.sourceLabel {
		margin-block-end: 8px;
		font-weight: bold;
	}

	.source {
		margin: 0;
		padding: 16px;
		overflow: auto;
		font-size: 0.82em;
		white-space: pre;
		background: var(--panelHighlight);
		border-radius: var(--radius);
	}
}
</style>
