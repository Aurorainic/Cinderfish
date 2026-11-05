<!-- Ported from firefish@b425f61:packages/client/src/pages/explore.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:tab="tab"
				:tabs="headerTabs"
				:actions="headerActions"
				@update:tab="onTabUpdate"
			/>
		</template>
		<MkSpacer :content-max="800">
			<div v-if="tab === 'tags'" class="tags">
				<CfLoading v-if="trendFetching && trends.length === 0" />
				<MkError v-else-if="trendFailed" @retry="fetchTrend" />
				<CfInfo v-else-if="trends.length === 0">
					近 7 天还没有本地用户使用话题标签。
				</CfInfo>
				<div v-else class="tagList">
					<MkA
						v-for="trend in trends"
						:key="trend.tag"
						class="tag _panel"
						:class="{ highlighted: trend.tag === highlightedTag }"
						:to="`/tags/${encodeURIComponent(trend.tag)}`"
					>
						<div class="body">
							<span class="name">#{{ trend.tag }}</span>
							<span class="count">{{
								t("nUsersMentioned", { n: trend.usersCount })
							}}</span>
						</div>
						<i class="ph-caret-right ph-bold"></i>
					</MkA>
				</div>
			</div>
			<MkNotes v-else-if="tab === 'local'" :pagination="localPagination" />
			<MkNotes v-else-if="tab === 'global'" :pagination="globalPagination" />
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's explore page is a swiper of "users" (users/search) and "featured"
// (notes/featured). Neither endpoint exists on Cinderfish (contract §2), so the
// page keeps the tabbed shape but the tabs become what the backend can serve:
// trending hashtags (`hashtags/trend`), the local timeline and — when the
// instance federates — the global timeline. The swiper is dropped: switching
// tabs renders one pane instead of slides.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { HashtagTrend } from "@/api/types";
import MkNotes from "@/components/MkNotes.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import type { Paging } from "@/components/ui";
import { t } from "@/i18n";
import { instance } from "@/instance";
import { me } from "@/account";
import { definePageMetadata } from "@/scripts/page-metadata";

type ExploreTab = "tags" | "local" | "global";

interface TabDefinition {
	key: ExploreTab;
	title: string;
	icon: string;
}

const route = useRoute();

const isGlobalTimelineAvailable = computed(
	() =>
		instance.features?.globalTimeline !== false ||
		me.value?.isAdmin === true ||
		me.value?.isModerator === true,
);

const headerTabs = computed<TabDefinition[]>(() => [
	{
		key: "tags",
		title: t("popularTags"),
		icon: "ph-hash ph-bold ph-lg",
	},
	{
		key: "local",
		title: t("_timelines.local"),
		icon: "ph-users ph-bold ph-lg",
	},
	...(isGlobalTimelineAvailable.value
		? [
				{
					key: "global" as const,
					title: t("_timelines.global"),
					icon: "ph-planet ph-bold ph-lg",
				},
			]
		: []),
]);

const tab = ref<ExploreTab>("tags");

// `/explore/tags/:tag` resolves to this page (router/routes.ts), so a tag in the
// path selects the trend pane and marks the matching row.
const highlightedTag = computed(() => {
	const tag = route.params.tag;
	return typeof tag === "string" ? tag.toLowerCase() : null;
});

function onTabUpdate(key: string): void {
	if (key === "tags" || key === "local" || key === "global") {
		tab.value = key;
	}
}

const trends = ref<HashtagTrend[]>([]);
const trendFetching = ref(true);
const trendFailed = ref(false);

async function fetchTrend(): Promise<void> {
	trendFetching.value = true;
	trendFailed.value = false;
	try {
		trends.value = await api("hashtags/trend", {});
	} catch {
		trendFailed.value = true;
		trends.value = [];
	} finally {
		trendFetching.value = false;
	}
}

const localPagination: Paging = {
	endpoint: "notes/local-timeline",
	limit: 20,
};

const globalPagination: Paging = {
	endpoint: "notes/global-timeline",
	limit: 20,
};

// The trend list is only rendered on the "tags" tab, so it is fetched when that
// tab is first shown instead of on mount of the whole page.
watch(
	tab,
	(value) => {
		if (value === "tags" && trends.value.length === 0 && !trendFailed.value) {
			void fetchTrend();
		}
	},
	{ immediate: true },
);

// Lists and antennas have no endpoints; Firefish's header actions are empty here
// as well, but the array is kept so the header shape matches.
const headerActions = computed<
	{ text: string; icon: string; handler: () => void }[]
>(() => []);

definePageMetadata(
	computed(() => ({
		title: t("explore"),
		icon: "ph-compass ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.tags {
	> .tagList {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 10px;

		> .tag {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 8px;
			padding: 12px 14px;
			border-radius: var(--radius);
			color: var(--fg);

			&:hover {
				background: var(--buttonBg);
			}

			&.highlighted {
				background: var(--accentedBg);
				color: var(--accent);
			}

			> .body {
				display: flex;
				flex-direction: column;
				min-width: 0;

				> .name {
					font-weight: bold;
					white-space: nowrap;
					overflow: hidden;
					text-overflow: ellipsis;
				}

				> .count {
					font-size: 0.8em;
					color: var(--fgTransparentWeak);
				}
			}

			> .ph-caret-right {
				flex-shrink: 0;
				color: var(--fgTransparentWeak);
			}
		}
	}
}
</style>
