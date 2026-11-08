<!-- Ported from firefish@b425f61:packages/client/src/pages/my-lists/list.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-my-list">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<template v-else-if="list != null">
					<!-- name / visibility / delete -->
					<section class="_formBlock _panel settings">
						<div class="sectionTitle">
							{{ t("settings") }}
						</div>
						<form class="renameRow" @submit.prevent="saveName">
							<CfInput
								v-model="editingName"
								:placeholder="t('enterListName')"
								@enter="saveName"
							/>
							<CfButton
								type="submit"
								primary
								:disabled="
									saving ||
									editingName.trim() === '' ||
									editingName.trim() === list.name
								"
							>
								{{ t("save") }}
							</CfButton>
						</form>
						<CfSwitch
							:model-value="list.isPublic"
							:disabled="saving"
							@update:model-value="setPublic"
						>
							{{ t("public") }}
						</CfSwitch>
						<div class="dangerRow">
							<CfButton danger :disabled="deleting" @click="deleteList">
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("delete") }}
							</CfButton>
						</div>
					</section>

					<!-- add member -->
					<section class="_formBlock _panel">
						<div class="sectionTitle">{{ t("addUser") }}</div>
						<form class="searchRow" @submit.prevent="search">
							<CfInput
								v-model="searchQuery"
								:placeholder="t('searchUser')"
								@enter="search"
							/>
							<CfButton
								type="submit"
								primary
								:disabled="searching || searchQuery.trim() === ''"
							>
								{{ t("search") }}
							</CfButton>
						</form>
						<CfInfo v-if="searchError !== null" warn class="searchError">
							{{ searchError }}
						</CfInfo>
						<div v-if="searchResults.length > 0" class="searchResults">
							<div
								v-for="user in searchResults"
								:key="user.id"
								class="userRow"
							>
								<MkAvatar
									class="avatar"
									:user="user"
									:disable-preview="true"
									:show-indicator="true"
								/>
								<div class="body">
									<MkUserName :user="user" class="name" />
									<span class="acct">@{{ user.username }}</span>
								</div>
								<CfButton
									mini
									:disabled="addingUserId !== null"
									@click="addUser(user)"
								>
									{{ t("add") }}
								</CfButton>
							</div>
						</div>
						<p v-else-if="searched && !searching" class="emptyHint">
							{{ t("nothing") }}
						</p>
					</section>

					<!-- members -->
					<section class="_formBlock">
						<div class="sectionTitle">
							{{ t("members") }} ({{ memberships.length }})
						</div>
						<div v-if="memberships.length === 0" class="_fullinfo">
							<i class="ph-users ph-bold ph-lg"></i>
							<div>{{ t("nothing") }}</div>
						</div>
						<div v-else class="members">
							<div
								v-for="membership in memberships"
								:key="membership.id"
								class="member _panel"
							>
								<MkAvatar
									class="avatar"
									:user="membership.user"
									:show-indicator="true"
								/>
								<div class="body">
									<MkUserName
										:user="membership.user"
										class="name"
									/>
									<span class="acct"
										>@{{ membership.user.username }}</span
									>
								</div>
								<CfSwitch
									class="withReplies"
									:model-value="membership.withReplies"
									:disabled="updatingMembershipId !== null"
									@update:model-value="
										(value: boolean) =>
											setWithReplies(membership, value)
									"
								>
									{{ t("withReplies") }}
								</CfSwitch>
								<button
									class="_buttonIcon remove"
									:aria-label="t('removeMember')"
									:title="t('removeMember')"
									@click="removeMember(membership)"
								>
									<i class="ph-x ph-bold ph-lg"></i>
								</button>
							</div>
						</div>
					</section>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's list editor: rename, add/remove members and delete. Cinderfish's
// `users/lists/create` takes only a name, so visibility is an update; the
// membership list comes from `users/lists/get-memberships` (which carries the
// per-member `withReplies` flag) instead of Firefish's `users/show` batch call.
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type {
	UserDetailed,
	UserLite,
	UserList,
	UserListMembership,
} from "@/api/types";
import MkAvatar from "@/components/global/MkAvatar.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkUserName from "@/components/global/MkUserName.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyListsList" });

const route = useRoute();
const router = useRouter();

const listId = computed(() => String(route.params.listId ?? ""));
const list = ref<UserList | null>(null);
const memberships = ref<UserListMembership[]>([]);
const loading = ref(true);
const loadError = ref<string | null>(null);
const editingName = ref("");
const saving = ref(false);
const deleting = ref(false);
const updatingMembershipId = ref<string | null>(null);

const searchQuery = ref("");
const searching = ref(false);
const searched = ref(false);
const searchError = ref<string | null>(null);
const searchResults = ref<UserDetailed[]>([]);
const addingUserId = ref<string | null>(null);

const headerTabs = computed(() => []);
const headerActions = computed(() => []);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function loadMembers(): Promise<void> {
	memberships.value = await api(ENDPOINTS.usersListsGetMemberships, {
		listId: listId.value,
		limit: 100,
	});
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		const detail = await api(ENDPOINTS.usersListsShow, {
			listId: listId.value,
		});
		list.value = detail;
		editingName.value = detail.name;
		await loadMembers();
	} catch (cause) {
		list.value = null;
		memberships.value = [];
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

watch(listId, load, { immediate: true });

async function saveName(): Promise<void> {
	if (list.value == null || saving.value) return;
	const name = editingName.value.trim();
	if (name === "" || name === list.value.name) return;
	saving.value = true;
	try {
		list.value = await api(ENDPOINTS.usersListsUpdate, {
			listId: list.value.id,
			name,
		});
		editingName.value = list.value.name;
		os.toast(t("saved"), "success");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function setPublic(value: boolean): Promise<void> {
	if (list.value == null || saving.value) return;
	saving.value = true;
	try {
		list.value = await api(ENDPOINTS.usersListsUpdate, {
			listId: list.value.id,
			isPublic: value,
		});
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function removeMember(membership: UserListMembership): Promise<void> {
	if (list.value == null) return;
	try {
		await os.apiWithDialog(ENDPOINTS.usersListsPull, {
			listId: list.value.id,
			userId: membership.userId,
		});
		memberships.value = memberships.value.filter(
			(candidate) => candidate.id !== membership.id,
		);
	} catch {
		// apiWithDialog already reported the server error.
	}
}

async function setWithReplies(
	membership: UserListMembership,
	value: boolean,
): Promise<void> {
	if (list.value == null) return;
	const previous = membership.withReplies;
	membership.withReplies = value;
	updatingMembershipId.value = membership.id;
	try {
		await os.apiWithDialog(ENDPOINTS.usersListsUpdateMembership, {
			listId: list.value.id,
			userId: membership.userId,
			withReplies: value,
		});
	} catch {
		membership.withReplies = previous;
	} finally {
		updatingMembershipId.value = null;
	}
}

async function search(): Promise<void> {
	const query = searchQuery.value.trim();
	if (query === "" || searching.value) return;
	searching.value = true;
	searched.value = true;
	searchError.value = null;
	try {
		const found = await api(ENDPOINTS.usersSearch, {
			query,
			origin: "combined",
			limit: 10,
		});
		const memberIds = new Set(
			memberships.value.map((membership) => membership.userId),
		);
		searchResults.value = found.filter((user) => !memberIds.has(user.id));
	} catch (cause) {
		searchResults.value = [];
		searchError.value = errorText(cause);
	} finally {
		searching.value = false;
	}
}

function alreadyMember(user: UserLite): boolean {
	return memberships.value.some(
		(membership) => membership.userId === user.id,
	);
}

async function addUser(user: UserLite): Promise<void> {
	if (list.value == null || alreadyMember(user)) return;
	addingUserId.value = user.id;
	try {
		await os.apiWithDialog(ENDPOINTS.usersListsPush, {
			listId: list.value.id,
			userId: user.id,
		});
		searchResults.value = searchResults.value.filter(
			(candidate) => candidate.id !== user.id,
		);
		await loadMembers();
	} catch {
		// apiWithDialog already reported the server error.
	} finally {
		addingUserId.value = null;
	}
}

async function deleteList(): Promise<void> {
	if (list.value == null || deleting.value) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("removeAreYouSure", { x: list.value.name }),
	});
	if (canceled) return;
	deleting.value = true;
	try {
		await os.apiWithDialog(ENDPOINTS.usersListsDelete, {
			listId: list.value.id,
		});
		void router.push("/my/lists");
	} catch {
		// apiWithDialog already reported the server error.
	} finally {
		deleting.value = false;
	}
}

definePageMetadata(
	computed(() => ({
		title: list.value?.name ?? t("manageLists"),
		icon: "ph-list-bullets ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-my-list {
	.sectionTitle {
		margin-block-end: 12px;
		font-weight: bold;
	}

	.settings {
		display: flex;
		flex-direction: column;
		gap: 12px;

		.renameRow,
		.searchRow {
			display: flex;
			align-items: flex-end;
			gap: 8px;

			:deep(.matxzzsk) {
				flex: 1;
				min-width: 0;
			}
		}

		.dangerRow {
			display: flex;
			justify-content: flex-end;
		}
	}

	.searchError {
		margin-block-start: 10px;
	}

	.searchResults {
		margin-block-start: 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.userRow,
	.member {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 12px;

		> .avatar {
			inline-size: 36px;
			block-size: 36px;
			flex: 0 0 auto;
		}

		> .body {
			flex: 1;
			min-width: 0;
		}

		.name {
			display: block;
			font-weight: bold;
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}

		.acct {
			display: block;
			font-size: 0.82em;
			color: var(--fgTransparentWeak);
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}
	}

	.members {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.member {
		.withReplies {
			flex: 0 0 auto;
			font-size: 0.82em;
		}

		.remove {
			flex: 0 0 auto;
			padding: 6px;
			color: var(--fgTransparentWeak);
			border-radius: 6px;

			&:hover {
				color: var(--fg);
				background: var(--listItemHoverBg);
			}
		}
	}

	.emptyHint {
		margin: 12px 0 0;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}
}
</style>
