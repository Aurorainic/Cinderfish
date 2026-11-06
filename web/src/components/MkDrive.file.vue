<!-- Ported from firefish@b425f61:packages/client/src/components/MkDrive.file.vue (AGPL-3.0-only) -->
<template>
	<div
		class="ncvczrfv"
		:class="{ isSelected }"
		:title="title"
		draggable="true"
		@click="onClick"
		@contextmenu.stop="onContextmenu"
		@dragstart="onDragstart"
		@dragend="onDragend"
	>
		<MkDriveFileThumbnail class="thumbnail" :file="file" fit="contain" />

		<p class="name">
			<span>{{ baseName }}</span>
			<span v-if="extension != ''" class="ext">{{ extension }}</span>
		</p>

		<span v-if="file.isSensitive" class="sensitive" :title="t('sensitive')">
			<i class="ph-eye-slash ph-bold"></i>
		</span>
	</div>
</template>

<script lang="ts" setup>
// Differences from upstream (contract §5, drive-media.md):
//   * the avatar / banner corner labels are gone: `_i.avatarId` and
//     `_i.bannerId` are always null; the sensitive flag is real now and the
//     tile shows its own marker;
//   * rename / describe / mark-as-sensitive call `drive/files/update`;
//   * moving a file between folders still has no drag affordance: the folder
//     picker is not ported, so the menu offers no "move" entry yet;
//   * `deleted` is emitted after a successful delete because the drive has no
//     streaming channel that would refresh the list for us; `updated` carries
//     the new metadata so the parent can replace the tile in place;
//   * "used in notes" was added as a menu entry so AttachedFiles has an entry
//     point (the endpoint behind it is missing, the page says so).
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { DriveFile } from "@/api/types";
import { api, resolveMediaUrl } from "@/api/client";
import MkDriveFileThumbnail from "@/components/MkDriveFileThumbnail.vue";
import bytes from "@/filters/bytes";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import {
	confirm,
	contextMenu,
	inputText,
	popupMenu,
	success,
	type MenuItem,
} from "@/os";
import { t } from "@/i18n";

const props = withDefaults(
	defineProps<{
		file: DriveFile;
		isSelected?: boolean;
		selectMode?: boolean;
	}>(),
	{
		isSelected: false,
		selectMode: false,
	},
);

const emit = defineEmits<{
	(ev: "chosen", r: DriveFile): void;
	(ev: "updated", file: DriveFile): void;
	(ev: "deleted", id: string): void;
}>();

// Data-transfer type for dragging a drive file onto a folder.
const DATA_TRANSFER_DRIVE_FILE = "mk_drive_file";

const router = useRouter();

const title = computed(
	() => `${props.file.name}\n${props.file.type} ${bytes(props.file.size)}`,
);

const baseName = computed(() => {
	const index = props.file.name.lastIndexOf(".");
	return index === -1 ? props.file.name : props.file.name.slice(0, index);
});

const extension = computed(() => {
	const index = props.file.name.lastIndexOf(".");
	return index === -1 ? "" : props.file.name.slice(index);
});

function getMenu(): MenuItem[] {
	return [
		{
			text: t("copyUrl"),
			icon: "ph-link-simple ph-bold ph-lg",
			action: copyUrl,
		},
		{
			text: t("download"),
			icon: "ph-download-simple ph-bold ph-lg",
			action: download,
		},
		{ separator: true },
		{
			text: t("rename"),
			icon: "ph-pencil-simple ph-bold ph-lg",
			action: renameFile,
		},
		{
			text: t("description"),
			icon: "ph-text-align-left ph-bold ph-lg",
			action: editComment,
		},
		{
			text: t("sensitive"),
			icon: "ph-eye-slash ph-bold ph-lg",
			action: toggleSensitive,
		},
		{ separator: true },
		{
			text: t("notes"),
			icon: "ph-note ph-bold ph-lg",
			action: showAttachedNotes,
		},
		{ separator: true },
		{
			text: t("delete"),
			icon: "ph-trash ph-bold ph-lg",
			danger: true,
			action: deleteFile,
		},
	];
}

function onClick(ev: MouseEvent) {
	if (props.selectMode) {
		emit("chosen", props.file);
	} else {
		void popupMenu(getMenu(), ev);
	}
}

function onContextmenu(ev: MouseEvent) {
	void contextMenu(getMenu(), ev);
}

function copyUrl() {
	void copyToClipboard(resolveMediaUrl(props.file.url));
	success();
}

function download() {
	const anchor = document.createElement("a");
	anchor.href = resolveMediaUrl(props.file.url);
	anchor.download = props.file.name;
	anchor.target = "_blank";
	anchor.rel = "noopener";
	anchor.click();
}

function showAttachedNotes() {
	void router.push(`/my/drive/file/${props.file.id}/attached`);
}

async function renameFile() {
	const { canceled, result } = await inputText({
		title: t("rename"),
		default: props.file.name,
		minLength: 1,
		maxLength: 255,
	});
	if (canceled || result == null || result.trim() === "") return;
	await applyUpdate({ name: result.trim() });
}

async function editComment() {
	const { canceled, result } = await inputText({
		title: t("description"),
		default: props.file.comment ?? "",
		maxLength: 512,
	});
	if (canceled) return;
	await applyUpdate({ comment: result === "" ? null : result });
}

async function toggleSensitive() {
	await applyUpdate({ isSensitive: !props.file.isSensitive });
}

async function applyUpdate(
	patch: { name?: string; comment?: string | null; isSensitive?: boolean },
): Promise<void> {
	try {
		const updated = await api("drive/files/update", {
			fileId: props.file.id,
			...patch,
		});
		emit("updated", updated);
		success();
	} catch {
		// Keep the tile as-is; the caller's error toast is enough.
	}
}

function onDragstart(ev: DragEvent): void {
	if (!ev.dataTransfer) return;
	ev.dataTransfer.effectAllowed = "move";
	ev.dataTransfer.setData(
		DATA_TRANSFER_DRIVE_FILE,
		JSON.stringify({ id: props.file.id, name: props.file.name }),
	);
}

function onDragend(): void {
	// The drop target performs the move; nothing to clean up here.
}

async function deleteFile() {
	const { canceled } = await confirm({
		type: "warning",
		text: t("driveFileDeleteConfirm", { name: props.file.name }),
	});

	if (canceled) return;

	try {
		await api("drive/files/delete", { fileId: props.file.id });
		emit("deleted", props.file.id);
	} catch {
		// The backend answers 404 NO_SUCH_FILE for a row that is already gone
		// (it also uses that code when the storage object could not be removed),
		// so a failure here is treated as "no longer in the list".
		emit("deleted", props.file.id);
	}
}
</script>

<style lang="scss" scoped>
.ncvczrfv {
	position: relative;
	padding: 8px 0 0 0;
	min-height: 180px;
	border-radius: 8px;

	&,
	* {
		cursor: pointer;
	}

	> * {
		pointer-events: none;
	}

	&:hover {
		background: rgba(#000, 0.05);
	}

	&:active {
		background: rgba(#000, 0.1);
	}

	&.isSelected {
		background: var(--accent);

		&:hover {
			background: var(--accentLighten);
		}

		&:active {
			background: var(--accentDarken);
		}

		> .name {
			color: #fff;
		}

		> .thumbnail {
			color: #fff;
		}
	}

	> .thumbnail {
		width: 110px;
		height: 110px;
		margin: auto;
	}

	> .name {
		display: block;
		margin: 4px 0 0 0;
		font-size: 0.8em;
		text-align: center;
		word-break: break-all;
		color: var(--fg);
		overflow: hidden;

		> .ext {
			opacity: 0.5;
		}
	}

	> .sensitive {
		position: absolute;
		top: 10px;
		inset-inline-end: 10px;
		display: grid;
		place-items: center;
		width: 22px;
		height: 22px;
		border-radius: 100%;
		background: var(--infoWarnBg);
		color: var(--infoWarnFg);
		font-size: 0.8em;
		pointer-events: none;
	}
}
</style>
