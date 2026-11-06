<!-- Ported from firefish@b425f61:packages/client/src/components/MkDriveFileThumbnail.vue (AGPL-3.0-only) -->
<template>
	<button class="zdjebgpv">
		<img
			v-if="is === 'image'"
			class="image"
			:src="file.thumbnailUrl ?? file.url"
			:alt="file.name"
			:title="file.name"
			:style="{ objectFit: fit === 'contain' ? 'contain' : 'cover' }"
		/>
		<i
			v-else-if="is === 'video'"
			class="ph-file-video ph-bold ph-lg icon"
		></i>
		<i
			v-else-if="is === 'audio' || is === 'midi'"
			class="ph-file-audio ph-bold ph-lg icon"
		></i>
		<i v-else-if="is === 'csv'" class="ph-file-csv ph-bold ph-lg icon"></i>
		<i v-else-if="is === 'pdf'" class="ph-file-pdf ph-bold ph-lg icon"></i>
		<i
			v-else-if="is === 'textfile'"
			class="ph-file-text ph-bold ph-lg icon"
		></i>
		<i
			v-else-if="is === 'archive'"
			class="ph-file-zip ph-bold ph-lg icon"
		></i>
		<i v-else class="ph-file ph-bold ph-lg icon"></i>
	</button>
</template>

<script lang="ts" setup>
// Images load `thumbnailUrl` when the server generated one (wave 46) and fall
// back to the original `url` otherwise. `blurhash` is still always null, so
// there is no placeholder to show while loading; the video badge and the
// per-type icons are unchanged from upstream.
import { computed } from "vue";
import type { DriveFile } from "@/api/types";

const props = defineProps<{
	file: DriveFile;
	fit: string;
}>();

const is = computed(() => {
	if (props.file.type.startsWith("image/")) return "image";
	if (props.file.type.startsWith("video/")) return "video";
	if (props.file.type === "audio/midi") return "midi";
	if (props.file.type.startsWith("audio/")) return "audio";
	if (props.file.type.endsWith("/csv")) return "csv";
	if (props.file.type.endsWith("/pdf")) return "pdf";
	if (props.file.type.startsWith("text/")) return "textfile";
	if (
		[
			"application/zip",
			"application/x-cpio",
			"application/x-bzip",
			"application/x-bzip2",
			"application/java-archive",
			"application/x-rar-compressed",
			"application/x-tar",
			"application/gzip",
			"application/x-7z-compressed",
		].some((archiveType) => archiveType === props.file.type)
	)
		return "archive";
	return "unknown";
});
</script>

<style lang="scss" scoped>
.zdjebgpv {
	position: relative;
	display: flex;
	background: var(--panel);
	border-radius: 8px;
	overflow: clip;
	border: 0;
	padding: 0;
	cursor: pointer;

	> .image {
		display: block;
		inline-size: 100%;
		block-size: 100%;
		pointer-events: none;
	}

	> .icon {
		pointer-events: none;
		margin: auto;
		font-size: 32px;
		color: #777;
	}
}
</style>
