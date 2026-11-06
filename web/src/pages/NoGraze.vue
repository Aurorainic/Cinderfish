<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<div class="public-local">
				<CfInfo v-if="me == null" class="_gap" icon="users">
					这是本站的本地时间线，访客无需登录即可阅读。
					<CfA to="/">登录</CfA>
					后可以发帖、回应与关注。
				</CfInfo>

				<MkTimeline src="local" />
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// /public/local. Firefish has no such route; the page exists so a visitor (or
// a crawler) has a stable local-timeline URL. `POST /api/notes/local-timeline`
// is public — internal/httpapi/timelines.go calls serveTimeline with
// requireCredential=false — so the timeline is rendered for everyone instead of
// behind a login wall. Empty and error states come from MkTimeline/MkNotes
// (CfPagination), which the page reuses rather than reimplementing.
import { computed } from "vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkTimeline from "@/components/MkTimeline.vue";
import { me } from "@/account";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageNoGraze" });

const headerActions = computed(() => []);

definePageMetadata({
	title: "本地时间线",
	icon: "ph-users ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.public-local {
	padding-block: 8px;

	> ._gap {
		margin-block-end: 12px;
	}
}
</style>
