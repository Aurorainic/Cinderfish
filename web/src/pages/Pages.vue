<!-- Ported from firefish@b425f61:packages/client/src/pages/pages.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="[]"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<CfTab v-model="tab" :tabs="tabItems" class="tabs" />

			<!-- featured -->
			<section v-if="tab === 'featured'" class="pane">
				<CfLoading v-if="featuredLoading && featured.length === 0" />
				<CfInfo v-else-if="featuredError !== null" warn>
					{{ featuredError }}
				</CfInfo>
				<CfInfo v-else-if="featured.length === 0">
					{{ t("nothing") }}
				</CfInfo>
				<div v-else class="list">
					<MkPagePreview
						v-for="entry in featured"
						:key="entry.id"
						:page="entry"
						class="item"
					/>
				</div>
			</section>

			<!-- liked -->
			<section v-else-if="tab === 'liked'" class="pane">
				<CfPagination :pagination="likedPagination">
					<template #empty>
						<CfInfo>{{ t("nothing") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<div class="list">
							<MkPagePreview
								v-for="like in asLikes(items)"
								:key="like.id"
								:page="like.page"
								class="item"
							/>
						</div>
					</template>
				</CfPagination>
			</section>

			<!-- my pages -->
			<section v-else-if="tab === 'my'" class="pane">
				<div class="_formBlock">
					<CfButton primary @click="create">
						<i class="ph-plus ph-bold ph-lg"></i>
						New page
					</CfButton>
				</div>
				<CfPagination :pagination="myPagination">
					<template #empty>
						<CfInfo>{{ t("nothing") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<div class="list">
							<MkPagePreview
								v-for="entry in asPages(items)"
								:key="entry.id"
								:page="entry"
								class="item"
							/>
						</div>
					</template>
				</CfPagination>
			</section>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish renders featured / liked / my as a swiper. The swiper dependency is
// outside the port contract, so CfTab shows the same three panes one at a time.
// `pages/featured` takes no parameters and always answers the top ten, so it is
// fetched directly instead of through CfPagination; the other two are ordinary
// cursor lists. The liked and my tabs need a credential and disappear for
// guests, with a sign-out falling back to featured.
import { computed, onMounted, ref, watch } from "vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { Page, PageLike } from "@/api/types";
import { me } from "@/account";
import MkPagePreview from "@/components/page/MkPagePreview.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import type { CfTabItem } from "@/components/ui/CfTab.vue";
import type { Paging } from "@/components/ui";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import CfTab from "@/components/ui/CfTab.vue";
import { t } from "@/i18n";
import { router } from "@/router";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPagePages" });

type PagesTab = "featured" | "liked" | "my";

const signedIn = computed(() => me.value != null);

const tab = ref<PagesTab>("featured");

const tabItems = computed<CfTabItem[]>(() => [
	{
		value: "featured",
		label: t("featured"),
		icon: "ph-fire-simple",
	},
	...(signedIn.value
		? [
				{
					value: "liked",
					label: "Liked",
					icon: "ph-heart",
				},
				{
					value: "my",
					label: "My pages",
					icon: "ph-crown-simple",
				},
			]
		: []),
]);

watch(signedIn, (value) => {
	if (!value && tab.value !== "featured") tab.value = "featured";
});

const featured = ref<Page[]>([]);
const featuredLoading = ref(false);
const featuredError = ref<string | null>(null);

const likedPagination: Paging = {
	endpoint: ENDPOINTS.iPageLikes,
	limit: 10,
};

const myPagination: Paging = {
	endpoint: ENDPOINTS.iPages,
	limit: 10,
};

const headerActions = computed(() => [
	{
		icon: "ph-plus ph-bold ph-lg",
		text: t("create"),
		handler: create,
	},
]);

/** CfPagination's slot is generic over `{ id: string }`. */
function asLikes(items: { id: string }[]): PageLike[] {
	return items as PageLike[];
}

function asPages(items: { id: string }[]): Page[] {
	return items as Page[];
}

function create(): void {
	void router.push("/pages/new");
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function loadFeatured(): Promise<void> {
	featuredLoading.value = true;
	featuredError.value = null;
	try {
		featured.value = await api(ENDPOINTS.pagesFeatured, {});
	} catch (cause) {
		featured.value = [];
		featuredError.value = errorText(cause);
	} finally {
		featuredLoading.value = false;
	}
}

onMounted(() => {
	void loadFeatured();
});

definePageMetadata(
	computed(() => ({
		title: t("pages"),
		icon: "ph-file-text ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.tabs {
	margin-block-end: var(--margin);
}

.pane {
	min-block-size: 120px;
}

.list {
	> .item + .item {
		margin-block-start: 8px;
	}
}
</style>
