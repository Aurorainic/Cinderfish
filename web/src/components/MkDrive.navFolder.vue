<!-- Ported from firefish@b425f61:packages/client/src/components/MkDrive.navFolder.vue (AGPL-3.0-only) -->
<template>
	<div
		class="drylbebk"
		:class="{ draghover }"
		@click="onClick"
		@dragover.prevent.stop="onDragover"
		@dragenter="onDragenter"
		@dragleave="onDragleave"
		@drop.stop="onDrop"
	>
		<i v-if="folder == null" class="ph-cloud ph-bold ph-lg"></i>
		<span>{{ folder == null ? t("drive") : folder.name }}</span>
	</div>
</template>

<script lang="ts" setup>
// Dropping a *file* onto a breadcrumb is still not handled: files move by
// being dropped on a folder tile in the grid. This component keeps OS uploads
// and drive-folder moves.
// The dropped-folder move reports the backend's INVALID_PARAM for a cycle.
import { ref } from "vue";
import { api, ApiError } from "@/api/client";
import type { DriveFolder } from "@/api/types";
import { alert } from "@/os";
import { t } from "@/i18n";

// Firefish injects these through vite `define`; the string values are kept
// identical so drag payloads stay compatible across components.
const DATA_TRANSFER_DRIVE_FOLDER = "mk_drive_folder";

const props = defineProps<{
	folder?: DriveFolder;
	parentFolder: DriveFolder | null;
}>();

const emit = defineEmits<{
	(ev: "move", v?: DriveFolder): void;
	(ev: "upload", file: File, folder?: DriveFolder | null): void;
	(ev: "updated", v: DriveFolder): void;
}>();

const draghover = ref(false);

function onClick() {
	emit("move", props.folder);
}

function onDragover(ev: DragEvent) {
	if (!ev.dataTransfer) return;

	// This folder is both root and the current directory: never a drop target.
	if (props.folder == null && props.parentFolder == null) {
		ev.dataTransfer.dropEffect = "none";
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

	return false;
}

function onDragenter() {
	if (props.folder || props.parentFolder) draghover.value = true;
}

function onDragleave() {
	if (props.folder || props.parentFolder) draghover.value = false;
}

function moveFolderInto(targetFolderId: string, parentId: string | null) {
	api<DriveFolder>("drive/folders/update", {
		folderId: targetFolderId,
		parentId,
	})
		.then((updated) => emit("updated", updated))
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
		if (props.folder && dropped.id === props.folder.id) return;
		moveFolderInto(
			dropped.id,
			props.folder == null ? null : props.folder.id,
		);
	}
}
</script>

<style lang="scss" scoped>
.drylbebk {
	> * {
		pointer-events: none;
	}

	&.draghover {
		background: var(--buttonHoverBg);
	}

	> i {
		margin-right: 4px;
	}
}
</style>
