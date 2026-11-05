<!-- Ported from firefish@b425f61:packages/client/src/pages/user/follow-list.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-following-or-followers">
		<CfLoading v-if="fetching && items.length === 0" />

		<CfInfo v-else-if="forbidden" warn icon="lock" class="privacy">
			<div class="privacyBody">
				<b>{{ t("_auth.denied") }}</b>
				<div>{{ t("ffVisibility") }}</div>
				<div v-if="visibilityLabel != null" class="visibility">
					{{ visibilityLabel }}
				</div>
			</div>
		</CfInfo>

		<div v-else-if="error != null && items.length === 0" class="error _fullinfo">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton primary class="retry" @click="reload">{{ t("retry") }}</CfButton>
		</div>

		<div v-else-if="users.length === 0" class="_fullinfo">
			<i class="ph-users ph-bold emptyIcon"></i>
			<div>{{ t("noUsers") }}</div>
		</div>

		<template v-else>
			<div class="users">
				<MkUserCardMini
					v-for="user in users"
					:key="user.id"
					class="user"
					:user="user"
				/>
			</div>

			<div v-if="hasMore" class="more">
				<CfButton v-if="!moreFetching" primary @click="loadMore">
					{{ t("loadMore") }}
				</CfButton>
				<CfLoading v-else mini />
			</div>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish renders `MkPagination` and maps each row to one side of the follow
// relation. Two Cinderfish facts change the shape of that translation
// (docs/research/api-contract/auth-users.md):
//   * a row carries only one embedded user (`followee` for users/following,
//     `follower` for users/followers), so the missing side is skipped instead
//     of dereferenced;
//   * a 403 FORBIDDEN means the target hid the list behind
//     `followingVisibility` / `followersVisibility`. `CfPagination` only
//     reports "an error happened", so this list drives `usePagination`
//     directly: that keeps the error object around and lets the 403 render as
//     an explanation instead of a failure.
// Degradation: Firefish's follow-list renders the full user card (MkUserInfo)
// per row. These endpoints pack each embedded user as `UserLite` only, so the
// card would have to invent a description and three zero counters — and its
// follow button reads `isFollowing`/`isLocked` from a UserDetailed. The row is
// therefore the mini card (avatar, name, handle, link to the profile).
// Degradation: no IntersectionObserver auto-load, the list loads page by page
// with an explicit button.
import { computed } from "vue";
import { isApiError } from "@/api/errors";
import type { Following, UserDetailed, UserLite } from "@/api/types";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkUserCardMini from "@/components/MkUserCardMini.vue";
import { usePagination } from "@/composables/use-pagination";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
	type: "following" | "followers";
}>();

const {
	items,
	fetching,
	moreFetching,
	hasMore,
	error,
	reload,
	loadMore,
} = usePagination<Following>({
	endpoint: props.type === "following" ? "users/following" : "users/followers",
	params: () => ({ userId: props.user.id }),
	limit: 20,
});

const forbidden = computed<boolean>(
	() => isApiError(error.value) && error.value.status === 403,
);

const users = computed<UserLite[]>(() => {
	const result: UserLite[] = [];
	for (const item of items.value) {
		const user = props.type === "following" ? item.followee : item.follower;
		if (user != null) result.push(user);
	}
	return result;
});

// The 403 body does not say which visibility caused it, but users/show already
// carries both fields, so the message names the one this list is governed by.
// Only the "followers" wording is used: the shipped dictionary translates
// `_ffVisibility.private` as "私信" (a direct message), which would misdescribe
// a hidden list, so the private case states nothing beyond the field name.
const visibilityLabel = computed<string | null>(() => {
	const visibility =
		props.type === "following"
			? props.user.followingVisibility
			: props.user.followersVisibility;
	return visibility === "followers" ? t("_ffVisibility.followers") : null;
});
</script>

<style lang="scss" scoped>
.mk-following-or-followers {
	> .users {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		grid-gap: var(--margin);
	}

	> .privacy {
		align-items: flex-start;

		> .privacyBody > .visibility {
			margin-top: 2px;
			font-weight: 700;
		}
	}

	> .error {
		color: var(--fgTransparentWeak);

		> .retry {
			margin-block-start: 12px;
		}
	}

	> .more {
		display: flex;
		justify-content: center;
		padding-block: var(--margin);
	}

	.emptyIcon {
		display: block;
		margin: 0 auto 8px auto;
		font-size: 32px;
		opacity: 0.5;
	}
}
</style>
