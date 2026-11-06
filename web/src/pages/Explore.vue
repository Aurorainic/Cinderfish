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
			<template v-if="tab === 'users'">
				<CfLoading v-if="usersFetching && users.length === 0" />
				<MkError v-else-if="usersError !== null && users.length === 0" @retry="loadUsers" />
				<CfInfo v-else-if="users.length === 0">还没有可展示的账号。</CfInfo>
				<template v-else>
					<div class="userList">
						<MkUserInfo
							v-for="user in users"
							:key="user.id"
							class="user"
							:user="user"
						/>
					</div>
					<div v-if="usersHasMore" class="more">
						<CfButton v-if="!usersFetching" @click="loadMoreUsers">
							加载更多
						</CfButton>
						<CfLoading v-else mini />
					</div>
				</template>
			</template>
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
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { HashtagTrend } from "@/api/types";
import MkNotes from "@/components/MkNotes.vue";
import MkUserInfo from "@/components/MkUserInfo.vue";
import MkError from "@/components/global/MkError.vue";
import CfButton from "@/components/ui/CfButton.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import type { UserDetailed } from "@/api/types";
import type { Paging } from "@/components/ui";
import { t } from "@/i18n";
import { instance } from "@/instance";
import { me } from "@/account";
import { definePageMetadata } from "@/scripts/page-metadata";

type ExploreTab = "users" | "tags" | "local" | "global";

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
		key: "users",
		title: t("users"),
		icon: "ph-users-three ph-bold ph-lg",
	},
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

const tab = ref<ExploreTab>("users");

// `/explore/tags/:tag` resolves to this page (router/routes.ts), so a tag in the
// path selects the trend pane and marks the matching row.
const highlightedTag = computed(() => {
	const tag = route.params.tag;
	return typeof tag === "string" ? tag.toLowerCase() : null;
});

onMounted(() => {
	void loadUsers();
});

function onTabUpdate(key: string): void {
	if (key === "users" || key === "tags" || key === "local" || key === "global") {
		const changed = tab.value !== key;
		tab.value = key;
		if (key === "users" && changed && users.value.length === 0) void loadUsers();
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

// `users` is offset-paginated (Misskey has no cursor for it), so this pane
// keeps its own offset instead of going through CfPagination's id cursor.
const USERS_LIMIT = 20;
const users = ref<UserDetailed[]>([]);
const usersFetching = ref(false);
const usersError = ref<string | null>(null);
const usersOffset = ref(0);
const usersHasMore = ref(false);

async function loadUsers(): Promise<void> {
	usersFetching.value = true;
	usersError.value = null;
	usersOffset.value = 0;
	try {
		const page = await api("users", {
			limit: USERS_LIMIT + 1,
			offset: 0,
			sort: "+follower",
			state: "all",
			origin: "local",
		});
		usersHasMore.value = page.length > USERS_LIMIT;
		users.value = page.slice(0, USERS_LIMIT);
		usersOffset.value = users.value.length;
	} catch (cause) {
		users.value = [];
		usersError.value = cause instanceof Error ? cause.message : String(cause);
	} finally {
		usersFetching.value = false;
	}
}

async function loadMoreUsers(): Promise<void> {
	if (usersFetching.value) return;
	usersFetching.value = true;
	usersError.value = null;
	try {
		const page = await api("users", {
			limit: USERS_LIMIT + 1,
			offset: usersOffset.value,
			sort: "+follower",
			state: "all",
			origin: "local",
		});
		usersHasMore.value = page.length > USERS_LIMIT;
		users.value = [...users.value, ...page.slice(0, USERS_LIMIT)];
		usersOffset.value = users.value.length;
	} catch (cause) {
		usersError.value = cause instanceof Error ? cause.message : String(cause);
	} finally {
		usersFetching.value = false;
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
.userList {
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.user {
	padding: 12px 14px;
	background: var(--panel);
	border-radius: var(--radius);
}

.more {
	display: flex;
	justify-content: center;
	padding: 12px 0;
}

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
