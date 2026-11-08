<!-- Ported from firefish@b425f61:packages/client/src/pages/user-list-timeline.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="800" :margin-min="6">
			<div class="cf-user-list-timeline">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_gap">
					{{ loadError }}
				</CfInfo>
				<template v-else-if="list != null">
					<div class="listHeader _block">
						<div class="name">
							<i class="ph-list-bullets ph-bold"></i>
							<span>{{ list.name }}</span>
						</div>
						<div class="members">{{ memberSummary }}</div>
					</div>
					<div class="tl _block">
						<MkTimeline
							:key="listId"
							src="list"
							:list-id="listId"
							:sound="true"
						/>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's user-list timeline: load the list header, then render MkTimeline
// with src="list". Firefish also offers a date jump (timetravel) header action;
// MkTimeline has no timetravel support here, so it is dropped. The settings
// action opens the same list editor Firefish links to (`/my/lists/:id`).
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { UserList } from "@/api/types";
import MkTimeline from "@/components/MkTimeline.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageUserListTimeline" });

const route = useRoute();
const router = useRouter();

const listId = computed(() => String(route.params.listId ?? ""));
const list = ref<UserList | null>(null);
const loading = ref(true);
const loadError = ref<string | null>(null);

const headerTabs = computed(() => []);

const memberSummary = computed(() => {
	const count = list.value?.userIds?.length ?? 0;
	return `${count} ${t("members")}`;
});

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		list.value = await api(ENDPOINTS.usersListsShow, {
			listId: listId.value,
		});
	} catch (cause) {
		list.value = null;
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

watch(listId, load, { immediate: true });

function openMenu(ev: MouseEvent): void {
	const id = list.value?.id ?? listId.value;
	const items: MenuItem[] = [
		{
			icon: "ph-sliders-horizontal ph-bold ph-lg",
			text: t("manageLists"),
			action: () => {
				void router.push(`/my/lists/${id}`);
			},
		},
	];
	void os.popupMenu(items, ev);
}

const headerActions = computed(() =>
	list.value == null
		? []
		: [
				{
					icon: "ph-gear-six ph-bold ph-lg",
					text: t("settings"),
					handler: openMenu,
				},
			],
);

definePageMetadata(
	computed(() => ({
		title: list.value?.name ?? t("timeline"),
		icon: "ph-list-bullets ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-user-list-timeline {
	.listHeader {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 16px;
		border-radius: var(--radius) var(--radius) 0 0;
		background: var(--panel);

		.name {
			display: flex;
			align-items: center;
			gap: 8px;
			min-width: 0;
			font-weight: bold;
			overflow-wrap: anywhere;

			> i {
				flex: 0 0 auto;
				color: var(--fgTransparentWeak);
			}
		}

		.members {
			flex: 0 0 auto;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}
	}

	.tl {
		background: none;
	}
}
</style>
