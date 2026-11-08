<!-- Ported from firefish@b425f61:packages/client/src/pages/my-lists/index.vue (AGPL-3.0-only) -->
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
			<div class="_formRoot cf-my-lists">
				<div class="_formBlock">
					<CfButton primary @click="startCreate">
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("createList") }}
					</CfButton>
				</div>

				<div v-if="creating" class="_formBlock createForm _panel">
					<CfInput
						v-model="newName"
						:placeholder="t('enterListName')"
						@enter="create"
					>
						<template #label>{{ t("enterListName") }}</template>
					</CfInput>
					<CfSwitch v-model="newIsPublic" class="_formBlock">
						{{ t("public") }}
					</CfSwitch>
					<div class="actions">
						<CfButton
							primary
							:disabled="busy || newName.trim() === ''"
							@click="create"
						>
							{{ t("createList") }}
						</CfButton>
						<CfButton :disabled="busy" @click="cancelCreate">
							{{ t("cancel") }}
						</CfButton>
					</div>
				</div>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<div
					v-else-if="lists.length === 0"
					class="_fullinfo"
				>
					<i class="ph-list-bullets ph-bold ph-lg"></i>
					<div>{{ t("noLists") }}</div>
				</div>
				<template v-else>
					<CfA
						v-for="list in lists"
						:key="list.id"
						class="list _panel"
						:to="`/my/lists/${list.id}`"
					>
						<div class="body">
							<div class="name">{{ list.name }}</div>
							<div class="meta">
								<span
									v-if="list.isPublic"
									class="badge"
									>{{ t("public") }}</span
								>
								<span class="count"
									>{{ memberCount(list) }} {{ t("members") }}</span
								>
							</div>
						</div>
						<i class="ph-caret-right ph-bold"></i>
					</CfA>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish lists the caller's lists with MkPagination and creates through an
// input dialog. Cinderfish's list endpoint returns a bare array, so the page
// fetches once and refreshes in place after a create; the inline form is the
// port's substitute for `os.inputText` (which exists, but a form also exposes
// the isPublic flag without a second dialog).
import { computed, onMounted, ref } from "vue";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { UserList } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyListsIndex" });

const lists = ref<UserList[]>([]);
const loading = ref(true);
const loadError = ref<string | null>(null);
const creating = ref(false);
const newName = ref("");
const newIsPublic = ref(false);
const busy = ref(false);

const headerTabs = computed(() => []);

const headerActions = computed(() => [
	{
		icon: "ph-plus ph-bold ph-lg",
		text: t("createList"),
		handler: () => startCreate(),
	},
]);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

function memberCount(list: UserList): number {
	return list.userIds?.length ?? 0;
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		lists.value = await api(ENDPOINTS.usersListsList, {});
	} catch (cause) {
		lists.value = [];
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

function startCreate(): void {
	creating.value = true;
	newName.value = "";
	newIsPublic.value = false;
}

function cancelCreate(): void {
	creating.value = false;
	newName.value = "";
	newIsPublic.value = false;
}

async function create(): Promise<void> {
	const name = newName.value.trim();
	if (busy.value || name === "") return;
	busy.value = true;
	try {
		const list = await api(ENDPOINTS.usersListsCreate, { name });
		// `users/lists/create` only takes a name; the visibility flag is a
		// separate update (same split as Misskey's contract).
		const created =
			newIsPublic.value
				? await api(ENDPOINTS.usersListsUpdate, {
						listId: list.id,
						isPublic: true,
					})
				: list;
		lists.value = [created, ...lists.value];
		cancelCreate();
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		busy.value = false;
	}
}

onMounted(load);

definePageMetadata(
	computed(() => ({
		title: t("manageLists"),
		icon: "ph-list-bullets ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-my-lists {
	.createForm {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	> .list {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 16px;

		&:hover {
			border-color: var(--accent);
			text-decoration: none;
		}

		> .body {
			flex: 1;
			min-width: 0;
		}

		.name {
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
			font-weight: bold;
		}

		.meta {
			display: flex;
			align-items: center;
			gap: 8px;
			margin-block-start: 4px;
			font-size: 0.82em;
			color: var(--fgTransparentWeak);
		}

		.badge {
			padding: 1px 6px;
			border: solid 1px var(--divider);
			border-radius: 999px;
		}

		> i {
			flex: 0 0 auto;
			color: var(--fgTransparentWeak);
		}
	}
}
</style>
