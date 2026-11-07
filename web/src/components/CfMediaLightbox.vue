<!-- Cinderfish: replaces firefish@b425f61:packages/client/src/components/MkMediaList.vue's PhotoSwipe lightbox (AGPL-3.0-only) -->
<template>
	<div class="cf-lightbox" @click.self="close">
		<button class="close" :aria-label="t('close')" @click="close">
			<i class="ph-x ph-bold ph-xl"></i>
		</button>

		<button
			v-if="medias.length > 1"
			class="nav prev"
			aria-label="上一张"
			@click.stop="prev"
		>
			<i class="ph-caret-left ph-bold ph-xl"></i>
		</button>

		<div class="stage" @click.stop>
			<img
				v-if="current != null && current.type.startsWith('image')"
				:src="resolveMediaUrl(current.url)"
				:alt="current.comment ?? ''"
			/>
			<video
				v-else-if="current != null"
				:src="resolveMediaUrl(current.url)"
				:poster="current.thumbnailUrl == null ? undefined : resolveMediaUrl(current.thumbnailUrl)"
				controls
				autoplay
			></video>
		</div>

		<button
			v-if="medias.length > 1"
			class="nav next"
			aria-label="下一张"
			@click.stop="next"
		>
			<i class="ph-caret-right ph-bold ph-xl"></i>
		</button>

		<div v-if="medias.length > 1" class="counter">
			{{ index + 1 }} / {{ medias.length }}
		</div>
	</div>
</template>

<script setup lang="ts">
// PhotoSwipe is not a dependency (contract §2), so the gallery viewer is a
// small in-house overlay: click outside or Escape closes, arrow keys and the
// side buttons page through the media list.
import { computed, onMounted, onUnmounted, ref } from "vue";
import type { DriveFile } from "@/api/types";
import { resolveMediaUrl } from "@/api/client";
import { t } from "@/i18n";

const props = defineProps<{
	medias: DriveFile[];
	index: number;
}>();

const emit = defineEmits<{ closed: [] }>();

const index = ref(props.index);
const current = computed<DriveFile | null>(() => props.medias[index.value] ?? null);

function close(): void {
	emit("closed");
}

function prev(): void {
	if (props.medias.length === 0) return;
	index.value = (index.value - 1 + props.medias.length) % props.medias.length;
}

function next(): void {
	if (props.medias.length === 0) return;
	index.value = (index.value + 1) % props.medias.length;
}

function onKeydown(ev: KeyboardEvent): void {
	if (ev.key === "Escape") {
		ev.stopPropagation();
		close();
	} else if (ev.key === "ArrowLeft") {
		ev.preventDefault();
		prev();
	} else if (ev.key === "ArrowRight") {
		ev.preventDefault();
		next();
	}
}

onMounted(() => {
	window.addEventListener("keydown", onKeydown);
	document.documentElement.style.overflow = "hidden";
});

onUnmounted(() => {
	window.removeEventListener("keydown", onKeydown);
	document.documentElement.style.overflow = "";
});
</script>

<style lang="scss" scoped>
.cf-lightbox {
	position: fixed;
	inset: 0;
	z-index: 3000;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgb(0 0 0 / 85%);
	animation: fade-in 0.15s ease-out;

	> .close,
	> .nav {
		position: absolute;
		z-index: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		border-radius: 50%;
		color: #fff;
		background: rgb(255 255 255 / 12%);
		transition: background 0.15s;

		&:hover {
			background: rgb(255 255 255 / 24%);
		}
	}

	> .close {
		top: 16px;
		right: 16px;
	}

	> .nav {
		top: 50%;
		transform: translateY(-50%);

		&.prev {
			left: 16px;
		}

		&.next {
			right: 16px;
		}
	}

	> .stage {
		display: flex;
		align-items: center;
		justify-content: center;
		max-width: calc(100vw - 128px);
		max-height: 100vh;

		> img,
		> video {
			display: block;
			max-width: 100%;
			max-height: 90vh;
			object-fit: contain;
		}
	}

	> .counter {
		position: absolute;
		bottom: 16px;
		left: 50%;
		transform: translateX(-50%);
		color: #fff;
		font-size: 0.85em;
		opacity: 0.8;
	}
}

@keyframes fade-in {
	from {
		opacity: 0;
	}
	to {
		opacity: 1;
	}
}
</style>
