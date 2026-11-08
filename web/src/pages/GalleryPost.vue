<!-- Ported from firefish@b425f61:packages/client/src/pages/gallery/post.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="[]"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="1000" :margin-min="16" :margin-max="32">
			<div v-if="post" class="cf-gallery-post">
				<div class="files">
					<div
						v-for="file in post.files"
						:key="file.id"
						class="file"
					>
						<a
							v-if="!isHidden(file)"
							:href="resolveMediaUrl(file.url)"
							target="_blank"
							rel="noopener noreferrer"
						>
							<MkImgWithBlurhash
								:src="resolveMediaUrl(file.url)"
								:hash="file.blurhash"
								:alt="file.name"
								:cover="false"
							/>
						</a>
						<button
							v-else
							class="reveal _button"
							@click="revealed = true"
						>
							<MkImgWithBlurhash
								class="blurred"
								:src="resolveMediaUrl(file.url)"
								:hash="file.blurhash"
								:alt="file.name"
								:cover="false"
							/>
							<span class="overlay">
								<i class="ph-warning ph-bold ph-lg"></i>
								<b>{{ t("sensitive") }}</b>
								<span>{{ t("clickToShow") }}</span>
							</span>
						</button>
					</div>
				</div>

				<div class="body _panel">
					<h1 class="title">{{ post.title }}</h1>
					<div
						v-if="post.description != null && post.description !== ''"
						class="description"
					>
						<Mfm :text="post.description" :is-note="false" />
					</div>
					<div class="info">
						<i class="ph-clock ph-bold"></i>
						<MkTime :time="post.createdAt" mode="detail" />
						<span v-if="post.updatedAt != null" class="updated">
							· {{ t("updatedAt") }}
							<MkTime :time="post.updatedAt" mode="detail" />
						</span>
					</div>

					<div class="actions">
						<CfButton
							v-if="canLike"
							:primary="post.isLiked === true"
							class="like"
							@click="toggleLike"
						>
							<i
								class="ph-heart"
								:class="{
									'ph-bold': post.isLiked !== true,
									'ph-fill': post.isLiked === true,
								}"
							></i>
							{{
								post.isLiked
									? t("_gallery.unlike")
									: t("_gallery.like")
							}}
							<span v-if="post.likedCount > 0" class="count">
								{{ post.likedCount }}
							</span>
						</CfButton>
						<span v-else class="likedCount">
							<i class="ph-heart ph-bold"></i>
							{{ post.likedCount }}
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
					</div>

					<div class="user">
						<MkAvatar :user="post.user" class="avatar" />
						<div class="name">
							<MkA :to="userPage(post.user)" class="nameText">
								<MkUserName :user="post.user" />
							</MkA>
							<MkAcct :user="post.user" />
						</div>
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
						<CfInfo v-if="otherPosts(items).length === 0">
							{{ t("nothing") }}
						</CfInfo>
						<div v-else class="grid">
							<MkGalleryPostPreview
								v-for="item in otherPosts(items)"
								:key="item.id"
								:post="item"
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
// Firefish's gallery post page: every file full-size, the MFM description, the
// like button, the author card and a strip of the author's other posts.
//
// Firefish renders the files as bare <img> tags. The port reuses
// MkImgWithBlurhash and adds the note-image treatment for sensitive content:
// a blurred thumbnail stays behind a warning until the reader clicks it, then
// the full-size file is linked. `gallery/posts/show` packs `isLiked` only for a
// credential, and liking one's own post answers YOUR_POST, so the like button
// is hidden for guests and for the author.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile, GalleryPost } from "@/api/types";
import { me } from "@/account";
import { userPage } from "@/filters/user";
import MkGalleryPostPreview from "@/components/MkGalleryPostPreview.vue";
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
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageGalleryPost" });

const route = useRoute();

const postId = computed<string>(() => {
	const value = route.params.postId;
	return typeof value === "string" ? value : "";
});

const post = ref<GalleryPost | null>(null);
const loadError = ref<string | null>(null);
const reactionBusy = ref(false);
const revealed = ref(false);

const signedIn = computed(() => me.value != null);
const isOwner = computed(
	() =>
		post.value != null &&
		me.value != null &&
		me.value.id === post.value.userId,
);
const canLike = computed(() => signedIn.value && !isOwner.value);

const otherPagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.usersGalleryPosts,
	limit: 6,
	params: { userId: post.value?.userId ?? "" },
}));

const headerActions = computed(() => []);

function isHidden(file: DriveFile): boolean {
	const current = post.value;
	if (current == null || revealed.value) return false;
	return current.isSensitive || file.isSensitive;
}

/** CfPagination's slot is generic over `{ id: string }`. */
function otherPosts(items: { id: string }[]): GalleryPost[] {
	const current = post.value;
	if (current == null) return [];
	return (items as GalleryPost[]).filter((item) => item.id !== current.id);
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function fetchPost(): Promise<void> {
	if (postId.value === "") return;
	loadError.value = null;
	post.value = null;
	revealed.value = false;
	try {
		post.value = await api(ENDPOINTS.galleryPostsShow, {
			postId: postId.value,
		});
	} catch (cause) {
		loadError.value = errorText(cause);
	}
}

async function toggleLike(): Promise<void> {
	const current = post.value;
	if (current == null || reactionBusy.value) return;
	const liked = current.isLiked === true;
	reactionBusy.value = true;
	try {
		await api(
			liked ? ENDPOINTS.galleryPostsUnlike : ENDPOINTS.galleryPostsLike,
			{ postId: current.id },
		);
		post.value = {
			...current,
			isLiked: !liked,
			likedCount: Math.max(0, current.likedCount + (liked ? -1 : 1)),
		};
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		reactionBusy.value = false;
	}
}

function edit(): void {
	const current = post.value;
	if (current == null) return;
	void router.push(`/gallery/${current.id}/edit`);
}

async function remove(): Promise<void> {
	const current = post.value;
	if (current == null) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteConfirm"),
	});
	if (canceled) return;
	try {
		await api(ENDPOINTS.galleryPostsDelete, { postId: current.id });
		await router.replace("/gallery");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	}
}

watch(postId, () => void fetchPost(), { immediate: true });

definePageMetadata(
	computed(() =>
		post.value == null
			? null
			: {
					title: post.value.title,
					icon: "ph-image-square ph-bold",
				},
	),
);
</script>

<style lang="scss" scoped>
.cf-gallery-post {
	> .files {
		margin-block-end: var(--margin);

		> .file {
			& + .file {
				margin-block-start: 16px;
			}

			> a {
				display: block;

				img {
					display: block;
					max-inline-size: 100%;
					max-block-size: 500px;
					margin-inline: auto;
					border-radius: var(--radius);
				}
			}

			> .reveal {
				position: relative;
				display: block;
				inline-size: 100%;
				padding: 0;
				border: 0;
				background: none;
				cursor: pointer;
				border-radius: var(--radius);
				overflow: hidden;

				> .blurred {
					display: block;
					inline-size: 100%;
					max-block-size: 500px;
					object-fit: cover;
					filter: blur(24px);
					transform: scale(1.05);
				}

				> .overlay {
					position: absolute;
					inset: 0;
					display: flex;
					flex-direction: column;
					align-items: center;
					justify-content: center;
					gap: 6px;
					padding: 24px;
					font-size: 0.9em;
					color: #fff;
					text-shadow: 0 0 8px rgba(0, 0, 0, 0.6);
					background: rgba(0, 0, 0, 0.25);
				}
			}
		}
	}

	> .body {
		padding: 32px;

		> .title {
			margin: 0 0 16px;
			font-size: 1.4em;
			font-weight: bold;
		}

		> .description {
			overflow-wrap: anywhere;
		}

		> .info {
			display: flex;
			align-items: center;
			flex-wrap: wrap;
			gap: 6px;
			margin-block-start: 16px;
			font-size: 0.9em;
			color: var(--fgTransparentWeak);

			> .updated {
				display: inline-flex;
				align-items: center;
				gap: 4px;
			}
		}

		> .actions {
			display: flex;
			align-items: center;
			gap: 8px;
			margin-block-start: 16px;
			padding-block-start: 16px;
			border-block-start: solid 0.5px var(--divider);

			> .like .count {
				margin-inline-start: 4px;
				opacity: 0.8;
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
		}

		> .user {
			display: flex;
			align-items: center;
			gap: 12px;
			margin-block-start: 16px;
			padding-block-start: 16px;
			border-block-start: solid 0.5px var(--divider);

			> .avatar {
				inline-size: 52px;
				block-size: 52px;
				flex: 0 0 auto;
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

	> .sectionTitle {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-block: 24px 12px;
		font-size: 1em;
		color: var(--fgTransparentWeak);
	}

	> .grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 12px;
	}
}
</style>
