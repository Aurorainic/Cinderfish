// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/index.photos.vue
//
// Adaptations: blurhash images are dropped (Cinderfish has no blurhash
// decoder component), NSFW exclusion follows defaultStore.nsfw, and the
// MkContainer shell is inlined because the port has no container component:
// the panel, its sticky header and Firefish's 8px body padding are kept.
<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { DriveFile, Note, UserDetailed } from "../../api/types";
import AppIcon from "../../components/AppIcon.vue";
import MkA from "../../components/ui/MkA.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
import { isImageType } from "../../lib/mfm";
import { defaultStore } from "../../store/firefish-store";

const props = defineProps<{ user: UserDetailed }>();

const IMAGE_LIMIT = 30;
const NOTE_LIMIT = 30;

interface Photo {
	note: Note;
	file: DriveFile;
}

const fetching = ref(true);
const images = ref<Photo[]>([]);

function thumbnail(file: DriveFile): string {
	return file.thumbnailUrl || file.url;
}

const showNsfw = computed(() => defaultStore.reactiveState.nsfw.value === "ignore");

onMounted(async () => {
	try {
		const notes = await api.usersNotes({
			userId: props.user.id,
			limit: NOTE_LIMIT,
			withReplies: false,
		});
		const out: Photo[] = [];
		for (const note of notes) {
			for (const file of note.files) {
				if (!isImageType(file.type)) continue;
				if (file.isSensitive && !showNsfw.value) continue;
				out.push({ note, file });
				if (out.length >= IMAGE_LIMIT) break;
			}
			if (out.length >= IMAGE_LIMIT) break;
		}
		images.value = out;
	} finally {
		fetching.value = false;
	}
});
</script>

<template>
	<section class="photos">
		<header class="head">
			<AppIcon name="image" class="head-icon" />
			<span>图片</span>
		</header>
		<div class="body">
			<MkLoading v-if="fetching" />
			<div v-else-if="images.length > 0" class="stream">
				<MkA
					v-for="image in images"
					:key="image.note.id + image.file.id"
					class="img"
					:to="`/notes/${image.note.id}`"
					:title="image.file.name"
				>
					<img :src="thumbnail(image.file)" :alt="image.file.name" loading="lazy" />
				</MkA>
			</div>
			<p v-else class="empty">没有图片</p>
		</div>
	</section>
</template>

<style scoped>
.photos {
	overflow: clip;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
}

.head {
	display: flex;
	align-items: center;
	gap: 6px;
	padding: 10px 16px;
	border-block-end: 1px solid var(--divider);
	background: var(--panelHeaderBg);
	color: var(--panelHeaderFg);
	font-size: 0.9em;
	font-weight: bold;
	line-height: 1.4;
}

.head-icon {
	inline-size: 15px;
	block-size: 15px;
	transform: translateY(0.05em);
}

/* Firefish pads the photo stream by 8px inside the container body. */
.body {
	padding: 8px;
	min-inline-size: 0;
}

.body :deep(.cinderfish-loading) {
	padding-block: 16px;
}

.stream {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
	gap: 6px;
}

.img {
	display: block;
	block-size: 110px;
	border-radius: 6px;
	overflow: clip;
	background: var(--panelHighlight);
}

.img img {
	inline-size: 100%;
	block-size: 100%;
	object-fit: cover;
	transition: transform 0.15s ease;
}

.img:hover img {
	transform: scale(1.04);
}

.empty {
	margin: 0;
	padding: 16px;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
	text-align: center;
}

@media (max-width: 500px) {
	.head {
		padding-inline: 12px;
	}

	.stream {
		grid-template-columns: repeat(auto-fill, minmax(88px, 1fr));
	}

	.img {
		block-size: 88px;
	}
}
</style>
