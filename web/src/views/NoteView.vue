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
						<div class="more-empty">没有更早的笔记了。</div>
					</template>
				</PagedFeed>
			</section>
		</template>
		<div v-else-if="notFound" class="_fullinfo empty-placeholder">
			<AppIcon class="ghost" name="search" />
			<p>这条笔记不存在或已被删除。</p>
			<MkButton link to="/">回到时间线</MkButton>
		</div>
		<MkError v-else-if="error" @retry="load()" />
		<MkLoading v-else />
	</div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import type { Note } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteDetailed from "../components/note/NoteDetailed.vue";
import NoteItem from "../components/note/Note.vue";
import MkButton from "../components/ui/MkButton.vue";
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
const notFound = ref(false);
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
	notFound.value = false;
	error.value = null;
	if (!id) {
		note.value = null;
		notFound.value = true;
		return;
	}
	loading.value = true;
	try {
		note.value = await api.showNote(id);
	} catch (err) {
		note.value = null;
		// notes/show answers 404 NO_SUCH_NOTE for both a missing id and a note
		// the viewer may not see; Firefish's not-found page is the right shape
		// for that, while any other failure keeps the retryable MkError.
		if (err instanceof ApiError && (err.status === 404 || err.code === "NO_SUCH_NOTE")) {
			notFound.value = true;
		} else {
			error.value = err;
		}
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
	/* The reading column already paints --panel; notes are transparent rows,
	   so the page root must not repaint a different background. */
	min-inline-size: 0;
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

/* PagedFeed's empty wrapper lost its global class with the Firefish stylesheet
   port; keep the "no older notes" line centred instead of flush left. */
.fcuexfpr :deep(.paged-feed > .placeholder) {
	padding: 16px;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
	text-align: center;
}

.more-empty {
	margin: 0;
}

/* Missing / restricted note: Firefish's not-found page (_fullinfo) rather than
   the retryable MkError, which is kept for real request failures. */
.empty-placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 12px;
	color: var(--fgTransparentWeak);
}

.empty-placeholder > .ghost {
	inline-size: 64px;
	block-size: 64px;
	opacity: 0.45;
}

.empty-placeholder > p {
	margin: 0;
	font-size: 0.95em;
	line-height: 1.5;
}
</style>
