<!-- Ported from firefish@b425f61:packages/client/src/pages/follow-requests.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header><MkPageHeader /></template>
		<MkSpacer :content-max="800">
			<CfPagination ref="paginationComponent" :pagination="pagination">
				<template #empty>
					<div class="_fullinfo">
						<i class="ph-paper-plane-tilt ph-bold emptyIcon"></i>
						<div>还没有发出中的关注请求</div>
					</div>
				</template>
				<template #default="{ items }">
					<CfInfo class="info">
						这里只列出对方尚未处理、或对端尚未回执的关注请求。请求被接受或被拒绝后，这一行会直接消失，列表本身无法区分两种结果。
					</CfInfo>
					<div class="mk-follow-requests-sent">
						<div
							v-for="req in requests(items)"
							:key="req.id"
							class="user _panel"
						>
							<MkAvatar
								class="avatar"
								:user="req.followee"
								:show-indicator="true"
								disableLink
							/>
							<div class="body">
								<div class="name">
									<MkA
										v-user-preview="req.followee.id"
										class="name"
										:to="userPage(req.followee)"
										><MkUserName :user="req.followee"
									/></MkA>
									<p class="acct">@{{ acct(req.followee) }}</p>
								</div>
								<div class="actions">
									<button
										class="cancel _button"
										:aria-label="t('cancel')"
										:title="t('cancel')"
										@click="cancel(req)"
									>
										<i class="ph-x ph-bold ph-lg"></i>
										<span>{{ t("cancel") }}</span>
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
// Firefish has no "sent follow requests" page (its follow-requests.vue only
// covers incoming ones), so the layout is that page with the accept/reject pair
// replaced by a single cancel action. The backend endpoint is
// `following/requests/cancel`, which takes the *followee* id — the target of the
// original follow — not the request row id.
import { computed, ref } from "vue";
import type { FollowRequest } from "@/api/types";
import { CfInfo, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";
import { apiWithDialog } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

interface SentRequestsPaging {
	removeItem: (finder: (item: { id: string }) => boolean) => boolean;
}

const pagination: Paging = {
	endpoint: "following/requests/sent",
	limit: 10,
};

const paginationComponent = ref<SentRequestsPaging | null>(null);

function requests(items: readonly { id: string }[]): FollowRequest[] {
	return items as FollowRequest[];
}

async function cancel(request: FollowRequest): Promise<void> {
	try {
		// `cancel` takes the followee id — the target of the original follow —
		// not the request row id.
		await apiWithDialog("following/requests/cancel", {
			userId: request.followee.id,
		});
	} catch {
		// apiWithDialog already showed the error dialog (400 FOLLOW_REQUEST_NOT_FOUND
		// when the request vanished in the meantime); the row is left in place.
		return;
	}
	paginationComponent.value?.removeItem((item) => item.id === request.id);
}

definePageMetadata(
	computed(() => ({
		title: t("followRequests"),
		subtitle: "已发出",
		icon: "ph-paper-plane-tilt ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.emptyIcon {
	display: block;
	margin: 0 auto 8px auto;
	font-size: 32px;
	opacity: 0.5;
}

.info {
	margin: 10px 0;
}

.mk-follow-requests-sent {
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
				width: 100%;

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

				> .cancel {
					display: inline-flex;
					align-items: center;
					gap: 4px;
					padding: 8px 12px;
					border-radius: var(--radius);
					color: var(--fgTransparentWeak);

					&:hover {
						color: var(--accent);
						background: var(--buttonBg);
					}
				}
			}
		}
	}
}
</style>
