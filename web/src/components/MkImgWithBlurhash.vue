<!-- Ported from firefish@b425f61:packages/client/src/components/MkImgWithBlurhash.vue (AGPL-3.0-only) -->
<template>
	<img
		v-if="src"
		:src="src"
		:title="title ?? undefined"
		:alt="alt"
		:class="{ cover }"
		:style="{ 'object-fit': cover ? 'cover' : undefined }"
		loading="lazy"
	/>
</template>

<script setup lang="ts">
/**
 * Firefish paints a canvas placeholder by decoding `hash` with `fast-blurhash`
 * before the image loads.
 *
 * Cinderfish has no blurhash dependency and the API never emits one
 * (`DriveFile.blurhash` is always null), so the canvas branch is dropped. The
 * props keep their Firefish shape so callers do not have to branch.
 */
withDefaults(
	defineProps<{
		src?: string | null;
		hash?: string | null;
		alt?: string;
		type?: string | null;
		title?: string | null;
		size?: number;
		cover?: boolean;
	}>(),
	{
		src: null,
		hash: null,
		alt: "",
		type: null,
		title: null,
		size: 64,
		cover: true,
	},
);
</script>

<style lang="scss" scoped>
img {
	display: block;
	width: 100%;
	height: 100%;
	object-fit: contain;
}
</style>
