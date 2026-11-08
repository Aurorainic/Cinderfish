<!-- Ported from firefish@b425f61:packages/client/src/pages/gallery/index.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="[]"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="1200" :margin-min="16" :margin-max="32">
			<CfTab v-model="tab" :tabs="tabItems" class="tabs" />

			<!-- explore -->
			<section v-if="tab === 'explore'" class="pane">
				<h2 class="sectionTitle">
					<i class="ph-clock ph-bold ph-lg"></i>
					{{ t("recentPosts") }}
				</h2>
				<CfPagination :pagination="recentPagination">
					<template #empty>
						<CfInfo>{{ t("nothing") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<div class="grid">
							<MkGalleryPostPreview
								v-for="post in asPosts(items)"
								:key="post.id"
								:post="post"
							/>
						</div>
					</template>
				</CfPagination>

				<h2 class="sectionTitle">
					<i class="ph-fire-simple ph-bold ph-lg"></i>
					{{ t("popularPosts") }}
				</h2>
				<CfLoading v-if="popularLoading && popular.length === 0" />
				<CfInfo v-else-if="popularError !== null" warn>
					{{ popularError }}
				</CfInfo>
				<CfInfo v-else-if="popular.length === 0">
					{{ t("nothing") }}
				</CfInfo>
				<div v-else class="grid">
					<MkGalleryPostPreview
						v-for="post in popular"
						:key="post.id"
						:post="post"
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
						<div class="grid">
							<MkGalleryPostPreview
								v-for="like in asLikes(items)"
								:key="like.id"
								:post="like.post"
							/>
						</div>
					</template>
				</CfPagination>
			</section>

			<!-- my posts -->
			<section v-else-if="tab === 'my'" class="pane">
				<div class="_formBlock">
					<CfButton primary @click="create">
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("postToGallery") }}
					</CfButton>
				</div>
				<CfPagination :pagination="myPagination">
					<template #empty>
						<CfInfo>{{ t("nothing") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<div class="grid">
							<MkGalleryPostPreview
								v-for="post in asPosts(items)"
								:key="post.id"
								:post="post"
							/>
						</div>
					</template>
				</CfPagination>
			</section>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish pages gallery as a swiper of explore / liked / my. The swiper is not
// installed (port contract §5), so CfTab renders the same three panes one at a
// time. The explore pane keeps Firefish's two blocks: `gallery/posts` (the
// newest posts) and a popular strip. Firefish filled the strip from
// `gallery/featured`; the backend also exposes the fixed top-ten
// `gallery/popular`, which is the literal "popular" list, so that is used here.
// The liked and my tabs are credential-only and disappear for guests.
import { computed, onMounted, ref, watch } from "vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { GalleryLike, GalleryPost } from "@/api/types";
import { me } from "@/account";
import MkGalleryPostPreview from "@/components/MkGalleryPostPreview.vue";
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

defineOptions({ name: "CfPageGalleryIndex" });

type GalleryTab = "explore" | "liked" | "my";

const signedIn = computed(() => me.value != null);

const tab = ref<GalleryTab>("explore");

const tabItems = computed<CfTabItem[]>(() => [
	{
		value: "explore",
		label: t("gallery"),
		icon: "ph-image-square",
	},
	...(signedIn.value
		? [
				{
					value: "liked",
					label: t("_gallery.liked"),
					icon: "ph-heart",
				},
				{
					value: "my",
					label: t("_gallery.my"),
					icon: "ph-crown-simple",
				},
			]
		: []),
]);

// A sign-out while reading the liked/my tab would leave the pane on an
// endpoint that now answers UNAUTHENTICATED; fall back to explore instead.
watch(signedIn, (value) => {
	if (!value && tab.value !== "explore") tab.value = "explore";
});

const recentPagination: Paging = {
	endpoint: ENDPOINTS.galleryPosts,
	limit: 18,
};

const likedPagination: Paging = {
	endpoint: ENDPOINTS.iGalleryLikes,
	limit: 18,
};

const myPagination: Paging = {
	endpoint: ENDPOINTS.iGalleryPosts,
	limit: 18,
};

const popular = ref<GalleryPost[]>([]);
const popularLoading = ref(false);
const popularError = ref<string | null>(null);

async function loadPopular(): Promise<void> {
	popularLoading.value = true;
	popularError.value = null;
	try {
		popular.value = await api(ENDPOINTS.galleryPopular, {});
	} catch (cause) {
		popular.value = [];
		popularError.value = errorText(cause);
	} finally {
		popularLoading.value = false;
	}
}

const headerActions = computed(() =>
	signedIn.value
		? [
				{
					icon: "ph-plus ph-bold ph-lg",
					text: t("postToGallery"),
					handler: create,
				},
			]
		: [],
);

/** CfPagination's slot is generic over `{ id: string }`. */
function asPosts(items: { id: string }[]): GalleryPost[] {
	return items as GalleryPost[];
}

function asLikes(items: { id: string }[]): GalleryLike[] {
	return items as GalleryLike[];
}

function create(): void {
	void router.push("/gallery/new");
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

onMounted(() => {
	void loadPopular();
});

definePageMetadata(
	computed(() => ({
		title: t("gallery"),
		icon: "ph-image-square ph-bold",
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

.sectionTitle {
	display: flex;
	align-items: center;
	gap: 8px;
	margin-block: 8px 12px;
	font-size: 1em;
	color: var(--fgTransparentWeak);

	&:not(:first-child) {
		margin-block-start: 24px;
	}
}

.grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
	gap: 12px;
	margin-block-end: var(--margin);
}
</style>
