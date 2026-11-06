<!-- Ported from firefish@b425f61:packages/client/src/pages/tag.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="header">
				<span class="tag">#{{ tag }}</span>
				<span v-if="trend != null" class="stat">{{
					t("nUsersMentioned", { n: trend.usersCount })
				}}</span>
				<span v-if="trend != null" class="stat period">{{
					t("recentNDays", { n: 7 })
				}}</span>
			</div>

			<CfLoading v-if="fetching && items.length === 0" />
			<MkError v-else-if="error != null && items.length === 0" @retry="reload" />
			<template v-else>
				<CfInfo v-if="items.length === 0">
					还没有带这个标签的帖子。标签匹配不区分大小写，只搜索引过的正文里的
					<code>#标签</code>。
				</CfInfo>
				<CfDateSeparatedList v-else :items="items" direction="down">
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
// Firefish pages a tag with `notes/search-by-tag`. Cinderfish now implements
// it (jsonb tag containment + visibility, mutes and blocks), so the list comes
// from the server instead of filtering the local timeline in the browser. That
// also means remote notes carrying the tag are found when the instance
// federates, which the old client-side filter could never do.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { HashtagTrend, NoteDetailed } from "@/api/types";
import CfButton from "@/components/ui/CfButton.vue";
import CfDateSeparatedList from "@/components/ui/CfDateSeparatedList.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkError from "@/components/global/MkError.vue";
import MkNote from "@/components/MkNote.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { usePagination } from "@/composables/use-pagination";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageTag" });

const route = useRoute();

const tag = computed(() =>
	typeof route.params.tag === "string" ? route.params.tag : "",
);

const { items, fetching, moreFetching, hasMore, error, reload, loadMore } =
	usePagination<NoteDetailed>({
		endpoint: "notes/search-by-tag",
		limit: 20,
		params: () => ({ tag: tag.value }),
		// The tag watcher below owns the first load and every tag change.
		immediate: false,
	});

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
