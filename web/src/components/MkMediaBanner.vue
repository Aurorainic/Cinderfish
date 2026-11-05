<!-- Ported from firefish@b425f61:packages/client/src/components/MkMediaBanner.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-media-banner" @click.stop>
		<div
			v-if="media.isSensitive && hide"
			class="sensitive"
			@click="unhide"
		>
			<span class="icon"><i class="ph-warning ph-bold ph-lg"></i></span>
			<b>{{ t("sensitive") }}</b>
			<span>{{ t("clickToShow") }}</span>
		</div>
		<div
			v-else-if="
				media.type.startsWith('audio') && media.type !== 'audio/midi'
			"
			class="audio"
		>
			<audio
				ref="audioEl"
				:src="url"
				:title="media.name"
				controls
				preload="metadata"
				@volumechange="volumechange"
			/>
		</div>
		<a
			v-else
			class="download"
			:href="url"
			:title="media.name"
			:download="media.name"
		>
			<span class="icon"
				><i class="ph-download-simple ph-bold ph-lg"></i
			></span>
			<b>{{ media.name }}</b>
		</a>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { DriveFile } from "@/api/types";
import { resolveMediaUrl } from "@/api/client";
import { t } from "@/i18n";
import { ColdDeviceStorage } from "@/store";

/**
 * Ported from Firefish. `vue-plyr` is not installed, so audio uses a native
 * `<audio controls>`; the persisted volume from `ColdDeviceStorage` is kept
 * because it is a plain element property.
 */
const props = defineProps<{
	media: DriveFile;
}>();

const audioEl = ref<HTMLAudioElement | null>(null);
const hide = ref(true);

/** Cinderfish serves files from root-relative `/media/...` paths. */
const url = resolveMediaUrl(props.media.url);

function unhide(): void {
	hide.value = false;
}

function volumechange(): void {
	if (audioEl.value != null) {
		ColdDeviceStorage.set("mediaVolume", audioEl.value.volume);
	}
}

onMounted(() => {
	if (audioEl.value != null) {
		audioEl.value.volume = ColdDeviceStorage.get("mediaVolume");
	}
});
</script>

<style lang="scss" scoped>
.mk-media-banner {
	width: 100%;
	border-radius: 4px;
	margin-top: 4px;
	overflow: hidden;

	> .download,
	> .sensitive {
		display: flex;
		align-items: center;
		font-size: 12px;
		padding: 8px 12px;
		white-space: nowrap;

		> * {
			display: block;
		}

		> b {
			overflow: hidden;
			text-overflow: ellipsis;
		}

		> *:not(:last-child) {
			margin-right: 0.2em;
		}

		> .icon {
			font-size: 1.6em;
		}
	}

	> .download {
		background: var(--noteAttachedFile);
	}

	> .sensitive {
		background: #111;
		color: #fff;
	}

	> .audio {
		audio {
			display: block;
			width: 100%;
		}
	}
}
</style>
