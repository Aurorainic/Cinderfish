// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/note/MkNoteMediaList.vue
// and MkNoteMedia.vue (merged into one component).
//
// Dropped versus the original, with reasons:
// - VuePlyr video player: no plyr dependency (TODO: custom video chrome when
//   one exists; a bare native video element is used meanwhile).
// - ImgWithBlurhash: no blurhash decoder, so the sensitive cover falls back
//   to a plain dimmed panel.
// - The GIF badge and alt-text popup button (the caption degrades to a title
//   attribute).
// - The media-proxy URL rewriting (Cinderfish serves media directly).
<template>
	<div class="media-list" :class="`grid-${Math.min(count, 4)}`">
		<template v-for="media in mediaList" :key="media.id">
			<button
				v-if="hideState.get(media.id)"
				type="button"
				class="media hidden"
				@click="hideState.set(media.id, false)"
			>
				<div class="hidden-text">
					<b><AppIcon name="warning" /> {{ i18n.ts.sensitive }}</b>
					<span>{{ i18n.ts.clickToShow }}</span>
				</div>
			</button>
			<div v-else-if="media.type.startsWith('image')" class="media image">
				<a :href="media.url" target="_blank" rel="noopener" :title="media.comment ?? undefined">
					<img :src="media.thumbnailUrl ?? media.url" :alt="media.comment ?? ''" loading="lazy" />
				</a>
				<button type="button" class="rehide" :title="i18n.ts._cw.hide" @click.stop="hideState.set(media.id, true)">
					<AppIcon name="eyeSlash" />
				</button>
			</div>
			<div v-else-if="media.type.startsWith('video')" class="media video">
				<!-- TODO: Cinderfish has no video player dependency; native controls
				     are used instead of Firefish's plyr chrome. -->
				<video
					:poster="media.thumbnailUrl ?? undefined"
					:aria-label="media.comment ?? undefined"
					preload="none"
					controls
					playsinline
					@contextmenu.stop
				>
					<source :src="media.url" :type="media.type === 'video/quicktime' ? 'video/mp4' : media.type" />
				</video>
			</div>
		</template>
	</div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import type { DriveFile } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import { i18n } from "../../i18n";
import { defaultStore } from "../../store/firefish-store";

const props = defineProps<{
	mediaList: DriveFile[];
}>();

const count = computed(() => props.mediaList.length);

// Mirrors the original's per-file `hide` ref initialised from the nsfw
// setting: "force" hides everything, "ignore" hides nothing, "respect" (the
// default) hides sensitive files only.
const hideState = reactive(
	new Map<string, boolean>(
		props.mediaList.map((media) => [
			media.id,
			defaultStore.reactiveState.nsfw.value === "force" ||
				(media.isSensitive && defaultStore.reactiveState.nsfw.value !== "ignore"),
		]),
	),
);
</script>

<style scoped>
.media-list {
	display: grid;
	gap: 4px;
	margin-block-start: 0.4em;
	border-radius: var(--radius, 8px);
	overflow: clip;
}

.grid-1 {
	grid-template-columns: 1fr;
}

.grid-2,
.grid-3,
.grid-4 {
	grid-template-columns: 1fr 1fr;
}

.grid-3 > .media:first-child {
	grid-column: span 2;
}

.media {
	position: relative;
	background: var(--bg);
	min-block-size: 0;
}

.media img,
.media video {
	inline-size: 100%;
	block-size: 100%;
	max-block-size: 340px;
	object-fit: cover;
	display: block;
}

.grid-1 .media img,
.grid-1 .media video {
	max-block-size: 420px;
}

.media.hidden {
	all: unset;
	position: relative;
	inline-size: 100%;
	block-size: 200px;
	display: flex;
	justify-content: center;
	align-items: center;
	cursor: pointer;
}

.hidden-text {
	position: relative;
	z-index: 1;
	inline-size: 100%;
	block-size: 100%;
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	gap: 4px;
	padding: 30px;
	box-sizing: border-box;
	background: rgba(0, 0, 0, 0.5);
	color: #fff;
	font-size: 0.8em;
	text-align: center;
}

.media.hidden:focus-visible {
	outline: 2px solid var(--accent);
}

.rehide {
	position: absolute;
	inset-block-start: 6px;
	inset-inline-end: 6px;
	border: 0;
	border-radius: 6px;
	padding: 4px;
	background: rgba(0, 0, 0, 0.5);
	color: #fff;
	cursor: pointer;
	opacity: 0;
	transition: opacity 0.2s;
}

.media:hover .rehide,
.rehide:focus-visible {
	opacity: 1;
}
</style>
