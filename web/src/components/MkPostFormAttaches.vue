<!-- Ported from firefish@b425f61:packages/client/src/components/MkPostFormAttaches.vue (AGPL-3.0-only) -->
<template>
	<div v-show="files.length > 0" class="skeikyzd">
		<div class="files">
			<div v-for="file in files" :key="file.id" class="file">
				<MkDriveFileThumbnail class="thumbnail" :file="file" fit="cover" />
				<button
					class="_button remove"
					:aria-label="t('attachCancel')"
					@click="detach(file)"
				>
					<i class="ph-x ph-bold ph-lg"></i>
				</button>
				<div v-if="file.isSensitive" class="sensitive">
					<i class="ph-warning ph-bold ph-lg icon"></i>
				</div>
			</div>
		</div>
		<p class="remain">{{ 16 - files.length }}/16</p>
	</div>
</template>

<script lang="ts" setup>
// Wave-3 degradations (contract §2):
//   * `vue-draggable-plus` is not a dependency, so attachments keep the order in
//     which they were picked and cannot be drag-sorted;
//   * `drive/files/update` does not exist, so rename / sensitive flag / caption
//     are gone and the only per-file action is "remove".
import type { DriveFile } from "@/api/types";
import MkDriveFileThumbnail from "@/components/MkDriveFileThumbnail.vue";
import { t } from "@/i18n";

defineProps<{
	files: DriveFile[];
}>();

const emit = defineEmits<{
	detach: [id: string];
}>();

function detach(file: DriveFile): void {
	emit("detach", file.id);
}
</script>

<style lang="scss" scoped>
.skeikyzd {
	padding: 8px 24px;
	position: relative;

	> .files {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;

		> .file {
			position: relative;
			inline-size: 64px;
			block-size: 64px;
			border-radius: 4px;

			&:hover > .remove {
				opacity: 1;
			}

			> .thumbnail {
				inline-size: 100%;
				block-size: 100%;
				z-index: 1;
				color: var(--fg);
			}

			> .remove {
				position: absolute;
				inset-block-start: 2px;
				inset-inline-end: 2px;
				z-index: 3;
				display: flex;
				align-items: center;
				justify-content: center;
				inline-size: 22px;
				block-size: 22px;
				border-radius: 100%;
				background: var(--panel);
				color: var(--fg);
				opacity: 0;
				transition: opacity 0.2s;

				&:focus-visible {
					opacity: 1;
				}
			}

			> .sensitive {
				display: flex;
				position: absolute;
				inline-size: 100%;
				block-size: 100%;
				inset-block-start: 0;
				inset-inline-start: 0;
				z-index: 2;
				background: var(--header);
				color: var(--fg);

				> .icon {
					margin: auto;
				}
			}
		}
	}

	> .remain {
		display: block;
		position: absolute;
		inset-block-start: 8px;
		inset-inline-end: 8px;
		margin: 0;
		padding: 0;
		font-size: 0.75em;
		opacity: 0.7;
	}
}

// Touch devices never hover, so the remove button stays visible there.
@media (hover: none) {
	.skeikyzd > .files > .file > .remove {
		opacity: 1;
	}
}
</style>
