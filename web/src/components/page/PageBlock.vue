<!-- Ported from firefish@b425f61:packages/client/src/components/page/page.block.vue (AGPL-3.0-only) -->
<!-- Ported from firefish@b425f61:packages/client/src/components/page/page.text.vue (AGPL-3.0-only) -->
<!-- Ported from firefish@b425f61:packages/client/src/components/page/page.section.vue (AGPL-3.0-only) -->
<!-- Ported from firefish@b425f61:packages/client/src/components/page/page.image.vue (AGPL-3.0-only) -->
<!-- Ported from firefish@b425f61:packages/client/src/components/page/page.note.vue (AGPL-3.0-only) -->
<template>
	<div v-if="block.type === 'text'" class="cf-page-block text">
		<Mfm :text="textValue" :is-note="false" />
	</div>

	<section v-else-if="block.type === 'section'" class="cf-page-block section">
		<component :is="headingTag" v-if="sectionTitle !== ''">
			{{ sectionTitle }}
		</component>
		<div class="children">
			<PageBlock
				v-for="(child, index) in sectionChildren"
				:key="blockKey(child, index)"
				:block="child"
				:attached-files="attachedFiles"
				:depth="depth + 1"
			/>
		</div>
	</section>

	<div v-else-if="block.type === 'image'" class="cf-page-block image">
		<MkImgWithBlurhash
			v-if="image != null"
			:src="resolveMediaUrl(image.url)"
			:hash="image.blurhash"
			:alt="image.name"
			:title="image.name"
			:cover="false"
		/>
		<div v-else class="missing">
			<i class="ph-image-square ph-bold ph-lg"></i>
			<span>Image unavailable</span>
		</div>
	</div>

	<div v-else-if="block.type === 'note'" class="cf-page-block note">
		<MkNote v-if="note != null" :note="note" />
		<CfInfo v-else-if="noteError !== null" warn>{{ noteError }}</CfInfo>
		<CfLoading v-else />
	</div>

	<!--
		Static-rendering contract. At the pinned upstream commit (Misskey
		c05e844) the client's own page viewer only implements text / section /
		image / note; every dynamic type (button, if, inputs, switch, counter,
		post, canvas, …) renders a "dynamic block unavailable" placeholder, and
		the editor offers only those four types with `script: ''` and
		`variables: []`. Cinderfish reproduces that exactly: a Firefish-authored
		dynamic page round-trips through the backend but is painted here as a
		placeholder instead of being evaluated.
	-->
	<div v-else class="cf-page-block unsupported">
		<i class="ph-puzzle-piece ph-bold ph-lg"></i>
		<span>
			<strong>{{ block.type }}</strong> blocks are not rendered by
			Cinderfish. The stored block is preserved if the page is edited.
		</span>
	</div>
</template>

<script setup lang="ts">
// Dispatches one page block onto the statically rendered subset. Section
// children recurse through this same component; an unknown type falls through
// to the placeholder above rather than evaluating anything.
import { computed, ref, watch } from "vue";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile, NoteDetailed, PageBlock } from "@/api/types";
import MkImgWithBlurhash from "@/components/MkImgWithBlurhash.vue";
import MkNote from "@/components/MkNote.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";

defineOptions({ name: "PageBlock" });

const props = withDefaults(
	defineProps<{
		block: PageBlock;
		/** The page's packed `attachedFiles`, used to resolve image blocks. */
		attachedFiles?: DriveFile[];
		/** Nesting level; the top level is 1. */
		depth?: number;
	}>(),
	{
		attachedFiles: () => [],
		depth: 1,
	},
);

const textValue = computed<string>(() =>
	typeof props.block.text === "string" ? props.block.text : "",
);

const sectionTitle = computed<string>(() =>
	typeof props.block.title === "string" ? props.block.title : "",
);

const sectionChildren = computed<PageBlock[]>(() => {
	const value = props.block.children;
	return Array.isArray(value)
		? value.filter((child): child is PageBlock => child != null && typeof child === "object")
		: [];
});

const headingTag = computed<string>(() => {
	const level = Math.min(props.depth + 1, 6);
	return `h${level}`;
});

const image = computed<DriveFile | null>(() => {
	const fileId = typeof props.block.fileId === "string" ? props.block.fileId : "";
	if (fileId === "") return null;
	return props.attachedFiles.find((file) => file.id === fileId) ?? null;
});

const noteId = computed<string>(() => {
	if (typeof props.block.note === "string") return props.block.note;
	// Firefish blocks spell it `note`; accept `noteId` for documents written by
	// other clients without changing what the editor sends.
	return typeof props.block.noteId === "string" ? props.block.noteId : "";
});

const note = ref<NoteDetailed | null>(null);
const noteError = ref<string | null>(null);

function blockKey(block: PageBlock, index: number): string {
	return typeof block.id === "string" ? block.id : `${props.depth}-${index}-${block.type}`;
}

async function loadNote(): Promise<void> {
	if (noteId.value === "") {
		note.value = null;
		noteError.value = null;
		return;
	}
	note.value = null;
	noteError.value = null;
	try {
		note.value = await api(ENDPOINTS.notesShow, { noteId: noteId.value });
	} catch (cause) {
		noteError.value = t("somethingHappened");
		if (import.meta.env.DEV) console.warn("[page] notes/show failed", cause);
	}
}

watch(noteId, () => void loadNote(), { immediate: true });
</script>

<style lang="scss" scoped>
.cf-page-block {
	& + & {
		margin-block-start: 0.75em;
	}

	&.text {
		overflow-wrap: anywhere;
	}

	&.section {
		margin-block: 1.5em;

		> h2 {
			margin-block: 0 0.5em;
			font-size: 1.35em;
		}

		> h3,
		> h4,
		> h5,
		> h6 {
			margin-block: 0 0.5em;
			font-size: 1em;
		}
	}

	&.image {
		display: flex;
		justify-content: center;

		img {
			display: block;
			max-inline-size: 100%;
			max-block-size: 600px;
			object-fit: contain;
		}

		.missing {
			display: flex;
			align-items: center;
			gap: 8px;
			padding: 24px;
			color: var(--fgTransparentWeak);
		}
	}

	&.note {
		margin-block: 1em;
	}

	&.unsupported {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 16px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
		background: var(--panelHighlight);
		border: dashed 1px var(--divider);
		border-radius: var(--radius);
	}
}
</style>
