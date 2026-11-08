<!-- Ported from firefish@b425f61:packages/client/src/components/MkPagePreview.vue (AGPL-3.0-only) -->
<template>
	<MkA :to="`/@${page.user.username}/pages/${page.name}`" class="cf-page-preview _block">
		<div v-if="thumbnailUrl !== ''" class="thumbnail">
			<MkImgWithBlurhash
				:src="thumbnailUrl"
				:hash="page.eyeCatchingImage?.blurhash"
				:alt="page.title"
			/>
		</div>
		<article>
			<header>
				<h1 :title="page.title">{{ page.title }}</h1>
			</header>
			<p v-if="summaryExcerpt !== ''" :title="page.summary ?? undefined">
				{{ summaryExcerpt }}
			</p>
			<footer>
				<MkAvatar :user="page.user" class="avatar" disable-link />
				<MkUserName :user="page.user" class="name" />
			</footer>
		</article>
	</MkA>
</template>

<script setup lang="ts">
// Firefish paints the eye-catching image as a 200px banner above the title,
// summary excerpt and author line. Cinderfish resolves the file URL through
// MkImgWithBlurhash (which degrades to a plain <img> because the backend never
// emits a blurhash) and uses MkAvatar/MkUserName so the avatar URL goes
// through the same media resolver as the rest of the client.
import { computed } from "vue";
import { resolveMediaUrl } from "@/api/client";
import type { Page } from "@/api/types";
import MkImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";

const props = defineProps<{
	page: Page;
}>();

const SUMMARY_MAX = 85;

const thumbnailUrl = computed<string>(() => {
	const image = props.page.eyeCatchingImage;
	if (image == null) return "";
	return resolveMediaUrl(image.thumbnailUrl ?? image.url);
});

const summaryExcerpt = computed<string>(() => {
	const summary = props.page.summary;
	if (summary == null || summary === "") return "";
	return summary.length > SUMMARY_MAX
		? `${summary.slice(0, SUMMARY_MAX)}…`
		: summary;
});
</script>

<style lang="scss" scoped>
.cf-page-preview {
	display: block;
	overflow: hidden;

	&:hover {
		text-decoration: none;
		color: var(--accent);
	}

	> .thumbnail {
		block-size: 200px;
		overflow: hidden;

		img {
			display: block;
			inline-size: 100%;
			block-size: 100%;
			object-fit: cover;
		}
	}

	> article {
		padding: 16px;

		> header {
			margin-block-end: 8px;

			> h1 {
				margin: 0;
				font-size: 1em;
				color: var(--urlPreviewTitle);
			}
		}

		> p {
			margin: 0;
			color: var(--urlPreviewText);
			font-size: 0.8em;
			overflow-wrap: anywhere;
		}

		> footer {
			display: flex;
			align-items: center;
			gap: 4px;
			margin-block-start: 8px;
			color: var(--urlPreviewInfo);
			font-size: 0.8em;

			> .avatar {
				inline-size: 16px;
				block-size: 16px;
				flex: 0 0 auto;
			}

			> .name {
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
			}
		}
	}

	@media (max-inline-size: 700px) {
		> .thumbnail {
			block-size: 100px;
		}
	}
}
</style>
