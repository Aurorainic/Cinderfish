<!-- Ported from firefish@b425f61:packages/client/src/pages/user/pages.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-user-pages">
		<CfPagination :pagination="pagination">
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
	</div>
</template>

<script setup lang="ts">
// Firefish's profile pages tab: `users/pages` returns the account's public
// pages as a standard cursor list, so CfPagination drives the same
// MkPagePreview rows the /pages index uses.
import { computed } from "vue";
import { ENDPOINTS } from "@/api/endpoints";
import type { Page, UserDetailed } from "@/api/types";
import MkPagePreview from "@/components/page/MkPagePreview.vue";
import type { Paging } from "@/components/ui";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
}>();

const pagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.usersPages,
	limit: 10,
	params: { userId: props.user.id },
}));

/** CfPagination's slot is generic over `{ id: string }`. */
function asPages(items: { id: string }[]): Page[] {
	return items as Page[];
}
</script>

<style lang="scss" scoped>
.cf-user-pages {
	.list {
		> .item + .item {
			margin-block-start: 8px;
		}
	}
}
</style>
