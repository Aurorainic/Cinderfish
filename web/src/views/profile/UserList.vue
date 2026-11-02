// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkUserList.vue
<script lang="ts" setup>
import type { UserDetailed } from "../../api/types";
import MkPagination from "../../components/ui/MkPagination.vue";
import UserInfo from "./UserInfo.vue";

defineProps<{
	// Firefish passes a Paging<entities.UserDetailed>; the ported MkPagination
	// takes the same shape with a fetcher function instead of an endpoint name.
	pagination: {
		endpoint: string | ((params: Record<string, unknown>) => Promise<UserDetailed[]>);
		limit: number;
		params?: Record<string, unknown>;
		noPaging?: boolean;
	};
}>();
</script>

<template>
	<MkPagination :pagination="pagination">
		<template #empty>
			<div class="fullinfo">
				<div class="ghost">🫥</div>
				<div>没有用户</div>
			</div>
		</template>

		<template #default="{ items }">
			<div class="users">
				<UserInfo
					v-for="user in items as UserDetailed[]"
					:key="user.id"
					class="user"
					:user="user"
				/>
			</div>
		</template>
	</MkPagination>
</template>

<style scoped>
.users {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
	gap: 16px;
}

.fullinfo {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 32px;
	color: var(--text-weak);
}

.ghost {
	font-size: 48px;
	opacity: 0.6;
}
</style>
