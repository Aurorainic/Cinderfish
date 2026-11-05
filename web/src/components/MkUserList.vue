<!-- Ported from firefish@b425f61:packages/client/src/components/MkUserList.vue (AGPL-3.0-only) -->
<template>
	<CfPagination :pagination="pagination">
		<template #empty>
			<div class="_fullinfo">
				<i class="ph-users ph-bold emptyIcon"></i>
				<div>{{ t("noUsers") }}</div>
			</div>
		</template>

		<template #default="{ items }">
			<div class="efvhhmdq">
				<MkUserInfo
					v-for="user in resolveUsers(items)"
					:key="user.id"
					class="user"
					:user="user"
				/>
			</div>
		</template>
	</CfPagination>
</template>

<script lang="ts" setup>
// Firefish's MkUserList pages over a bare user array. Cinderfish has both
// shapes: `users/followers|following` return Misskey's `Following` envelope
// (`{ id, follower | followee }`, see docs/research/api-contract/auth-users.md)
// while `admin/show-users` returns bare UserDetailed[], so each item is
// resolved to a user before rendering. That keeps this component usable by the
// follow pages and by admin lists without a second list component.
import type { UserLite } from "@/api/types";
import CfPagination from "@/components/ui/CfPagination.vue";
import type { Paging } from "@/components/ui";
import MkUserInfo from "@/components/MkUserInfo.vue";
import { t } from "@/i18n";

withDefaults(
	defineProps<{
		pagination: Paging;
		noGap?: boolean;
	}>(),
	{
		noGap: false,
	},
);

interface UserListEntry {
	id: string;
	username?: string;
	follower?: UserLite;
	followee?: UserLite;
}

function resolveUsers(items: { id: string }[]): UserLite[] {
	const users: UserLite[] = [];
	for (const raw of items) {
		const item = raw as UserListEntry;
		if (item.username != null) {
			users.push(raw as unknown as UserLite);
			continue;
		}
		const user = item.followee ?? item.follower;
		if (user != null) users.push(user);
	}
	return users;
}
</script>

<style lang="scss" scoped>
.efvhhmdq {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
	grid-gap: var(--margin);
}
</style>
