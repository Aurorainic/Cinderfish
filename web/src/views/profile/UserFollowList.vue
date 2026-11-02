// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/follow-list.vue
// (the list itself) and pages/user/{following,followers}.vue (the page shells,
// merged into one component selected by the `mode` prop).
//
// Adaptations versus the original:
// - The router hands in `mode`; the handle comes from the route the same way
//   ProfileView reads it (local part only, remote accounts are not browsable).
// - Firefish renders MkUserInfo cards in a responsive grid. The rows here
//   keep the account header PageHeader and list avatar, name, @handle and the
//   ported FollowButton, which is what the first release needs.
// - users/followers and users/following refuse a non-public list with 403
//   FORBIDDEN. MkPagination swallows the error object, so the page probes the
//   first row itself and turns that refusal into an MkInfo instead of a
//   retryable generic error; the probe also absorbs the empty-list case.
// - Each Following row embeds UserLite (Misskey's schema), so FollowButton is
//   fed a UserDetailed view with isFollowing=false; see asDetailedUser.
<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api, ApiError } from "../../api/client";
import type { Following, UserDetailed, UserLite } from "../../api/types";
import MkA from "../../components/ui/MkA.vue";
import MkAvatar from "../../components/ui/MkAvatar.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkPagination from "../../components/ui/MkPagination.vue";
import MkUserName from "../../components/ui/MkUserName.vue";
import PageHeader from "../../components/ui/PageHeader.vue";
import { usePageTitle } from "../../composables/use-page-title";
import { i18n } from "../../i18n";
import FollowButton from "./FollowButton.vue";

const props = defineProps<{
	mode: "following" | "followers";
}>();

const PAGE_SIZE = 20;

const route = useRoute();

const routeUsername = computed(() => {
	const raw = route.params.handle;
	const value = Array.isArray(raw) ? (raw[0] ?? "") : String(raw ?? "");
	const at = value.indexOf("@");
	return at < 0 ? value : value.slice(0, at);
});

const user = ref<UserDetailed | null>(null);
const loading = ref(false);
const loadError = ref("");
const missing = ref(false);
const forbidden = ref(false);

const label = computed(() => (props.mode === "followers" ? i18n.ts.followers : i18n.ts.following));

const displayName = computed(
	() => user.value?.name || user.value?.username || routeUsername.value,
);

const acct = computed(() =>
	user.value?.host
		? `${user.value.username}@${user.value.host}`
		: (user.value?.username ?? routeUsername.value),
);

const forbiddenMessage = computed(() =>
	props.mode === "followers"
		? "该用户的关注者列表不公开。"
		: "该用户的关注列表不公开。",
);

const emptyMessage = computed(() =>
	props.mode === "followers" ? "没有关注者。" : "没有正在关注的用户。",
);

usePageTitle(() => (displayName.value ? `${displayName.value} (@${routeUsername.value})` : ""));

function errorMessage(err: unknown, fallback: string): string {
	if (err instanceof ApiError) return err.message || fallback;
	if (err instanceof Error) return err.message || fallback;
	return fallback;
}

async function load() {
	user.value = null;
	loadError.value = "";
	missing.value = false;
	forbidden.value = false;

	if (!routeUsername.value) {
		missing.value = true;
		return;
	}

	loading.value = true;
	try {
		const fresh = await api.usersShow({ username: routeUsername.value });
		user.value = fresh;

		// Probe the list before handing it to MkPagination, which reports every
		// failure as a retryable MkError: a privacy refusal must read as a
		// plain statement, not as a broken page.
		try {
			if (props.mode === "followers") {
				await api.usersFollowers({ userId: fresh.id, limit: 1 });
			} else {
				await api.usersFollowing({ userId: fresh.id, limit: 1 });
			}
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) forbidden.value = true;
			else if (err instanceof ApiError && err.status === 404) missing.value = true;
			else loadError.value = errorMessage(err, "加载列表失败");
		}
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) missing.value = true;
		else loadError.value = errorMessage(err, "加载用户失败");
	} finally {
		loading.value = false;
	}
}

const pagination = computed(() => ({
	endpoint: (p: Record<string, unknown>): Promise<Following[]> => {
		const userId = user.value?.id;
		if (!userId) return Promise.resolve([]);
		const params = {
			userId,
			limit: Number(p.limit) || PAGE_SIZE,
			sinceId: typeof p.sinceId === "string" ? p.sinceId : undefined,
			untilId: typeof p.untilId === "string" ? p.untilId : undefined,
		};
		return props.mode === "followers"
			? api.usersFollowers(params)
			: api.usersFollowing(params);
	},
	limit: PAGE_SIZE,
}));

// Firefish reads the far end of each Following row: followers pages carry
// x.follower, following pages carry x.followee.
function peopleIn(items: Following[]): UserLite[] {
	const out: UserLite[] = [];
	for (const item of items) {
		const person = props.mode === "followers" ? item.follower : item.followee;
		if (person) out.push(person);
	}
	return out;
}

function acctOf(person: UserLite): string {
	return person.host ? `${person.username}@${person.host}` : person.username;
}

// The embedded users are UserLite, so the viewer relation FollowButton reads
// is absent. It starts in the "not following" state; a duplicate follow is
// rejected by the server (ALREADY_FOLLOWING) and surfaced in the usual alert.
function asDetailedUser(person: UserLite): UserDetailed {
	return { ...person, isFollowing: false } as unknown as UserDetailed;
}

watch([routeUsername, () => props.mode], () => void load(), { immediate: true });
</script>

<template>
	<div class="follow-page">
		<PageHeader
			:title="displayName"
			:subtitle="`@${acct} · ${label}`"
			:avatar="user"
			back
			:back-to="`/@${routeUsername}`"
		/>

		<div v-if="loading" class="skeleton"></div>

		<div v-else-if="missing" class="placeholder">
			<h2>没有找到这个账户</h2>
			<p>账户可能已经被删除或从未存在过；远程账户暂不支持浏览。</p>
		</div>

		<MkInfo v-else-if="loadError" warn class="notice">
			<span>{{ loadError }}</span>
			<button type="button" class="retry" @click="load">重试</button>
		</MkInfo>

		<template v-else-if="user">
			<MkInfo v-if="forbidden" class="notice">{{ forbiddenMessage }}</MkInfo>

			<div v-else class="panel">
				<MkPagination :key="`${mode}-${user.id}`" :pagination="pagination">
					<template #empty>
						<p class="empty">{{ emptyMessage }}</p>
					</template>
					<template #default="{ items }">
						<div
							v-for="person in peopleIn(items as Following[])"
							:key="person.id"
							class="user-row"
						>
							<MkAvatar class="avatar" :user="person" />
							<div class="identity">
								<MkA class="name" :to="`/@${person.username}`">
									<MkUserName :user="person" :nowrap="true" />
								</MkA>
								<span class="acct">@{{ acctOf(person) }}</span>
							</div>
							<FollowButton class="follow" :user="asDetailedUser(person)" />
						</div>
					</template>
				</MkPagination>
			</div>
		</template>
	</div>
</template>

<style scoped>
.follow-page {
	min-width: 0;
}

.skeleton {
	block-size: 220px;
	margin-block-start: var(--margin);
	border-radius: var(--radius);
	background: var(--panelHighlight);
}

.placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	margin-block-start: var(--margin);
	padding: 32px 20px;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
	text-align: center;
}

.placeholder h2 {
	margin: 0;
	font-size: 1.1em;
	color: var(--fgHighlighted);
}

.placeholder p {
	margin: 0;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
}

.notice {
	display: flex;
	align-items: center;
	gap: 10px;
	margin-block-start: var(--margin);
}

.notice .retry {
	padding: 4px 10px;
	border: 1px solid currentcolor;
	border-radius: 6px;
	background: transparent;
	color: inherit;
	font: inherit;
	font-size: 0.85em;
	cursor: pointer;
}

.panel {
	margin-block-start: var(--margin);
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
	overflow: hidden;
}

.user-row {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 12px 16px;
	min-width: 0;
}

.user-row + .user-row {
	border-block-start: 1px solid var(--divider);
}

.avatar {
	flex: none;
	--avatar-size: 44px;
}

.identity {
	display: flex;
	flex-direction: column;
	gap: 2px;
	flex: 1 1 auto;
	min-width: 0;
}

.name {
	overflow: hidden;
	color: var(--fgHighlighted);
	font-size: 0.95em;
	font-weight: bold;
	line-height: 1.3;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.acct {
	overflow: hidden;
	color: var(--fgTransparentWeak);
	font-size: 0.8em;
	line-height: 1.3;
	overflow-wrap: anywhere;
}

.follow {
	flex: none;
}

.empty {
	margin: 0;
	padding: 32px;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
	text-align: center;
}

@media (max-width: 500px) {
	.user-row {
		padding-inline: 12px;
	}
}
</style>
