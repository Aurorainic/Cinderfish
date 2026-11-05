<!-- Ported from firefish@b425f61:packages/client/src/components/MkDrive.folder.vue (AGPL-3.0-only) -->
<template>
	<div
		class="rghtznwe"
		:class="{ draghover }"
		draggable="true"
		:title="title"
		@click="onClick"
		@contextmenu.stop="onContextmenu"
		@mouseover="onMouseover"
		@mouseout="onMouseout"
		@dragover.prevent.stop="onDragover"
		@dragenter.prevent="onDragenter"
		@dragleave="onDragleave"
		@drop.prevent.stop="onDrop"
		@dragstart="onDragstart"
		@dragend="onDragend"
	>
		<p class="name">
			<template v-if="hover"
				><i class="ph-folder-notch-open ph-bold ph-lg ph-fw"></i
			></template>
			<template v-if="!hover"
				><i class="ph-folder-notch ph-bold ph-lg ph-fw"></i
			></template>
			{{ folder.name }}
		</p>
		<p v-if="defaultStore.state.uploadFolder == folder.id" class="upload">
			{{ t("uploadFolder") }}
		</p>
		<button
			v-if="selectMode"
			class="checkbox _button"
			:class="{ checked: isSelected }"
			@click.prevent.stop="checkboxClicked"
		></button>
	</div>
</template>

<script lang="ts" setup>
// Wave-2 differences from upstream (contract §5, drive-media.md):
//   * renaming/deleting a folder emits `updated`/`deleted` because there is no
//     drive streaming channel to refresh the list;
//   * dropping a *file* onto a folder is not implemented (drive/files/update is
//     missing), so only OS uploads and folder moves are accepted;
//   * the menu has no `danger` styling: our os.MenuItem has no such field.
import { computed, defineAsyncComponent, ref } from "vue";
import { api, ApiError } from "@/api/client";
import type { DriveFolder } from "@/api/types";
import { alert, contextMenu, inputText, popup, type MenuItem } from "@/os";
import { t } from "@/i18n";
import { defaultStore } from "@/store/base";

// Firefish injects these through vite `define`; string values are unchanged.
const DATA_TRANSFER_DRIVE_FOLDER = "mk_drive_folder";

const props = withDefaults(
	defineProps<{
		folder: DriveFolder;
		isSelected?: boolean;
		selectMode?: boolean;
	}>(),
	{
		isSelected: false,
		selectMode: false,
	},
);

const emit = defineEmits<{
	(ev: "chosen", v: DriveFolder): void;
	(ev: "move", v: DriveFolder): void;
	(ev: "upload", file: File, folder: DriveFolder): void;
	(ev: "updated", v: DriveFolder): void;
	(ev: "deleted", id: string): void;
	(ev: "dragstart"): void;
	(ev: "dragend"): void;
}>();

const hover = ref(false);
const draghover = ref(false);
const isDragging = ref(false);

const title = computed(() => props.folder.name);

function checkboxClicked() {
	emit("chosen", props.folder);
}

function onClick() {
	emit("move", props.folder);
}

function onMouseover() {
	hover.value = true;
}

function onMouseout() {
	hover.value = false;
}

function onDragover(ev: DragEvent) {
	if (!ev.dataTransfer) return;

	if (isDragging.value) {
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

function onDragenter() {
	if (!isDragging.value) draghover.value = true;
}

function onDragleave() {
	draghover.value = false;
}

function moveFolderInto(movedId: string, parentId: string) {
	api<DriveFolder>("drive/folders/update", { folderId: movedId, parentId })
		.then(() => emit("deleted", movedId))
		.catch((cause: unknown) => {
			if (cause instanceof ApiError && cause.code === "INVALID_PARAM") {
				void alert({
					title: t("unableToProcess"),
					text: t("circularReferenceFolder"),
				});
			} else {
				void alert({ type: "error", text: t("somethingHappened") });
			}
		});
}

function onDrop(ev: DragEvent) {
	draghover.value = false;

	if (!ev.dataTransfer) return;

	if (ev.dataTransfer.files.length > 0) {
		for (const file of Array.from(ev.dataTransfer.files)) {
			emit("upload", file, props.folder);
		}
		return;
	}

	const driveFolder = ev.dataTransfer.getData(DATA_TRANSFER_DRIVE_FOLDER);
	if (driveFolder != null && driveFolder !== "") {
		const dropped = JSON.parse(driveFolder) as DriveFolder;
		if (dropped.id === props.folder.id) return;
		moveFolderInto(dropped.id, props.folder.id);
	}
}

function onDragstart(ev: DragEvent) {
	if (!ev.dataTransfer) return;

	ev.dataTransfer.effectAllowed = "move";
	ev.dataTransfer.setData(
		DATA_TRANSFER_DRIVE_FOLDER,
		JSON.stringify(props.folder),
	);
	isDragging.value = true;

	// Tell the owner that one of its children started the drag.
	emit("dragstart");
}

function onDragend() {
	isDragging.value = false;
	emit("dragend");
}

function openInWindow() {
	popup(
		defineAsyncComponent(() => import("@/components/MkDriveWindow.vue")),
		{ initialFolder: props.folder },
		{},
		"closed",
	);
}

function rename() {
	inputText({
		title: t("renameFolder"),
		placeholder: t("inputNewFolderName"),
		default: props.folder.name,
	}).then(({ canceled, result: name }) => {
		if (canceled) return;
		api<DriveFolder>("drive/folders/update", {
			folderId: props.folder.id,
			name,
		})
			.then((updated) => emit("updated", updated))
			.catch(() => {
				void alert({ type: "error", text: t("somethingHappened") });
			});
	});
}

function deleteFolder() {
	api("drive/folders/delete", { folderId: props.folder.id })
		.then(() => {
			if (defaultStore.state.uploadFolder === props.folder.id) {
				defaultStore.set("uploadFolder", null);
			}
			emit("deleted", props.folder.id);
		})
		.catch(() => {
			void alert({ type: "error", text: t("unableToDelete") });
		});
}

function onContextmenu(ev: MouseEvent) {
	const items: MenuItem[] = [
		{ text: t("openInWindow"), icon: "ph-copy ph-bold ph-lg", action: openInWindow },
		{ separator: true },
		{ text: t("rename"), icon: "ph-cursor-text ph-bold ph-lg", action: rename },
		{ separator: true },
		{
			text: t("delete"),
			icon: "ph-trash ph-bold ph-lg",
			danger: true,
			action: deleteFolder,
		},
	];
	void contextMenu(items, ev);
}
</script>

<style lang="scss" scoped>
.rghtznwe {
	position: relative;
	padding: 8px;
	height: 64px;
	background: var(--driveFolderBg);
	border-radius: 4px;

	&,
	* {
		cursor: pointer;
	}

	*:not(.checkbox) {
		pointer-events: none;
	}

	> .checkbox {
		position: absolute;
		bottom: 8px;
		right: 8px;
		width: 16px;
		height: 16px;
		background: #fff;
		border: solid 1px #000;

		&.checked {
			background: var(--accent);
		}
	}

	&.draghover {
		&:after {
			content: "";
			pointer-events: none;
			position: absolute;
			top: -4px;
			right: -4px;
			bottom: -4px;
			left: -4px;
			border: 2px dashed var(--focus);
			border-radius: 4px;
		}
	}

	> .name {
		margin: 0;
		font-size: 0.9em;
		color: var(--fg);

		> i {
			margin-right: 4px;
			margin-left: 2px;
			text-align: left;
		}
	}

	> .upload {
		margin: 4px 4px;
		font-size: 0.8em;
		text-align: right;
		color: var(--fg);
	}
}
</style>
