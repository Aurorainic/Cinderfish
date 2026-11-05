<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.photos.vue (AGPL-3.0-only) -->
<template>
	<div class="photos-container _panel">
		<header class="header">
			<i class="ph-image ph-bold ph-lg"></i>
			<span>{{ t("images") }}</span>
		</header>
		<CfPagination
			:pagination="pagination"
			:disable-auto-load="!paginated"
			class="body"
		>
			<template #empty>
				<div class="_fullinfo">
					<div>{{ t("nothing") }}</div>
				</div>
			</template>

			<template #default="{ items }">
				<div class="stream">
					<MkA
						v-for="image in collectImages(items)"
						:key="image.note.id + image.file.id"
						class="img"
						:to="notePage(image.note)"
					>
						<img
							:src="resolveMediaUrl(image.file.url)"
							:title="image.file.name"
							:alt="image.file.name"
							loading="lazy"
						/>
					</MkA>
				</div>
			</template>
		</CfPagination>
	</div>
</template>

<script lang="ts" setup>
// Degradations versus Firefish:
//   * `users/notes` has no `fileType` filter, so every note with attachments is
//     fetched (`withFiles: true`) and the images are picked client-side;
//   * `thumbnailUrl` and `blurhash` are always null (no thumbnailing), so the
//     original `url` is shown instead of MkImgWithBlurhash;
//   * `excludeNsfw` does not exist; sensitive files are hidden locally unless
//     the reader opted into seeing them (`defaultStore.state.nsfw`).
//
// The same component backs the sidebar card (`paginated` false, first page
// only) and the profile's media tab (`paginated` true, infinite scroll).
import { computed } from "vue";
import { resolveMediaUrl } from "@/api/client";
import type { DriveFile, NoteDetailed, UserDetailed } from "@/api/types";
import CfPagination from "@/components/ui/CfPagination.vue";
import type { Paging } from "@/components/ui";
import { notePage } from "@/filters/note";
import { t } from "@/i18n";
import { defaultStore } from "@/store";

const props = withDefaults(
	defineProps<{
		user: UserDetailed;
		paginated?: boolean;
	}>(),
	{
		paginated: false,
	},
);

const pagination = computed<Paging>(() => ({
	endpoint: "users/notes",
	limit: props.paginated ? 20 : 10,
	params: {
		userId: props.user.id,
		withFiles: true,
		withReplies: true,
	},
}));

interface PhotoEntry {
	note: NoteDetailed;
	file: DriveFile;
}

function collectImages(items: { id: string }[]): PhotoEntry[] {
	const showSensitive = defaultStore.state.nsfw === "ignore";
	const photos: PhotoEntry[] = [];
	for (const raw of items as NoteDetailed[]) {
		for (const file of raw.files) {
			if (!file.type.startsWith("image/")) continue;
			if (file.isSensitive && !showSensitive) continue;
			photos.push({ note: raw, file });
		}
	}
	return photos;
}
</script>

<style lang="scss" scoped>
.photos-container {
	overflow: hidden;

	> .header {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 12px 16px;
		border-bottom: solid 0.5px var(--divider);
		font-weight: bold;
	}

	> .body {
		padding: 8px;
	}

	.stream {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
		grid-gap: 6px;

		> .img {
			position: relative;
			height: 112px;
			border-radius: 6px;
			overflow: clip;
			background: var(--buttonBg);

			> img {
				display: block;
				width: 100%;
				height: 100%;
				object-fit: cover;
			}
		}
	}
}
</style>
