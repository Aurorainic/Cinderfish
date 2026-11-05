<!-- Ported from firefish@b425f61:packages/client/src/components/MkMedia.vue (AGPL-3.0-only) -->
<template>
	<div v-size="{ max: [350] }" class="media">
		<button v-if="hide" class="hidden" @click="unhide">
			<ImgWithBlurhash
				:hash="media.blurhash"
				:title="media.comment"
				:alt="media.comment ?? ''"
			/>
			<div class="text">
				<div class="wrapper">
					<b style="display: block"
						><i class="ph-warning ph-bold ph-lg"></i>
						{{ t("sensitive") }}</b
					>
					<span style="display: block">{{
						t("clickToShow")
					}}</span>
				</div>
			</div>
		</button>
		<template v-else>
			<a
				v-if="media.type.startsWith('image')"
				:href="url"
				target="_blank"
				rel="noopener noreferrer"
			>
				<ImgWithBlurhash
					:hash="media.blurhash"
					:src="url"
					:alt="media.comment ?? ''"
					:type="media.type"
					:cover="false"
				/>
				<div v-if="media.type === 'image/gif'" class="gif">GIF</div>
			</a>
			<video
				v-if="media.type.startsWith('video')"
				:poster="media.thumbnailUrl ?? undefined"
				:aria-label="media.comment ?? undefined"
				preload="none"
				controls
				@contextmenu.stop
			>
				<source :src="url" :type="mediaType" />
			</video>
		</template>
		<div class="buttons">
			<button
				v-if="media.comment"
				v-tooltip.noLabel="altText"
				:aria-label="t('alt')"
				class="_button"
				@click.stop="captionPopup"
			>
				<i class="ph-subtitles ph-bold ph-lg"></i>
			</button>
			<button
				v-if="!hide"
				v-tooltip="t('hide')"
				class="_button"
				@click.stop="hideSelf"
			>
				<i class="ph-eye-slash ph-bold ph-lg"></i>
			</button>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { DriveFile } from "@/api/types";
import { resolveMediaUrl } from "@/api/client";
import { t } from "@/i18n";
import { alert } from "@/os";
import { defaultStore } from "@/store";
import ImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";

/**
 * Ported from Firefish. Degradations, all forced by the Cinderfish API contract:
 *   - vue-plyr is not installed: videos use a native video controls element;
 *   - thumbnailUrl is always null, so there is no poster and no still-frame
 *     fallback for animated images (getStaticImageUrl returns the raw URL);
 *   - PhotoSwipe is not installed: the image links to the file in a new tab.
 *
 * The raw prop keeps its Firefish shape but has no effect: it only chose
 * between url and thumbnailUrl, and thumbnailUrl is always null.
 */
const props = defineProps<{
	media: DriveFile;
	raw?: boolean;
}>();

const hide = ref(true);

/** Cinderfish serves local files from a root-relative /media/... path. */
const url = computed(() => resolveMediaUrl(props.media.url));

const mediaType = computed(() =>
	props.media.type === "video/quicktime" ? "video/mp4" : props.media.type,
);

const altText = computed(() => {
	const comment = (props.media.comment ?? "").trim();
	const short = comment.length > 200 ? comment.slice(0, 200) + "..." : comment;
	return t("alt") + ": " + short;
});

function unhide(): void {
	hide.value = false;
}

function hideSelf(): void {
	hide.value = true;
}

function captionPopup(): void {
	void alert({ type: "info", text: props.media.comment });
}

// Plugin:register_note_view_interruptor を使って書き換えられる可能性があるためwatchする
watch(
	() => props.media,
	() => {
		hide.value =
			defaultStore.state.nsfw === "force"
				? true
				: props.media.isSensitive && defaultStore.state.nsfw !== "ignore";
	},
	{
		deep: true,
		immediate: true,
	},
);
</script>

<style lang="scss" scoped>
.hidden {
	all: unset;
	position: relative;
	width: 100%;
	height: 100%;

	> .text {
		position: relative;
		width: 100%;
		height: 100%;
		z-index: 1;
		display: flex;
		justify-content: center;
		align-items: center;
		padding: 30px;
		box-sizing: border-box;
		background: rgba(0, 0, 0, 0.5);

		> .wrapper {
			display: table-cell;
			text-align: center;
			font-size: 0.8em;
			color: #fff;
		}
	}

	&:focus-visible {
		border: 2px solid var(--accent);
	}
}

.media {
	position: relative;
	background: var(--bg);

	> .buttons {
		display: flex;
		gap: 4px;
		position: absolute;
		border-radius: 6px;
		overflow: hidden;
		top: 12px;
		right: 12px;
		> * {
			background-color: var(--accentedBg);
			-webkit-backdrop-filter: var(--blur, blur(15px));
			backdrop-filter: var(--blur, blur(15px));
			color: var(--accent);
			font-size: 0.8em;
			padding: 6px 8px;
			text-align: center;
		}
	}

	> a {
		display: block;
		cursor: zoom-in;
		overflow: hidden;
		width: 100%;
		height: 100%;
		background-position: center;
		background-size: contain;
		background-repeat: no-repeat;
		box-sizing: border-box;
		&:focus-visible {
			border: 2px solid var(--accent);
		}

		> .gif {
			background-color: var(--fg);
			border-radius: 6px;
			color: var(--accentLighten);
			display: inline-block;
			font-size: 14px;
			font-weight: bold;
			left: 12px;
			opacity: 0.5;
			padding: 0 6px;
			text-align: center;
			top: 12px;
			pointer-events: none;
		}
	}

	> video {
		display: block;
		width: 100%;
		height: 100%;
		background: #000;
	}

	&.max-width_350px {
		> video {
			min-width: unset;
		}
	}
}
</style>
