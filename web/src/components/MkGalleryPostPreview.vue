<!-- Ported from firefish@b425f61:packages/client/src/components/MkGalleryPostPreview.vue (AGPL-3.0-only) -->
<template>
	<MkA :to="`/gallery/${post.id}`" class="cf-gallery-post-preview _panel">
		<div class="thumbnail">
			<MkImgWithBlurhash
				v-if="thumbnailUrl !== ''"
				class="img"
				:class="{ blurred: hidden }"
				:src="thumbnailUrl"
				:hash="post.files[0].blurhash"
			/>
			<div v-else class="missing _fullinfo">
				<i class="ph-image-square ph-bold ph-lg"></i>
				<span>{{ t("notFound") }}</span>
			</div>
			<div v-if="hidden" class="sensitiveOverlay">
				<i class="ph-eye-slash ph-bold ph-lg"></i>
				<span>{{ t("sensitive") }}</span>
			</div>
		</div>
		<article>
			<header>
				<MkAvatar :user="post.user" class="avatar" />
			</header>
			<footer>
				<span class="title">{{ post.title }}</span>
				<span class="meta">
					<span v-if="post.isSensitive" class="flag">
						<i class="ph-warning ph-bold"></i>
						{{ t("sensitive") }}
					</span>
					<span class="likes">
						<i
							class="ph-heart ph-bold"
							:class="{ 'ph-fill': post.isLiked === true }"
						></i>
						{{ post.likedCount }}
					</span>
				</span>
			</footer>
		</article>
	</MkA>
</template>

<script setup lang="ts">
// Firefish paints the first file's thumbnail over a 200px card with the author
// in the top-right and the title along the bottom. Cinderfish never emits a
// blurhash, so MkImgWithBlurhash degrades to a plain <img>; a deleted first
// file (the card then has no image) falls back to a themed box in place of
// Firefish's `not-found.webp` asset, which this repository does not ship.
//
// Sensitive posts keep the Google-images treatment: the thumbnail is blurred
// and covered by a warning until the reader opens the post page and reveals it.
import { computed } from "vue";
import type { GalleryPost } from "@/api/types";
import MkImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";
import { t } from "@/i18n";

const props = defineProps<{
	post: GalleryPost;
}>();

const firstFile = computed(() => props.post.files[0] ?? null);

const thumbnailUrl = computed<string>(() => {
	const file = firstFile.value;
	if (file == null) return "";
	return file.thumbnailUrl ?? file.url ?? "";
});

const hidden = computed<boolean>(
	() =>
		props.post.isSensitive ||
		(firstFile.value != null && firstFile.value.isSensitive),
);
</script>

<style lang="scss" scoped>
.cf-gallery-post-preview {
	position: relative;
	display: block;
	block-size: 200px;
	overflow: hidden;

	&:hover,
	&:focus {
		text-decoration: none;
		color: var(--accent);

		> .thumbnail > .img {
			transform: scale(1.1);
		}
	}

	> .thumbnail {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: transform 0.5s ease;

		> .img {
			inline-size: 100%;
			block-size: 100%;
			object-fit: cover;
			transition: transform 0.5s ease;

			&.blurred {
				filter: blur(16px);
				transform: scale(1.1);
			}
		}

		> .missing {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 8px;
			color: var(--fgTransparentWeak);
		}

		> .sensitiveOverlay {
			position: absolute;
			inset: 0;
			display: flex;
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: 6px;
			font-size: 0.85em;
			font-weight: bold;
			color: #fff;
			text-shadow: 0 0 8px rgba(0, 0, 0, 0.6);
			background: rgba(0, 0, 0, 0.25);
		}
	}

	> article {
		position: absolute;
		inset: 0;
		z-index: 1;

		> header {
			position: absolute;
			inset-block-start: 0;
			inline-size: 100%;
			box-sizing: border-box;
			display: flex;
			padding: 12px;

			> .avatar {
				margin-inline-start: auto;
				inline-size: 32px;
				block-size: 32px;
			}
		}

		> footer {
			position: absolute;
			inset-block-end: 0;
			inline-size: 100%;
			box-sizing: border-box;
			display: flex;
			flex-direction: column;
			gap: 4px;
			padding: 16px;
			color: #fff;
			text-shadow: 0 0 8px var(--shadow);
			background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));

			> .title {
				font-weight: bold;
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
			}

			> .meta {
				display: flex;
				align-items: center;
				gap: 12px;
				font-size: 0.82em;

				> span {
					display: inline-flex;
					align-items: center;
					gap: 4px;
				}

				> .flag {
					opacity: 0.9;
				}
			}
		}
	}
}
</style>
