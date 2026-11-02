// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/explore.vue and
// pages/explore.users.vue
//
// Adaptations:
// - Firefish's users/featured tab (notes/featured) and the hashtag clouds
//   (hashtags/list) are dropped: the backend has neither endpoint (capability
//   matrix marks 探索/hashtags as 需新增后端), so the page keeps the single
//   users tab.
// - The users/ endpoint with sort/origin parameters does not exist either;
//   the user lists are fed from admin/show-users, which returns local
//   accounts oldest-first with administrators at the front. That is a
//   degraded substitute for Firefish's popular/recently-updated/
//   recently-registered ranking (TODO when a sortable users endpoint lands).
// - The origin (local/remote) MkTab is dropped: no remote directory exists.
// - The swiper is replaced by remount-on-tab like TimelineView.
<template>
	<PageHeader v-model:tab="tab" :title="i18n.ts.explore" :tabs="tabs" />
	<div class="explore-body">
		<template v-if="isSignedIn(me)">
			<MkFolder class="gap" persist-key="explore-pinned-users">
				<template #header>精选用户</template>
				<UserList :pagination="pinnedUsers" />
			</MkFolder>
			<MkFolder class="gap" persist-key="explore-recently-updated-users">
				<template #header>最近活跃</template>
				<UserList :pagination="recentlyUpdatedUsers" />
			</MkFolder>
			<MkFolder class="gap" persist-key="explore-recently-registered-users">
				<template #header>新加入</template>
				<UserList :pagination="recentlyRegisteredUsers" />
			</MkFolder>
		</template>
		<MkFolder v-else class="gap" persist-key="explore-pinned-users">
			<template #header>精选用户</template>
			<UserList :pagination="pinnedUsers" />
		</MkFolder>
	</div>
</template>

<script lang="ts" setup>
import { api } from "../api/client";
import type { UserDetailed } from "../api/types";
import MkFolder from "../components/ui/MkFolder.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import { usePageTitle } from "../composables/use-page-title";
import { i18n } from "../i18n";
import { isSignedIn, me } from "../me";
import UserList from "./profile/UserList.vue";

usePageTitle(i18n.ts.explore);

// Local mirror of PageHeader's tab shape (the interface is not exported).
type Tab = { key: string; label: string; icon?: "users" | "feather" };

const tabs: Tab[] = [{ key: "users", label: i18n.ts.users, icon: "users" }];

const tab = defineModel<string>("tab", { default: "users" });

const LIST_LIMIT = 30;

// admin/show-users requires an administrator token and pages by offset; a
// signed-in non-admin gets an empty section rather than an error, and the
// page keeps rendering the folders. A public, sortable users directory is a
// backend TODO; until it exists the three sections share this one data
// source and the section headers make no ranking claim.
async function fetcherPage(limit: number, offset: number): Promise<UserDetailed[]> {
	return api.adminShowUsers({ limit, offset }).catch(() => [] as UserDetailed[]);
}

function listUsers(onlyAlive: boolean) {
	return async (p: Record<string, unknown>): Promise<UserDetailed[]> => {
		const res = await fetcherPage(Number(p.limit) || LIST_LIMIT, Math.max(0, Number(p.offset ?? 0)));
		return onlyAlive ? res.filter((u) => !u.isSuspended) : res;
	};
}

const pinnedUsers = {
	endpoint: listUsers(true),
	limit: 10,
	noPaging: true,
};
const recentlyUpdatedUsers = {
	endpoint: listUsers(false),
	limit: 10,
	noPaging: true,
};
const recentlyRegisteredUsers = {
	endpoint: listUsers(false),
	limit: 10,
	noPaging: true,
};
</script>

<style scoped>
.explore-body {
	padding-top: var(--space-16);
	padding-bottom: var(--space-24);
}

.gap {
	margin-bottom: var(--space-16);
}
</style>
