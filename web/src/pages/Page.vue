<!-- Ported from firefish@b425f61:packages/client/src/pages/page.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="[]" :tabs="[]" :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div v-if="page" class="cf-page">
				<div class="_block main">
					<div class="banner" :style="{ backgroundImage: bannerImage }">
						<div class="header">
							<h1>{{ page.title }}</h1>
						</div>
						<div class="menu-actions">
							<button
								v-tooltip="t('copyUrl')"
								class="menu _button"
								@click="copyUrl"
							>
								<i class="ph-link-simple ph-bold ph-lg"></i>
							</button>
							<MkA
								v-tooltip="'View source'"
								:to="sourcePath"
								class="menu _button"
							>
								<i class="ph-code ph-bold ph-lg"></i>
							</MkA>
							<template v-if="isOwner">
								<MkA
									v-tooltip="t('edit')"
									:to="`/pages/edit/${page.id}`"
									class="menu _button"
								>
									<i class="ph-pencil ph-bold ph-lg"></i>
								</MkA>
								<button
									v-if="isPinned"
									v-tooltip="t('unpin')"
									class="menu _button"
									@click="pin(false)"
								>
									<i class="ph-push-pin-slash ph-bold ph-lg"></i>
								</button>
								<button
									v-else
									v-tooltip="t('pin')"
									class="menu _button"
									@click="pin(true)"
								>
									<i class="ph-push-pin ph-bold ph-lg"></i>
								</button>
							</template>
						</div>
					</div>

					<div
						class="content"
						:class="{
							center: page.alignCenter,
							serif: page.font === 'serif',
						}"
					>
						<PageBlockView
							v-for="(block, index) in page.content"
							:key="blockKey(block, index)"
							:block="block"
							:attached-files="page.attachedFiles"
						/>
					</div>

					<div class="actions">
						<CfButton
							v-if="canLike"
							:primary="page.isLiked === true"
							:disabled="likeBusy"
							class="like"
							@click="toggleLike"
						>
							<i
								class="ph-heart"
								:class="{
									'ph-bold': page.isLiked !== true,
									'ph-fill': page.isLiked === true,
								}"
							></i>
							<span v-if="page.likedCount > 0" class="count">
								{{ page.likedCount }}
							</span>
						</CfButton>
						<span v-else class="likedCount">
							<i class="ph-heart ph-bold"></i>
							{{ page.likedCount }}
						</span>

						<div v-if="isOwner" class="other">
							<CfButton @click="edit">
								<i class="ph-pencil ph-bold ph-lg"></i>
								{{ t("edit") }}
							</CfButton>
							<CfButton danger @click="remove">
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("delete") }}
							</CfButton>
						</div>

						<div class="user">
							<MkAvatar :user="page.user" class="avatar" />
							<div class="name">
								<MkA :to="userPage(page.user)" class="nameText">
									<MkUserName :user="page.user" />
								</MkA>
								<MkAcct :user="page.user" />
							</div>
						</div>
					</div>

					<div class="meta">
						<i class="ph-clock ph-bold"></i>
						{{ t("createdAt") }}
						<MkTime :time="page.createdAt" mode="detail" />
						<span v-if="page.updatedAt !== page.createdAt" class="updated">
							· {{ t("updatedAt") }}
							<MkTime :time="page.updatedAt" mode="detail" />
						</span>
					</div>
				</div>

				<h2 class="sectionTitle">
					<i class="ph-clock ph-bold ph-lg"></i>
					{{ t("recentPosts") }}
				</h2>
				<CfPagination :pagination="otherPagination">
					<template #empty>
						<CfInfo>{{ t("nothing") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<CfInfo v-if="otherPages(items).length === 0">
							{{ t("nothing") }}
						</CfInfo>
						<div v-else class="list">
							<MkPagePreview
								v-for="entry in otherPages(items)"
								:key="entry.id"
								:page="entry"
								class="item"
							/>
						</div>
					</template>
				</CfPagination>
			</div>
			<CfLoading v-else-if="loadError === null" />
			<CfInfo v-else warn class="_formBlock">{{ loadError }}</CfInfo>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's page viewer: eye-catching banner, title and author, like/unlike,
// the owner's pin/edit/delete controls, and the block document. The blocks go
// through PageBlock, which renders only the four static types (text, section,
// image, note) and shows a placeholder for dynamic ones, matching the pinned
// upstream client (c05e844). Visibility is always public, so there is no
// visibility chip or "private page" branch.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { Page, PageBlock } from "@/api/types";
import { me, setMe } from "@/account";
import { userPage } from "@/filters/user";
import MkPagePreview from "@/components/page/MkPagePreview.vue";
import PageBlockView from "@/components/page/PageBlock.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import type { Paging } from "@/components/ui";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { router } from "@/router";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPagePage" });

const route = useRoute();

const username = computed<string>(() => {
	const value = route.params.username;
	return typeof value === "string" ? value : "";
});

const pageName = computed<string>(() => {
	const value = route.params.pageName;
	return typeof value === "string" ? value : "";
});

const page = ref<Page | null>(null);
const loadError = ref<string | null>(null);
const likeBusy = ref(false);

const signedIn = computed(() => me.value != null);
const isOwner = computed(
	() => page.value != null && me.value != null && me.value.id === page.value.userId,
);
const canLike = computed(() => signedIn.value && !isOwner.value);
const isPinned = computed(
	() => page.value != null && me.value?.pinnedPageId === page.value.id,
);

const bannerImage = computed<string>(() => {
	const image = page.value?.eyeCatchingImage;
	if (image == null) return "linear-gradient(to bottom right, #31748f, #9ccfd8)";
	return `url(${resolveMediaUrl(image.url)})`;
});

const sourcePath = computed<string>(() => {
	const current = page.value;
	if (current == null) return "";
	return `/@${current.user.username}/pages/${current.name}/view-source`;
});

const otherPagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.usersPages,
	limit: 6,
	params: { userId: page.value?.userId ?? "" },
}));

function blockKey(block: PageBlock, index: number): string {
	return typeof block.id === "string" ? block.id : `${index}-${block.type}`;
}

/** CfPagination's slot is generic over `{ id: string }`. */
function otherPages(items: { id: string }[]): Page[] {
	const current = page.value;
	if (current == null) return [];
	return (items as Page[]).filter((entry) => entry.id !== current.id);
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function fetchPage(): Promise<void> {
	if (username.value === "" || pageName.value === "") return;
	loadError.value = null;
	page.value = null;
	try {
		page.value = await api(ENDPOINTS.pagesShow, {
			name: pageName.value,
			username: username.value,
		});
	} catch (cause) {
		loadError.value = errorText(cause);
	}
}

async function toggleLike(): Promise<void> {
	const current = page.value;
	if (current == null || likeBusy.value) return;
	const liked = current.isLiked === true;
	likeBusy.value = true;
	try {
		await api(liked ? ENDPOINTS.pagesUnlike : ENDPOINTS.pagesLike, {
			pageId: current.id,
		});
		page.value = {
			...current,
			isLiked: !liked,
			likedCount: Math.max(0, current.likedCount + (liked ? -1 : 1)),
		};
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		likeBusy.value = false;
	}
}

async function pin(pinned: boolean): Promise<void> {
	const current = page.value;
	if (current == null) return;
	try {
		const updated = await api(ENDPOINTS.iUpdate, {
			pinnedPageId: pinned ? current.id : null,
		});
		setMe(updated);
		os.success();
	} catch (cause) {
		os.toast(errorText(cause), "error");
	}
}

function copyUrl(): void {
	void copyToClipboard(window.location.href).then((ok) => {
		if (ok) os.success();
	});
}

function edit(): void {
	const current = page.value;
	if (current == null) return;
	void router.push(`/pages/edit/${current.id}`);
}

async function remove(): Promise<void> {
	const current = page.value;
	if (current == null) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteConfirm"),
	});
	if (canceled) return;
	try {
		await api(ENDPOINTS.pagesDelete, { pageId: current.id });
		await router.replace("/pages");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	}
}

watch([username, pageName], () => void fetchPage(), { immediate: true });

definePageMetadata(
	computed(() =>
		page.value == null
			? null
			: {
					title: page.value.title || page.value.name,
					icon: "ph-file-text ph-bold",
					avatar: page.value.user,
				},
	),
);
</script>

<style lang="scss" scoped>
.cf-page {
	> .main {
		overflow: hidden;

		> .banner {
			block-size: 180px;
			background-position: center;
			background-size: cover;

			> .header {
				padding: 16px;

				> h1 {
					margin: 0;
					color: #fff;
					text-shadow: 0 0 8px var(--shadow);
				}
			}

			> .menu-actions {
				display: inline-flex;
				align-items: center;
				gap: 2px;
				margin-inline-start: 1rem;
				padding: 4px;
				border-radius: 24px;
				background: rgba(0, 0, 0, 0.2);
				backdrop-filter: var(--blur, blur(8px));

				> .menu {
					display: inline-flex;
					align-items: center;
					justify-content: center;
					inline-size: 31px;
					block-size: 31px;
					color: #fff;
					font-size: 16px;
					text-shadow: 0 0 8px var(--shadow);
				}
			}
		}

		> .content {
			padding-block: 16px;
			padding-inline: 16px;

			&.serif {
				font-family: serif;
			}

			&.center {
				text-align: center;
			}
		}

		> .actions {
			display: flex;
			align-items: center;
			gap: 8px;
			margin-inline: 16px;
			padding-block: 16px;
			border-block-start: solid 0.5px var(--divider);

			> .like {
				--accent: #eb6f92;
				color: #eb6f92;

				.count {
					margin-inline-start: 4px;
				}
			}

			> .likedCount {
				display: inline-flex;
				align-items: center;
				gap: 4px;
				color: var(--fgTransparentWeak);
			}

			> .other {
				display: flex;
				gap: 8px;
				margin-inline-start: auto;
			}

			> .user {
				display: flex;
				align-items: center;
				gap: 12px;
				margin-inline-start: auto;

				> .avatar {
					inline-size: 40px;
					block-size: 40px;
				}

				> .name {
					display: flex;
					flex-direction: column;
					min-inline-size: 0;
					font-size: 0.9em;

					> .nameText {
						font-weight: bold;
					}
				}
			}
		}

		> .meta {
			display: flex;
			align-items: center;
			flex-wrap: wrap;
			gap: 6px;
			margin-inline: 16px;
			padding-block-end: 16px;
			font-size: 0.85em;
			opacity: 0.75;
		}
	}

	> .sectionTitle {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-block: 24px 12px;
		font-size: 1em;
		color: var(--fgTransparentWeak);
	}

	> .list {
		> .item + .item {
			margin-block-start: 8px;
		}
	}
}
</style>
