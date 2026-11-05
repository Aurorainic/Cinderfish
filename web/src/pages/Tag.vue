<!-- Ported from firefish@b425f61:packages/client/src/pages/tag.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800">
			<div class="header">
				<span class="tag">#{{ tag }}</span>
				<span v-if="trend != null" class="stat">{{
					t("nUsersMentioned", { n: trend.usersCount })
				}}</span>
				<span v-if="trend != null" class="stat period">{{
					t("recentNDays", { n: 7 })
				}}</span>
			</div>

			<template v-if="fetching && items.length === 0">
				<CfLoading />
			</template>
			<MkError v-else-if="error != null && items.length === 0" @retry="reload" />
			<template v-else>
				<CfInfo v-if="notes.length === 0">
					最近加载的本地时间线里没有带这个标签的帖子；继续加载更多可以往下找。
				</CfInfo>
				<CfDateSeparatedList v-else :items="notes" direction="down">
					<template #default="{ item }">
						<MkNote :key="item.id" class="note" :note="item" />
					</template>
				</CfDateSeparatedList>

				<div v-if="hasMore" class="more">
					<CfButton v-if="!moreFetching" @click="loadMore">
						{{ t("loadMore") }}
					</CfButton>
					<CfLoading v-else mini />
				</div>
			</template>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish searches notes with `notes/search-by-tag` and users with
// `users/search`; Cinderfish has neither endpoint (contract §2). The page keeps
// the tag header and the trend statistic, and filters the local timeline in the
// browser by `note.tags` — the same field the backend indexes for
// `hashtags/trend`. Remote notes never appear here, matching the "local
// timeline" scope the contract prescribes.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { HashtagTrend, NoteDetailed } from "@/api/types";
import { CfButton, CfDateSeparatedList, CfInfo, CfLoading } from "@/components/ui";
import MkNote from "@/components/MkNote.vue";
import { usePagination } from "@/composables/use-pagination";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

const route = useRoute();

const tag = computed(() =>
	typeof route.params.tag === "string" ? route.params.tag : "",
);

const {
	items,
	fetching,
	moreFetching,
	hasMore,
	error,
	reload,
	loadMore,
} = usePagination<NoteDetailed>({
	endpoint: "notes/local-timeline",
	limit: 20,
	// The tag watcher below owns the first load (and every tag change), so the
	// hook must not fire its own immediate request.
	immediate: false,
});

function matchesTag(note: NoteDetailed): boolean {
	const needle = tag.value.toLowerCase();
	return note.tags.some((noteTag) => noteTag.toLowerCase() === needle);
}

const notes = computed(() => items.value.filter(matchesTag));

const trend = ref<HashtagTrend | null>(null);

async function fetchTrend(): Promise<void> {
	trend.value = null;
	try {
		const trends = await api("hashtags/trend", {});
		const needle = tag.value.toLowerCase();
		trend.value = trends.find((entry) => entry.tag === needle) ?? null;
	} catch {
		// The trend block is supplementary: a failure here leaves the note list
		// working and simply hides the statistic.
		trend.value = null;
	}
}

watch(
	tag,
	() => {
		void reload();
		void fetchTrend();
	},
	{ immediate: true },
);

definePageMetadata(
	computed(() => ({
		title: `#${tag.value}`,
		icon: "ph-hash ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.header {
	display: flex;
	align-items: baseline;
	flex-wrap: wrap;
	gap: 10px;
	padding: 12px 16px;
	margin-bottom: var(--margin);
	background: var(--panel);
	border-radius: var(--radius);

	> .tag {
		font-size: 1.2em;
		font-weight: bold;
	}

	> .stat {
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}
}

.note {
	background: var(--panel);
	border-radius: var(--radius);
}

.more {
	display: flex;
	justify-content: center;
	padding-block: var(--margin);
}
</style>
