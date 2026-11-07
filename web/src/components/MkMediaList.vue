<!-- Ported from firefish@b425f61:packages/client/src/components/MkMediaList.vue (AGPL-3.0-only) -->
<template>
	<div class="hoawjimk files">
		<MkMediaBanner
			v-for="media in mediaList.filter((media) => !previewable(media))"
			:key="media.id"
			:media="media"
		/>
		<div
			v-if="previewableCount > 0"
			class="grid-container"
			:data-count="previewableCount < 5 ? previewableCount : null"
			:class="{ dmWidth: inDm }"
		>
			<div ref="gallery" @click.stop="onGridClick">
				<MkMedia
					v-for="media in previewableMedias"
					:key="media.id"
					:class="{ image: media.type.startsWith('image') }"
					:data-id="media.id"
					:media="media"
					:raw="raw"
				/>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { DriveFile } from "@/api/types";
import { FILE_TYPE_BROWSERSAFE } from "@/const";
import MkMediaBanner from "@/components/MkMediaBanner.vue";
import MkMedia from "@/components/MkMedia.vue";
import CfMediaLightbox from "@/components/CfMediaLightbox.vue";
import { resolveMediaUrl } from "@/api/client";
import { popup } from "@/os";

/**
 * Ported from Firefish. PhotoSwipe is not installed (contract §5), so gallery
 * clicks open the in-house CfMediaLightbox instead; a plain image link still
 * works if the click misses. The grid sizes, breakpoint classes and
 * `data-count` branches are unchanged.
 */
const props = defineProps<{
	mediaList: DriveFile[];
	raw?: boolean;
	inDm?: boolean;
}>();

const previewable = (file: DriveFile): boolean => {
	if (file.type === "image/svg+xml") return true;
	// FILE_TYPE_BROWSERSAFEに適合しないものはブラウザで表示するのに不適切
	return (
		(file.type.startsWith("video") || file.type.startsWith("image")) &&
		(FILE_TYPE_BROWSERSAFE as readonly string[]).includes(file.type)
	);
};

const previewableMedias = computed(() =>
	props.mediaList.filter((media) => previewable(media)),
);

function onGridClick(ev: MouseEvent): void {
	const anchor = (ev.target as HTMLElement | null)?.closest("a");
	if (anchor == null) return;
	const href = anchor.getAttribute("href");
	const index = previewableMedias.value.findIndex(
		(media) => resolveMediaUrl(media.url) === href,
	);
	if (index < 0) return;
	ev.preventDefault();
	void popup(
		CfMediaLightbox,
		{ medias: previewableMedias.value, index },
		{},
		"closed",
	);
}

const previewableCount = computed(() => previewableMedias.value.length);
</script>

<style lang="scss" scoped>
.hoawjimk {
	> .dmWidth {
		min-width: 20rem;
		max-width: 40rem;
	}

	> .grid-container {
		position: relative;
		width: 100%;
		margin-top: 4px;
		border-radius: var(--radius);
		overflow: hidden;
		pointer-events: none;

		&[data-count] {
			padding-top: 56.25%; // 16:9;
			> div {
				position: absolute;
				inset: 0;
			}
		}

		&[data-count="1"] > div {
			grid-template-rows: 1fr;
		}

		&[data-count="2"] > div {
			grid-template-columns: 1fr 1fr;
			grid-template-rows: 1fr;
		}

		&[data-count="3"] > div {
			grid-template-columns: 1fr 0.5fr;
			grid-template-rows: 1fr 1fr;

			> *:nth-child(1) {
				grid-row: 1 / 3;
			}

			> *:nth-child(3) {
				grid-column: 2 / 3;
				grid-row: 2 / 3;
			}
		}

		&[data-count="4"] > div {
			grid-template-columns: 1fr 1fr;
			grid-template-rows: 1fr 1fr;
		}

		&:not([data-count]) > div > div {
			max-height: 300px;
		}

		> div {
			display: grid;
			grid-gap: 8px;

			> div,
			> button {
				overflow: hidden;
				border-radius: 6px;
				pointer-events: all;
				min-height: 50px;
			}

			> :nth-child(1) {
				grid-column: 1 / 2;
				grid-row: 1 / 2;
			}

			> :nth-child(2) {
				grid-column: 2 / 3;
				grid-row: 1 / 2;
			}

			> :nth-child(3) {
				grid-column: 1 / 2;
				grid-row: 2 / 3;
			}

			> :nth-child(4) {
				grid-column: 2 / 3;
				grid-row: 2 / 3;
			}
		}
	}
}
</style>
