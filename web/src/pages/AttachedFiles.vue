<!-- Cinderfish-only page: firefish@b425f61 has no pages/attached-files.vue. The drive
     page's layout conventions are reused; the metadata rows come from
     @/components/ui/CfKeyValue. -->
<template>
	<div class="cf-page-attached-files">
		<CfLoading v-if="loading" />
		<template v-else>
			<div v-if="file != null" class="file">
				<div class="preview">
					<img
						v-if="isImage"
						:src="resolveMediaUrl(file.url)"
						:alt="file.name"
						:title="file.name"
					/>
					<i v-else class="ph-file ph-bold ph-lg"></i>
				</div>
				<div class="meta">
					<CfKeyValue :copy="file.name">
						<template #key>{{ t("name") }}</template>
						<template #value>{{ file.name }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("type") }}</template>
						<template #value>{{ file.type }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("size") }}</template>
						<template #value>{{ bytes(file.size) }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("createdAt") }}</template>
						<template #value>{{ file.createdAt }}</template>
					</CfKeyValue>
				</div>
			</div>
			<div v-else-if="failed" class="empty">
				<p>{{ t("notFound") }}</p>
			</div>
			<div class="notes">
				<MkNote v-for="note in notes" :key="note.id" :note="note" />
				<div v-if="notes.length === 0 && !notesLoading" class="empty">
					<p>{{ t("noNotes") }}</p>
				</div>
				<div v-if="hasMore" class="more">
					<CfButton :disabled="notesLoading" @click="loadMore">
						{{ notesLoading ? "加载中…" : "加载更多" }}
					</CfButton>
				</div>
			</div>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Route: /my/drive/file/:fileId/attached.
//
// The page shows the file's metadata plus the notes that reference it, paged
// through `drive/files/attached-notes` (the endpoint Firefish's drive panel
// uses for the same panel).
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import type { DriveFile, Note } from "@/api/types";
import { api, resolveMediaUrl } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkNote from "@/components/MkNote.vue";
import bytes from "@/filters/bytes";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAttachedFiles" });

const route = useRoute();

const file = ref<DriveFile | null>(null);
const loading = ref(true);
const failed = ref(false);

const notes = ref<Note[]>([]);
const notesLoading = ref(false);
const hasMore = ref(false);

const PAGE_SIZE = 10;

const fileId = computed(() => {
	const raw = route.params.fileId;
	if (typeof raw === "string") return raw;
	if (Array.isArray(raw) && typeof raw[0] === "string") return raw[0];
	return "";
});

const isImage = computed(
	() => file.value != null && file.value.type.startsWith("image/"),
);

definePageMetadata(
	computed(() => ({
		title: file.value != null ? file.value.name : t("drive"),
		icon: "ph-file ph-bold ph-lg",
		hideHeader: true,
	})),
);

function loadNotes(reset: boolean): void {
	const id = fileId.value;
	if (id === "") return;
	notesLoading.value = true;
	const untilId = reset || notes.value.length === 0
		? undefined
		: notes.value[notes.value.length - 1].id;
	api<Note[]>("drive/files/attached-notes", {
		fileId: id,
		limit: PAGE_SIZE,
		...(untilId == null ? {} : { untilId }),
	})
		.then((page) => {
			notes.value = reset ? page : [...notes.value, ...page];
			hasMore.value = page.length === PAGE_SIZE;
		})
		.catch(() => {
			// The file metadata still renders; the note list stays as-is.
		})
		.finally(() => {
			notesLoading.value = false;
		});
}

function loadMore(): void {
	loadNotes(false);
}

onMounted(async () => {
	loadNotes(true);
	try {
		file.value = await api<DriveFile>("drive/files/show", {
			fileId: fileId.value,
		});
	} catch {
		failed.value = true;
	} finally {
		loading.value = false;
	}
});
</script>

<style lang="scss" scoped>
.cf-page-attached-files {
	display: flex;
	flex-direction: column;
	gap: 16px;
	padding: var(--margin);

	> .file {
		display: flex;
		gap: 16px;
		padding: 16px;
		background: var(--panel);
		border-radius: var(--radius);

		> .preview {
			display: flex;
			align-items: center;
			justify-content: center;
			inline-size: 96px;
			block-size: 96px;
			flex-shrink: 0;
			border-radius: 8px;
			overflow: clip;
			background: var(--bg);

			> img {
				inline-size: 100%;
				block-size: 100%;
				object-fit: contain;
			}

			> i {
				font-size: 32px;
				color: var(--fgTransparentWeak);
			}
		}

		> .meta {
			flex: 1;
			min-inline-size: 0;
		}
	}

	> .empty,
	> .notes > .empty {
		padding: 24px;
		text-align: center;
		color: var(--fgTransparentWeak);

		> p {
			margin: 0;
		}
	}
}
</style>
