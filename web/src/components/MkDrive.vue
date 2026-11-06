<!-- Ported from firefish@b425f61:packages/client/src/components/MkDrive.vue (AGPL-3.0-only) -->
<template>
	<div class="yfudmmck">
		<nav>
			<div class="path" @contextmenu.prevent.stop="() => {}">
				<MkDriveNavFolder
					:class="{ current: folder == null }"
					:parent-folder="folder"
					@move="move"
					@upload="upload"
					@updated="replaceFolder"
				/>
				<template v-for="f in hierarchyFolders" :key="f.id">
					<span class="separator"
						><i class="ph-caret-right ph-bold ph-lg"></i
					></span>
					<MkDriveNavFolder
						:folder="f"
						:parent-folder="folder"
						@move="move"
						@upload="upload"
						@updated="replaceFolder"
					/>
				</template>
				<span v-if="folder != null" class="separator"
					><i class="ph-caret-right ph-bold ph-lg"></i
				></span>
				<span v-if="folder != null" class="folder current">{{
					folder.name
				}}</span>
			</div>
			<button class="menu _button" @click="showMenu">
				<i class="ph-dots-three-outline ph-bold ph-lg"></i>
			</button>
		</nav>
		<div
			class="main"
			:class="{ uploading: uploads.length > 0, fetching }"
			@dragover.prevent.stop="onDragover"
			@dragenter="onDragenter"
			@dragleave="onDragleave"
			@drop.prevent.stop="onDrop"
			@contextmenu.stop="onContextmenu"
		>
			<div class="contents">
				<div v-show="folders.length > 0" class="folders">
					<MkDriveFolder
						v-for="f in folders"
						:key="f.id"
						class="folder"
						:folder="f"
						:select-mode="select === 'folder'"
						:is-selected="
							selectedFolders.some((x) => x.id === f.id)
						"
						@chosen="chooseFolder"
						@move="move"
						@upload="upload"
						@updated="replaceFolder"
						@deleted="onFolderDeleted"
						@file-moved="removeFile"
						@dragstart="isDragSource = true"
						@dragend="isDragSource = false"
					/>
					<!-- SEE: https://stackoverflow.com/questions/18744164/flex-box-align-last-row-to-grid -->
					<div v-for="n in 16" :key="n" class="padding"></div>
					<CfButton v-if="moreFolders" @click="fetchMoreFolders">{{
						t("loadMore")
					}}</CfButton>
				</div>
				<CfPagination
					ref="filesPaginationRef"
					:pagination="filesPagination"
				>
					<template #default="{ items }">
						<div v-show="items.length > 0" class="files">
							<MkDriveFile
								v-for="file in (items as DriveFile[])"
								:key="file.id"
								class="file"
								:file="file"
								:select-mode="select === 'file'"
								:is-selected="
									selectedFiles.some((x) => x.id === file.id)
								"
								@chosen="chooseFile"
								@updated="replaceFile"
								@deleted="removeFile"
							/>
							<!-- SEE: https://stackoverflow.com/questions/18744164/flex-box-align-last-row-to-grid -->
							<div v-for="n in 16" :key="n" class="padding"></div>
						</div>
					</template>
					<template #empty>
						<div
							v-if="folders.length == 0 && !fetching"
							class="empty"
						>
							<p v-if="draghover">{{ t("upload") }}</p>
							<p v-else-if="folder == null">
								<strong>{{ t("emptyDrive") }}</strong>
							</p>
							<p v-else>{{ t("emptyFolder") }}</p>
						</div>
					</template>
				</CfPagination>
			</div>
			<CfLoading v-if="fetching" />
		</div>
		<div v-if="draghover" class="dropzone"></div>
		<input
			ref="fileInput"
			type="file"
			accept="*/*"
			multiple
			tabindex="-1"
			@change="onChangeFileInput"
		/>
	</div>
</template>

<script lang="ts" setup>
// Drive browser, ported from Firefish's MkDrive.vue.
//
// Wave-2 differences from upstream (contract §5 + docs/research/api-contract/drive-media.md):
//   * no drive streaming channel exists, so folders/files are re-fetched after
//     every mutation instead of waiting for fileCreated/folderDeleted events;
//   * the file list is a CfPagination over drive/files (the folders list keeps
//     upstream's own "load more" button, since CfPagination can only page one
//     endpoint and the grid mixes two);
//   * `drive/files/update` backs rename / description / sensitive / folder
//     moves; files are draggable onto folder tiles and file tiles refresh in
//     place through the pagination's updateItem;
//   * folder moves go through drive/folders/update; because the packed folder
//     never carries a nested `parent`, the breadcrumb walks parentId with one
//     drive/folders/show per level;
//   * "upload from URL" calls drive/files/upload-from-url (the server refuses
//     private/loopback targets);
//   * the "keep original uploading" switch is gone with the recompression
//     pipeline (browser-image-resizer is not a dependency).
import { computed, onMounted, ref, watch } from "vue";
import { api } from "@/api/client";
import type { DriveFile, DriveFolder } from "@/api/types";
import CfButton from "@/components/ui/CfButton.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import type { Paging } from "@/components/ui/pagination";
import MkDriveFolder from "@/components/MkDrive.folder.vue";
import MkDriveFile from "@/components/MkDrive.file.vue";
import MkDriveNavFolder from "@/components/MkDrive.navFolder.vue";
import { ApiError } from "@/api/client";
import { alert, contextMenu, inputText, popupMenu, success, type MenuItem } from "@/os";
import { t } from "@/i18n";
import { defaultStore } from "@/store/base";
import { uploadFile, uploads } from "@/scripts/upload";

// Firefish injects these through vite `define`; the string values are kept.
const DATA_TRANSFER_DRIVE_FOLDER = "mk_drive_folder";

const FOLDERS_MAX = 30;
const FILES_MAX = 30;

const props = withDefaults(
	defineProps<{
		initialFolder?: DriveFolder | null;
		/** Route deep link: /my/drive/folder/:folder carries only the id. */
		initialFolderId?: string | null;
		multiple?: boolean;
		select?: "file" | "folder" | null;
	}>(),
	{
		initialFolder: null,
		initialFolderId: null,
		multiple: false,
		select: null,
	},
);

const emit = defineEmits<{
	(ev: "selected", v: DriveFile | DriveFolder): void;
	(ev: "change-selection", v: DriveFile[] | DriveFolder[]): void;
	(ev: "move-root"): void;
	(ev: "cd", v: DriveFolder | null): void;
	(ev: "open-folder", v: DriveFolder): void;
}>();

/** Methods CfPagination exposes through defineExpose. */
interface FilesPaginationHandle {
	reload: () => Promise<unknown>;
	prepend: (...items: DriveFile[]) => void;
	removeItem: (finder: (item: DriveFile) => boolean) => boolean;
	updateItem: (id: string, updater: (old: DriveFile) => DriveFile) => void;
}

const fileInput = ref<HTMLInputElement | null>(null);
const filesPaginationRef = ref<unknown>(null);

const folder = ref<DriveFolder | null>(null);
const folders = ref<DriveFolder[]>([]);
const moreFolders = ref(false);
const hierarchyFolders = ref<DriveFolder[]>([]);
const selectedFiles = ref<DriveFile[]>([]);
const selectedFolders = ref<DriveFolder[]>([]);

// Dropping something onto the pane, and whether the drag started here (a
// folder cannot be dropped into its own subtree).
const draghover = ref(false);
const isDragSource = ref(false);

const fetching = ref(true);

const filesPagination = computed<Paging>(() => ({
	endpoint: "drive/files",
	params: { folderId: folder.value == null ? null : folder.value.id },
	limit: FILES_MAX,
}));

watch(folder, (value) => emit("cd", value));

function pagingHandle(): FilesPaginationHandle | null {
	return filesPaginationRef.value as FilesPaginationHandle | null;
}

async function fetchFolders(): Promise<void> {
	const fetched = await api<DriveFolder[]>("drive/folders", {
		folderId: folder.value == null ? null : folder.value.id,
		limit: FOLDERS_MAX + 1,
	});

	if (fetched.length === FOLDERS_MAX + 1) {
		moreFolders.value = true;
		fetched.pop();
	} else {
		moreFolders.value = false;
	}

	folders.value = fetched;
}

async function fetchMoreFolders(): Promise<void> {
	const last = folders.value[folders.value.length - 1];
	if (last == null) return;

	const fetched = await api<DriveFolder[]>("drive/folders", {
		folderId: folder.value == null ? null : folder.value.id,
		untilId: last.id,
		limit: FOLDERS_MAX + 1,
	});

	if (fetched.length === FOLDERS_MAX + 1) {
		moreFolders.value = true;
		fetched.pop();
	} else {
		moreFolders.value = false;
	}

	folders.value = folders.value.concat(fetched);
}

async function reloadFiles(): Promise<void> {
	const handle = pagingHandle();
	if (handle == null) return;
	await handle.reload();
}

async function refresh(): Promise<void> {
	fetching.value = true;
	try {
		await Promise.all([fetchFolders(), reloadFiles()]);
	} catch {
		void alert({ type: "error", text: t("somethingHappened") });
	} finally {
		fetching.value = false;
	}
}

/** Walks parentId up to the root; the packed folder never nests its parent. */
async function buildHierarchy(from: DriveFolder): Promise<DriveFolder[]> {
	const chain: DriveFolder[] = [];
	let cursor = from.parentId;
	let guard = 0;

	while (cursor != null && guard < 32) {
		const parent = await api<DriveFolder>("drive/folders/show", {
			folderId: cursor,
		});
		chain.unshift(parent);
		cursor = parent.parentId;
		guard += 1;
	}

	return chain;
}

async function move(target?: DriveFolder | string | null): Promise<void> {
	if (target == null) {
		await goRoot();
		return;
	}

	const folderId = typeof target === "string" ? target : target.id;

	fetching.value = true;
	try {
		const folderToMove = await api<DriveFolder>("drive/folders/show", {
			folderId,
		});
		folder.value = folderToMove;
		hierarchyFolders.value = await buildHierarchy(folderToMove);
		emit("open-folder", folderToMove);
	} catch {
		void alert({ type: "error", text: t("somethingHappened") });
		fetching.value = false;
		return;
	}

	await refresh();
}

async function goRoot(): Promise<void> {
	if (folder.value == null) return;

	folder.value = null;
	hierarchyFolders.value = [];
	emit("move-root");
	await refresh();
}

function addFolder(created: DriveFolder): void {
	const current = folder.value == null ? null : folder.value.id;
	if (current !== created.parentId) return;
	if (folders.value.some((f) => f.id === created.id)) return;
	folders.value.unshift(created);
}

function addFile(created: DriveFile): void {
	const current = folder.value == null ? null : folder.value.id;
	if (current !== created.folderId) return;
	pagingHandle()?.prepend(created);
}

function removeFolder(folderId: string): void {
	folders.value = folders.value.filter((f) => f.id !== folderId);
	hierarchyFolders.value = hierarchyFolders.value.filter(
		(f) => f.id !== folderId,
	);
}

function removeFile(file: DriveFile | string): void {
	const id = typeof file === "string" ? file : file.id;
	pagingHandle()?.removeItem((item) => item.id === id);
}

/** Replaces a tile after `drive/files/update`. */
function replaceFile(updated: DriveFile): void {
	pagingHandle()?.updateItem(updated.id, () => updated);
}

/** Applies a rename/move result coming from a folder tile or breadcrumb. */
function replaceFolder(updated: DriveFolder): void {
	if (folder.value != null && folder.value.id === updated.id) {
		folder.value = updated;
		return;
	}

	const current = folder.value == null ? null : folder.value.id;
	if (updated.parentId !== current) {
		removeFolder(updated.id);
		return;
	}

	const index = folders.value.findIndex((f) => f.id === updated.id);
	if (index === -1) {
		folders.value.unshift(updated);
	} else {
		folders.value[index] = updated;
	}
}

function onFolderDeleted(folderId: string): void {
	removeFolder(folderId);
	if (folder.value != null && folder.value.id === folderId) {
		void goRoot();
	}
}

function selectLocalFile(): void {
	fileInput.value?.click();
}

function createFolder(): void {
	inputText({
		title: t("createFolder"),
		placeholder: t("folderName"),
	}).then(({ canceled, result: name }) => {
		if (canceled) return;
		api<DriveFolder>("drive/folders/create", {
			name,
			parentId: folder.value == null ? null : folder.value.id,
		})
			.then((created) => addFolder(created))
			.catch(() => {
				void alert({ type: "error", text: t("somethingHappened") });
			});
	});
}

function renameCurrentFolder(): void {
	const current = folder.value;
	if (current == null) return;

	inputText({
		title: t("renameFolder"),
		placeholder: t("inputNewFolderName"),
		default: current.name,
	}).then(({ canceled, result: name }) => {
		if (canceled) return;
		api<DriveFolder>("drive/folders/update", {
			folderId: current.id,
			name,
		})
			.then((updated) => {
				folder.value = updated;
			})
			.catch(() => {
				void alert({ type: "error", text: t("somethingHappened") });
			});
	});
}

function deleteCurrentFolder(): void {
	const current = folder.value;
	if (current == null) return;

	api("drive/folders/delete", { folderId: current.id })
		.then(() => {
			if (defaultStore.state.uploadFolder === current.id) {
				defaultStore.set("uploadFolder", null);
			}
			const parentId = current.parentId;
			folder.value = null;
			hierarchyFolders.value = [];
			if (parentId != null) {
				void move(parentId);
			} else {
				void refresh();
			}
		})
		.catch(() => {
			void alert({ type: "error", text: t("unableToDelete") });
		});
}

function onChangeFileInput(): void {
	const input = fileInput.value;
	if (input == null || input.files == null) return;
	for (const file of Array.from(input.files)) {
		upload(file, folder.value);
	}
	// Allow picking the same file again.
	input.value = "";
}

/**
 * Upstream semantics: an omitted target means the drive root, so dropping a
 * file on the root breadcrumb always uploads to the root. Callers that mean
 * "the folder being viewed" pass `folder.value` explicitly.
 */
function upload(
	file: File,
	folderToUpload?: DriveFolder | null,
): void {
	const folderId =
		folderToUpload != null && typeof folderToUpload === "object"
			? folderToUpload.id
			: null;

	uploadFile(file, { folderId })
		.then((created) => addFile(created))
		// uploadFile already surfaced the failure.
		.catch(() => {});
}

function chooseFile(file: DriveFile): void {
	const isAlreadySelected = selectedFiles.value.some((f) => f.id === file.id);
	if (props.multiple) {
		if (isAlreadySelected) {
			selectedFiles.value = selectedFiles.value.filter(
				(f) => f.id !== file.id,
			);
		} else {
			selectedFiles.value.push(file);
		}
		emit("change-selection", selectedFiles.value);
	} else {
		if (isAlreadySelected) {
			emit("selected", file);
		} else {
			selectedFiles.value = [file];
			emit("change-selection", [file]);
		}
	}
}

function chooseFolder(folderToChoose: DriveFolder): void {
	const isAlreadySelected = selectedFolders.value.some(
		(f) => f.id === folderToChoose.id,
	);
	if (props.multiple) {
		if (isAlreadySelected) {
			selectedFolders.value = selectedFolders.value.filter(
				(f) => f.id !== folderToChoose.id,
			);
		} else {
			selectedFolders.value.push(folderToChoose);
		}
		emit("change-selection", selectedFolders.value);
	} else {
		if (isAlreadySelected) {
			emit("selected", folderToChoose);
		} else {
			selectedFolders.value = [folderToChoose];
			emit("change-selection", [folderToChoose]);
		}
	}
}

function onDragover(ev: DragEvent): void {
	if (!ev.dataTransfer) return;

	// A drag that started here must not be dropped onto its own subtree.
	if (isDragSource.value) {
		ev.dataTransfer.dropEffect = "none";
		return;
	}

	const isFile = ev.dataTransfer.items[0]?.kind === "file";
	const isDriveFolder =
		ev.dataTransfer.types[0] === DATA_TRANSFER_DRIVE_FOLDER;

	if (isFile || isDriveFolder) {
		ev.dataTransfer.dropEffect =
			ev.dataTransfer.effectAllowed === "all" ? "copy" : "move";
	} else {
		ev.dataTransfer.dropEffect = "none";
	}
}

function onDragenter(): void {
	if (!isDragSource.value) draghover.value = true;
}

function onDragleave(): void {
	draghover.value = false;
}

function onDrop(ev: DragEvent): void {
	draghover.value = false;

	if (!ev.dataTransfer) return;

	if (ev.dataTransfer.files.length > 0) {
		for (const file of Array.from(ev.dataTransfer.files)) {
			upload(file, folder.value);
		}
		return;
	}

	const driveFolder = ev.dataTransfer.getData(DATA_TRANSFER_DRIVE_FOLDER);
	if (driveFolder != null && driveFolder !== "") {
		const dropped = JSON.parse(driveFolder) as DriveFolder;
		if (folder.value != null && dropped.id === folder.value.id) return;
		if (folders.value.some((f) => f.id === dropped.id)) return;

		api<DriveFolder>("drive/folders/update", {
			folderId: dropped.id,
			parentId: folder.value == null ? null : folder.value.id,
		})
			.then(() => removeFolder(dropped.id))
			.catch(() => {
				void alert({ type: "error", text: t("somethingHappened") });
			});
	}
}

async function uploadFromUrl(): Promise<void> {
	const { canceled, result } = await inputText({
		title: "从 URL 上传",
		placeholder: "https://example.com/image.png",
	});
	if (canceled || result == null || result.trim() === "") return;
	try {
		const file = await api("drive/files/upload-from-url", {
			url: result.trim(),
			folderId: folder.value?.id ?? null,
		});
		pagingHandle()?.prepend(file);
		success();
	} catch (error) {
		void alert({
			type: "error",
			text: error instanceof ApiError ? error.message : t("somethingHappened"),
		});
	}
}

function getMenu(): MenuItem[] {
	const items: MenuItem[] = [
		{ type: "label", text: t("addFile") },
		{
			text: t("upload"),
			icon: "ph-upload-simple ph-bold ph-lg",
			action: selectLocalFile,
		},
		{
			text: "从 URL 上传",
			icon: "ph-link-simple ph-bold ph-lg",
			action: uploadFromUrl,
		},
		{ type: "label", text: folder.value == null ? t("drive") : folder.value.name },
	];

	if (folder.value != null) {
		items.push({
			text: t("renameFolder"),
			icon: "ph-cursor-text ph-bold ph-lg",
			action: renameCurrentFolder,
		});
		items.push({
			text: t("deleteFolder"),
			icon: "ph-trash ph-bold ph-lg",
			action: deleteCurrentFolder,
		});
	}

	items.push({
		text: t("createFolder"),
		icon: "ph-folder-notch-plus ph-bold ph-lg",
		action: createFolder,
	});

	return items;
}

function showMenu(ev: MouseEvent): void {
	void popupMenu(getMenu(), ev);
}

function onContextmenu(ev: MouseEvent): void {
	void contextMenu(getMenu(), ev);
}

/** First paint of the root folder; CfPagination fetched the file page itself. */
async function loadRoot(): Promise<void> {
	fetching.value = true;
	try {
		await fetchFolders();
	} catch {
		void alert({ type: "error", text: t("somethingHappened") });
	} finally {
		fetching.value = false;
	}
}

watch(
	() => props.initialFolderId,
	(id) => {
		if (id == null || id === "") return;
		const current = folder.value == null ? null : folder.value.id;
		if (current === id) return;
		void move(id);
	},
);

onMounted(() => {
	if (props.initialFolder != null) {
		void move(props.initialFolder);
	} else if (props.initialFolderId != null && props.initialFolderId !== "") {
		void move(props.initialFolderId);
	} else {
		void loadRoot();
	}
});
</script>

<style lang="scss" scoped>
.yfudmmck {
	display: flex;
	flex-direction: column;
	height: 100%;

	> nav {
		display: flex;
		z-index: 2;
		width: 100%;
		padding: 0 8px;
		box-sizing: border-box;
		overflow: auto;
		font-size: 0.9em;
		box-shadow: 0 1px 0 var(--divider);

		&,
		* {
			user-select: none;
		}

		> .path {
			display: inline-block;
			vertical-align: bottom;
			line-height: 42px;
			white-space: nowrap;

			> * {
				display: inline-block;
				margin: 0;
				padding: 0 8px;
				line-height: 42px;
				cursor: pointer;

				* {
					pointer-events: none;
				}

				&:hover {
					text-decoration: underline;
				}

				&.current {
					font-weight: bold;
					cursor: default;

					&:hover {
						text-decoration: none;
					}
				}

				&.separator {
					margin: 0;
					padding: 0;
					opacity: 0.5;
					cursor: default;

					> i {
						margin: 0;
					}
				}
			}
		}

		> .menu {
			margin-left: auto;
			padding: 0 12px;
		}
	}

	> .main {
		flex: 1;
		overflow: auto;
		padding: var(--margin);

		&,
		* {
			user-select: none;
		}

		&.fetching {
			cursor: wait;

			* {
				pointer-events: none;
			}

			> .contents {
				opacity: 0.5;
			}
		}

		&.uploading {
			height: calc(100% - 38px - 100px);
		}

		> .contents {
			> .folders,
			.files {
				display: flex;
				flex-wrap: wrap;

				> .folder,
				> .file {
					flex-grow: 1;
					width: 128px;
					margin: 4px;
					box-sizing: border-box;
				}

				> .padding {
					flex-grow: 1;
					pointer-events: none;
					width: 128px + 8px;
				}
			}

			.empty {
				padding: 16px;
				text-align: center;
				pointer-events: none;
				opacity: 0.5;

				> p {
					margin: 0;
				}
			}
		}
	}

	> .dropzone {
		position: absolute;
		left: 0;
		top: 38px;
		width: 100%;
		height: calc(100% - 38px);
		border: dashed 2px var(--focus);
		pointer-events: none;
	}

	> input {
		display: none;
	}
}
</style>
