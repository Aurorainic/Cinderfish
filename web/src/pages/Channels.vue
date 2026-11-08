<!-- Ported from firefish@b425f61:packages/client/src/pages/channels.vue (AGPL-3.0-only) -->
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
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<CfInfo warn class="_formBlock">
				{{ t("channelFederationWarn") }}
			</CfInfo>

			<!-- search -->
			<section v-if="tab === 'search'" class="pane">
				<form class="_formBlock searchForm" @submit.prevent="search">
					<CfInput
						v-model="searchQuery"
						type="search"
						:placeholder="t('inChannelSearch')"
						@enter="search"
					>
						<template #prefix>
							<i class="ph-magnifying-glass ph-bold ph-lg"></i>
						</template>
					</CfInput>
					<CfRadios
						:model-value="searchType"
						@update:model-value="onSearchType"
					>
						<CfRadio value="nameAndDescription">
							{{ t("_channel.nameAndDescription") }}
						</CfRadio>
						<CfRadio value="nameOnly">
							{{ t("_channel.nameOnly") }}
						</CfRadio>
					</CfRadios>
					<CfButton type="submit" primary :disabled="searching">
						{{ t("search") }}
					</CfButton>
				</form>
				<CfLoading v-if="searching && searchResults.length === 0" />
				<CfInfo v-else-if="searchError !== null" warn>
					{{ searchError }}
				</CfInfo>
				<div v-else-if="searchResults.length === 0" class="_fullinfo">
					<i class="ph-television ph-bold ph-lg"></i>
					<div>{{ t("nothing") }}</div>
				</div>
				<template v-else>
					<ChannelPreview
						v-for="channel in searchResults"
						:key="channel.id"
						:channel="channel"
					/>
				</template>
			</section>

			<!-- featured -->
			<section v-else-if="tab === 'featured'" class="pane">
				<CfLoading v-if="featuredLoading && featured.length === 0" />
				<CfInfo v-else-if="featuredError !== null" warn>
					{{ featuredError }}
				</CfInfo>
				<div v-else-if="featured.length === 0" class="_fullinfo">
					<i class="ph-television ph-bold ph-lg"></i>
					<div>{{ t("nothing") }}</div>
				</div>
				<template v-else>
					<ChannelPreview
						v-for="channel in featured"
						:key="channel.id"
						:channel="channel"
					/>
				</template>
			</section>

			<!-- following -->
			<section v-else-if="tab === 'following'" class="pane">
				<CfPagination :pagination="followedPagination">
					<template #empty>
						<div class="_fullinfo">
							<i class="ph-heart ph-bold ph-lg"></i>
							<div>{{ t("nothing") }}</div>
						</div>
					</template>
					<template #default="{ items }">
						<ChannelPreview
							v-for="channel in asChannels(items)"
							:key="channel.id"
							:channel="channel"
						/>
					</template>
				</CfPagination>
			</section>

			<!-- owned -->
			<section v-else-if="tab === 'owned'" class="pane">
				<div class="_formBlock">
					<CfButton primary @click="create">
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("_channel.create") }}
					</CfButton>
				</div>
				<CfPagination :pagination="ownedPagination">
					<template #empty>
						<div class="_fullinfo">
							<i class="ph-crown-simple ph-bold ph-lg"></i>
							<div>{{ t("nothing") }}</div>
						</div>
					</template>
					<template #default="{ items }">
						<ChannelPreview
							v-for="channel in asChannels(items)"
							:key="channel.id"
							:channel="channel"
						/>
					</template>
				</CfPagination>
			</section>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish pages the channel directory as a swiper of search / featured /
// following / owned. The swiper is dropped (tab switching renders one pane),
// and the endpoints map directly onto the backend: `channels/featured` is a
// fixed list, while `channels/followed`, `channels/owned` and
// `channels/search` are cursor lists. An empty search query lists every
// non-archived channel, which is what the search tab shows before typing.
import { computed, onMounted, ref } from "vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { ChannelSearchType } from "@/api/endpoints";
import type { Channel } from "@/api/types";
import ChannelPreview from "@/components/ChannelPreview.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { Paging } from "@/components/ui";
import { t } from "@/i18n";
import { me } from "@/account";
import { router } from "@/router";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageChannels" });

type ChannelsTab = "search" | "featured" | "following" | "owned";

interface TabDefinition {
	key: ChannelsTab;
	title: string;
	icon: string;
}

const signedIn = computed(() => me.value != null);

const tab = ref<ChannelsTab>("featured");

const headerTabs = computed<TabDefinition[]>(() => [
	{
		key: "search",
		title: t("search"),
		icon: "ph-magnifying-glass ph-bold ph-lg",
	},
	{
		key: "featured",
		title: t("_channel.featured"),
		icon: "ph-fire-simple ph-bold ph-lg",
	},
	...(signedIn.value
		? [
				{
					key: "following" as const,
					title: t("_channel.following"),
					icon: "ph-heart ph-bold ph-lg",
				},
				{
					key: "owned" as const,
					title: t("_channel.owned"),
					icon: "ph-crown-simple ph-bold ph-lg",
				},
			]
		: []),
]);

const headerActions = computed(() =>
	signedIn.value
		? [
				{
					icon: "ph-plus ph-bold ph-lg",
					text: t("_channel.create"),
					handler: create,
				},
			]
		: [],
);

// ----- featured -----------------------------------------------------------

const featured = ref<Channel[]>([]);
const featuredLoading = ref(false);
const featuredError = ref<string | null>(null);

async function loadFeatured(): Promise<void> {
	featuredLoading.value = true;
	featuredError.value = null;
	try {
		featured.value = await api(ENDPOINTS.channelsFeatured, {});
	} catch (cause) {
		featured.value = [];
		featuredError.value = errorText(cause);
	} finally {
		featuredLoading.value = false;
	}
}

// ----- search -------------------------------------------------------------

const searchQuery = ref("");
const searchType = ref<ChannelSearchType>("nameAndDescription");
const searchResults = ref<Channel[]>([]);
const searching = ref(false);
const searchError = ref<string | null>(null);
const searched = ref(false);

function onSearchType(value: string | number | boolean | null): void {
	if (value === "nameAndDescription" || value === "nameOnly") {
		searchType.value = value;
	}
}

async function search(): Promise<void> {
	if (searching.value) return;
	searching.value = true;
	searched.value = true;
	searchError.value = null;
	try {
		// The backend treats an empty query as "list all non-archived channels".
		searchResults.value = await api(ENDPOINTS.channelsSearch, {
			query: searchQuery.value.trim(),
			type: searchType.value,
			limit: 20,
		});
	} catch (cause) {
		searchResults.value = [];
		searchError.value = errorText(cause);
	} finally {
		searching.value = false;
	}
}

// ----- cursor lists -------------------------------------------------------

const followedPagination: Paging = {
	endpoint: ENDPOINTS.channelsFollowed,
	limit: 10,
};

const ownedPagination: Paging = {
	endpoint: ENDPOINTS.channelsOwned,
	limit: 10,
};

/** CfPagination's slot is generic over `{ id: string }`. */
function asChannels(items: { id: string }[]): Channel[] {
	return items as Channel[];
}

// ----- navigation ---------------------------------------------------------

function create(): void {
	void router.push("/channels/new");
}

function onTabUpdate(key: string): void {
	if (
		key !== "search" &&
		key !== "featured" &&
		key !== "following" &&
		key !== "owned"
	) {
		return;
	}
	tab.value = key;
	if (key === "search" && !searched.value) void search();
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

onMounted(() => {
	void loadFeatured();
});

definePageMetadata(
	computed(() => ({
		title: t("channel"),
		icon: "ph-television ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.pane {
	min-block-size: 120px;
}

.searchForm {
	display: flex;
	flex-direction: column;
	gap: 12px;
	align-items: flex-start;

	> :first-child {
		inline-size: 100%;
	}
}
</style>
