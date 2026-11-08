<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/abuses.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-abuses">
				<CfInfo warn>
					需要管理员权限，无法查看举报。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-abuses">
				<form class="filters" @submit.prevent="apply">
					<CfSelect v-model="state" class="field" @update:model-value="apply">
						<template #label>状态</template>
						<option value="unresolved">未处理</option>
						<option value="resolved">已处理</option>
						<option value="">全部</option>
					</CfSelect>
					<CfSelect
						v-model="reporterOrigin"
						class="field"
						@update:model-value="apply"
					>
						<template #label>举报者来源</template>
						<option value="combined">全部</option>
						<option value="local">本地</option>
						<option value="remote">远程</option>
					</CfSelect>
					<CfSelect
						v-model="targetOrigin"
						class="field"
						@update:model-value="apply"
					>
						<template #label>被举报者来源</template>
						<option value="combined">全部</option>
						<option value="local">本地</option>
						<option value="remote">远程</option>
					</CfSelect>
					<div class="filterActions">
						<CfButton type="submit" primary>刷新</CfButton>
					</div>
				</form>

				<CfLoading v-if="fetching && items.length === 0" />
				<MkError v-else-if="error !== null && items.length === 0" @retry="reload" />
				<CfInfo v-else-if="items.length === 0">
					没有符合筛选条件的举报。
				</CfInfo>
				<template v-else>
					<ul class="reports">
						<li
							v-for="item in items"
							:key="item.id"
							class="report"
							:class="{ resolved: item.resolved }"
						>
							<div class="head">
								<MkAvatar class="avatar" :user="item.reporter" />
								<div class="who">
									<MkA class="name" :to="userPage(item.reporter)">
										<MkUserName :user="item.reporter" :nowrap="true" />
									</MkA>
									<span class="verb">举报</span>
									<MkA class="name target" :to="userPage(item.targetUser)">
										<MkUserName :user="item.targetUser" :nowrap="true" />
									</MkA>
								</div>
								<span v-if="item.resolved" class="badge">已处理</span>
								<MkTime class="date" :time="item.createdAt" mode="absolute" />
							</div>
							<p class="comment">{{ item.comment }}</p>
							<p v-if="item.moderationNote !== ''" class="note">
								<span class="noteLabel">版务备注</span>
								{{ item.moderationNote }}
							</p>
							<div class="actions">
								<CfButton
									v-if="!item.resolved"
									mini
									:disabled="resolvingId === item.id"
									@click="resolve(item)"
								>
									{{ resolvingId === item.id ? "处理中…" : "标记已处理" }}
								</CfButton>
								<CfButton
									v-if="item.targetUser.host != null && !item.forwarded"
									mini
									:disabled="forwardingId === item.id"
									@click="forward(item)"
								>
									{{ forwardingId === item.id ? "转发中…" : "转发到对方实例" }}
								</CfButton>
								<CfButton
									mini
									:disabled="notingId === item.id"
									@click="editNote(item)"
								>
									{{ notingId === item.id ? "保存中…" : "版务备注" }}
								</CfButton>
								<span v-if="item.forwarded" class="badge forwarded">已转发</span>
								<CfA class="profile" :to="userPage(item.targetUser)">
									查看被举报者
								</CfA>
							</div>
						</li>
					</ul>

					<div v-if="hasMore" class="more">
						<CfButton v-if="!moreFetching" @click="loadMore">
							加载更多
						</CfButton>
						<CfLoading v-else mini />
					</div>
				</template>

				<CfInfo v-if="actionError !== null" warn class="actionError">
					{{ actionError }}
				</CfInfo>
				<CfInfo class="notice">
					对远端账号的举报可以「转发到对方实例」：会以管理员的账号签名投递一条
					ActivityPub <code>Flag</code>，并把该举报标记为已转发（重复点击不会重复发送）。
					本地账号的举报不能转发；自动处置没有实现。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's abuse list pages `admin/abuse-user-reports` with state / origin
// filters and offers resolve + forward. Cinderfish implements the list, the
// filters, resolve and forwarding a report about a remote account as a Flag;
// the local-target refusal comes from the server and is shown verbatim.
// Automatic moderation ("auto-handle") has no endpoint and is called out in
// the page instead of being drawn as a dead button.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import { inputParagraph } from "@/os";
import type { AbuseReport } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkAvatar from "@/components/global/MkAvatar.vue";
import MkError from "@/components/global/MkError.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkUserName from "@/components/global/MkUserName.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import { usePagination } from "@/composables/use-pagination";
import { t } from "@/i18n";
import { userPage } from "@/filters/user";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminAbuses" });

const admin = computed<boolean>(() => isAdmin());

const state = ref("unresolved");
const reporterOrigin = ref("combined");
const targetOrigin = ref("combined");
const resolvingId = ref<string | null>(null);
const forwardingId = ref<string | null>(null);
const notingId = ref<string | null>(null);
const actionError = ref<string | null>(null);

const { items, fetching, moreFetching, hasMore, error, reload, loadMore } =
	usePagination<AbuseReport>({
		endpoint: "admin/abuse-user-reports",
		limit: 20,
		immediate: false,
		params: () => ({
			state: state.value === "" ? null : state.value,
			reporterOrigin: reporterOrigin.value,
			targetUserOrigin: targetOrigin.value,
		}),
	});

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function apply(): Promise<void> {
	if (admin.value) await reload();
}

async function resolve(item: AbuseReport): Promise<void> {
	if (resolvingId.value !== null) return;
	resolvingId.value = item.id;
	actionError.value = null;
	try {
		await api("admin/resolve-abuse-user-report", { reportId: item.id });
		item.resolved = true;
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		resolvingId.value = null;
	}
}

async function editNote(item: AbuseReport): Promise<void> {
	if (notingId.value !== null) return;
	const { canceled, result } = await inputParagraph({
		title: "版务备注",
		text: "只有管理员能看到这条备注。留空并保存会清除它。",
		default: item.moderationNote,
	});
	if (canceled) return;
	const note = result ?? "";
	notingId.value = item.id;
	actionError.value = null;
	try {
		await api("admin/update-abuse-user-report", {
			reportId: item.id,
			moderationNote: note,
		});
		item.moderationNote = note;
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		notingId.value = null;
	}
}

async function forward(item: AbuseReport): Promise<void> {
	if (forwardingId.value !== null) return;
	forwardingId.value = item.id;
	actionError.value = null;
	try {
		await api("admin/forward-abuse-user-report", { reportId: item.id });
		item.forwarded = true;
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		forwardingId.value = null;
	}
}

onMounted(() => {
	if (admin.value) void reload();
});

definePageMetadata({
	title: t("abuseReports"),
	icon: "ph-warning-octagon ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.filters {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
	gap: 10px;
	align-items: end;
	margin-bottom: 16px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.field {
		margin: 0;
	}

	.filterActions {
		display: flex;
		gap: 8px;
	}
}

.reports {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.report {
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	&.resolved {
		opacity: 0.65;
	}

	.head {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
	}

	.avatar {
		width: 36px;
		height: 36px;
		flex: none;
	}

	.who {
		display: flex;
		align-items: center;
		gap: 6px;
		flex-wrap: wrap;
		min-width: 0;
	}

	.name {
		font-weight: 600;
	}

	.verb {
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}

	.badge {
		padding: 1px 6px;
		border-radius: 999px;
		background: var(--bg);
		font-size: 0.72em;
	}

	.date {
		margin-left: auto;
		font-size: 0.78em;
		color: var(--fgTransparentWeak);
	}

	.comment {
		margin: 8px 0 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.note {
		margin: 8px 0 0;
		padding: 6px 10px;
		border-radius: 6px;
		background: var(--bg);
		font-size: 0.85em;
		white-space: pre-wrap;
		overflow-wrap: anywhere;

		> .noteLabel {
			display: inline-block;
			margin-right: 6px;
			padding: 0 6px;
			border-radius: 999px;
			background: var(--panel);
			font-size: 0.85em;
			opacity: 0.8;
		}
	}

	.actions {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-top: 10px;
	}

	.profile {
		font-size: 0.85em;
	}
}

.more {
	display: flex;
	justify-content: center;
	padding: 12px 0;
}

.actionError {
	margin-top: 12px;
}

.notice {
	margin-top: 12px;
}
</style>
