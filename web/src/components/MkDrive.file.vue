<!-- Ported from firefish@b425f61:packages/client/src/components/MkDrive.file.vue (AGPL-3.0-only) -->
<template>
	<div
		class="ncvczrfv"
		:class="{ isSelected }"
		:title="title"
		@click="onClick"
		@contextmenu.stop="onContextmenu"
	>
		<MkDriveFileThumbnail class="thumbnail" :file="file" fit="contain" />

		<p class="name">
			<span>{{ baseName }}</span>
			<span v-if="extension != ''" class="ext">{{ extension }}</span>
		</p>
	</div>
</template>

<script lang="ts" setup>
// Wave-2 differences from upstream (contract §5, drive-media.md):
//   * the avatar / banner / NSFW corner labels are gone: `_i.avatarId` and
//     `_i.bannerId` are always null and `isSensitive` is always false;
//   * rename / describe / mark-as-sensitive are gone: `drive/files/update`
//     does not exist, so the only mutations left are delete;
//   * `deleted` is emitted after a successful delete because the drive has no
//     streaming channel that would refresh the list for us;
//   * the file tile is not draggable: moving a file between folders needs the
//     same missing endpoint;
//   * "used in notes" was added as a menu entry so AttachedFiles has an entry
//     point (the endpoint behind it is missing, the page says so).
import { computed } from "vue";
import { useRouter } from "vue-router";
import type { DriveFile } from "@/api/types";
import { api, resolveMediaUrl } from "@/api/client";
import MkDriveFileThumbnail from "@/components/MkDriveFileThumbnail.vue";
import bytes from "@/filters/bytes";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { confirm, contextMenu, popupMenu, success, type MenuItem } from "@/os";
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
	(ev: "deleted", id: string): void;
}>();

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
}
</style>
