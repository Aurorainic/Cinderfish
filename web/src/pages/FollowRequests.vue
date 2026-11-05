<!-- Ported from firefish@b425f61:packages/client/src/pages/follow-requests.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header><MkPageHeader /></template>
		<MkSpacer :content-max="800">
			<CfPagination
				ref="paginationComponent"
				:pagination="pagination"
			>
				<template #empty>
					<div class="_fullinfo">
						<img
							src="/static-assets/badges/info.svg"
							aria-label="none"
							class="_ghost"
						/>
						<div>{{ t("noFollowRequests") }}</div>
					</div>
				</template>
				<template #default="{ items }">
					<CfInfo v-if="me != null && !me.isLocked" warn class="info">
						{{ t("silencedWarning") }}
					</CfInfo>
					<div class="mk-follow-requests">
						<div
							v-for="req in followRequests(items)"
							:key="req.id"
							class="user _panel"
						>
							<MkAvatar
								class="avatar"
								:user="req.follower"
								:show-indicator="true"
								disableLink
							/>
							<div class="body">
								<div class="name">
									<MkA
										v-user-preview="req.follower.id"
										class="name"
										:to="userPage(req.follower)"
										><MkUserName :user="req.follower"
									/></MkA>
									<p class="acct">@{{ acct(req.follower) }}</p>
								</div>
								<div class="actions">
									<button
										class="_button"
										:aria-label="t('accept')"
										@click="accept(req.follower)"
									>
										<i class="ph-check ph-bold ph-lg"></i>
									</button>
									<button
										class="_button"
										:aria-label="t('reject')"
										@click="reject(req.follower)"
									>
										<i class="ph-x ph-bold ph-lg"></i>
									</button>
								</div>
							</div>
						</div>
					</div>
				</template>
			</CfPagination>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { api } from "@/api/client";
import type { FollowRequest, UserLite } from "@/api/types";
import { me } from "@/account";
import { CfInfo, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

interface FollowRequestsPaging {
	reload: () => Promise<FollowRequest[]>;
}

const paginationComponent = ref<FollowRequestsPaging | null>(null);

// CfPagination is generic over `{ id: string }` when instantiated from a
// template, so the slot payload is re-typed here.
function followRequests(items: readonly { id: string }[]): FollowRequest[] {
	return items as FollowRequest[];
}

const pagination: Paging = {
	endpoint: "following/requests/list",
	limit: 10,
};

async function accept(user: UserLite): Promise<void> {
	await api("following/requests/accept", { userId: user.id });
	await paginationComponent.value?.reload();
}

async function reject(user: UserLite): Promise<void> {
	await api("following/requests/reject", { userId: user.id });
	await paginationComponent.value?.reload();
}

definePageMetadata(
	computed(() => ({
		title: t("followRequests"),
		icon: "ph-hand-waving ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.mk-follow-requests {
	> .user {
		display: flex;
		padding: 16px;
		margin: 10px 0 auto;

		> .avatar {
			display: block;
			flex-shrink: 0;
			margin: 0 12px 0 0;
			width: 42px;
			height: 42px;
			border-radius: 8px;
		}

		> .body {
			display: flex;
			width: calc(100% - 54px);
			position: relative;

			> .name {
				width: 45%;

				@media (max-width: 500px) {
					width: 100%;
				}

				> .name,
				> .acct {
					display: block;
					white-space: nowrap;
					text-overflow: ellipsis;
					overflow: hidden;
					margin: 0;
				}

				> .name {
					font-size: 16px;
					line-height: 24px;
				}

				> .acct {
					font-size: 15px;
					line-height: 16px;
					opacity: 0.7;
				}
			}

			> .actions {
				position: absolute;
				top: 0;
				bottom: 0;
				right: 0;
				margin: auto 0;

				> button {
					padding: 12px;
				}
			}
		}
	}
}
</style>
