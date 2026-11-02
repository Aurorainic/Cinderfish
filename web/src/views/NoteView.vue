// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/note.vue
//
// Dropped versus the original: the newer/older pagination (notes/children
// does not exist in Cinderfish, so hasPrev/hasNext and the MkNotes loaders
// are gone) and MkRemoteCaution (no remote-note rendering yet — TODO). The
// renote-unwrapping (appearNote) and the users/notes context structure are
// kept; the fallback note list uses the existing PagedFeed primitive.
<template>
	<div class="fcuexfpr">
		<template v-if="appearNote">
			<NoteDetailed
				:key="appearNote.id"
				:note="appearNote"
				class="note"
				@deleted="onDeleted"
			/>
			<section class="more">
				<h2 class="more-head">
					更多来自 @{{ appearNote.user.username }}
				</h2>
				<PagedFeed
					ref="feed"
					:key="appearNote.id"
					:fetcher="fetcher"
					:page-size="MORE_PAGE_SIZE"
				>
					<template #item="{ item }">
						<NoteItem
							:note="(item as Note)"
							@reply="onReply"
							@deleted="onFeedDeleted"
						/>
					</template>
					<template #empty>
						<span>没有更早的笔记了。</span>
					</template>
				</PagedFeed>
			</section>
		</template>
		<MkError v-else-if="error" @retry="load()" />
		<MkLoading v-else />
	</div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note } from "../api/types";
import NoteDetailed from "../components/note/NoteDetailed.vue";
import NoteItem from "../components/note/Note.vue";
import MkError from "../components/ui/MkError.vue";
import MkLoading from "../components/ui/MkLoading.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams, PagedFeedResult } from "../components/ui/PagedFeed.vue";
import { usePageTitle } from "../composables/use-page-title";
import { openPostForm } from "../os";

const route = useRoute();
const router = useRouter();

const MORE_PAGE_SIZE = 10;

const note = ref<Note | null>(null);
const loading = ref(false);
const error = ref<unknown>(null);
const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

// pages/note.vue: a pure renote shows the target note instead.
const isRenote = computed(() => {
	const n = note.value;
	return (
		n != null &&
		n.renote != null &&
		n.text == null &&
		n.fileIds.length === 0 &&
		n.poll == null
	);
});
const appearNote = computed(() =>
	isRenote.value ? ((note.value!.renote ?? null) as Note | null) : note.value,
);

const authorName = computed(
	() => appearNote.value?.user.name || appearNote.value?.user.username || "",
);
usePageTitle(() => (authorName.value ? `${authorName.value} 的笔记` : "笔记"));

async function load() {
	const id = String(route.params.id ?? "");
	if (!id) return;
	loading.value = true;
	error.value = null;
	try {
		note.value = await api.showNote(id);
	} catch (err) {
		note.value = null;
		error.value = err;
	} finally {
		loading.value = false;
	}
}

// Older notes by the same author are the only paginated context the backend
// offers: the reply tree needs notes/children, which does not exist yet.
async function fetcher(p: PagedFeedParams): Promise<PagedFeedResult> {
	const current = appearNote.value;
	if (!current) return { items: [], hasMore: false };
	const page = await api.usersNotes({
		userId: current.userId,
		limit: MORE_PAGE_SIZE,
		untilId: p.untilId,
	});
	return {
		items: page.filter((item) => item.id !== current.id),
		hasMore: page.length >= MORE_PAGE_SIZE,
	};
}

function onDeleted(id: string) {
	if (appearNote.value?.id === id) router.push("/");
}

function onFeedDeleted(id: string) {
	feed.value?.remove(id);
}

function onReply(reply: Note) {
	openPostForm({ reply });
}

watch(() => route.params.id, load, { immediate: true });
</script>

<style scoped>
.fcuexfpr {
	background: var(--bg);
}

.fcuexfpr > .note {
	border-radius: var(--radius);
	background: var(--panel);
	overflow: clip;
}

.fcuexfpr .more {
	margin-block-start: 16px;
	min-inline-size: 0;
}

.fcuexfpr .more-head {
	padding: 0 16px 8px;
	color: var(--fgTransparentWeak);
	font-size: 0.85em;
	font-weight: 700;
	letter-spacing: 0.04em;
}
</style>
