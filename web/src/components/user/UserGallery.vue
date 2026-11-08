<!-- Ported from firefish@b425f61:packages/client/src/pages/user/gallery.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-user-gallery">
		<CfPagination :pagination="pagination">
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
	</div>
</template>

<script setup lang="ts">
// Firefish's profile gallery tab: `users/gallery/posts` is a standard cursor
// list, so CfPagination drives the same 260px card grid MkGalleryPostPreview
// uses everywhere else.
import { computed } from "vue";
import { ENDPOINTS } from "@/api/endpoints";
import type { GalleryPost, UserDetailed } from "@/api/types";
import MkGalleryPostPreview from "@/components/MkGalleryPostPreview.vue";
import type { Paging } from "@/components/ui";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
}>();

const pagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.usersGalleryPosts,
	limit: 18,
	params: { userId: props.user.id },
}));

/** CfPagination's slot is generic over `{ id: string }`. */
function asPosts(items: { id: string }[]): GalleryPost[] {
	return items as GalleryPost[];
}
</script>

<style lang="scss" scoped>
.cf-user-gallery {
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 12px;
	}
}
</style>
