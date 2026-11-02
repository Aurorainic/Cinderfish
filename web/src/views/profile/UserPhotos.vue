// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/index.photos.vue
//
// Adaptations: blurhash images are dropped (Cinderfish has no blurhash
// decoder component), NSFW exclusion follows defaultStore.nsfw, and the
// container is the ported ProfileSection shell in the caller.
<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { DriveFile, Note } from "../../api/types";
import MkA from "../../components/ui/MkA.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
import { isImageType } from "../../lib/mfm";
import { defaultStore } from "../../store/firefish-store";
import type { UserDetailed } from "../../api/types";

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
	<div class="photos">
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
</template>

<style scoped>
.stream {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
	gap: 6px;
}

.img {
	display: block;
	aspect-ratio: 1;
	border-radius: var(--radius-sm);
	overflow: hidden;
	background: var(--surface-inset);
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
	color: var(--text-faint);
	font-size: var(--fs-xs);
}
</style>
