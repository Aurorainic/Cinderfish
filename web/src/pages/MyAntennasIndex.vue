<!-- Ported from firefish@b425f61:packages/client/src/pages/my-antennas/index.vue (AGPL-3.0-only) -->
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
			<div class="_formRoot cf-my-antennas">
				<CfInfo card class="_formBlock">
					<p class="description">{{ t("antennasDesc") }}</p>
					<CfButton primary @click="create">
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("add") }}
					</CfButton>
				</CfInfo>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<div v-else-if="antennas.length === 0" class="_fullinfo">
					<i class="ph-flying-saucer ph-bold ph-lg"></i>
					<div>{{ t("antennasDesc") }}</div>
				</div>
				<template v-else>
					<!-- antennas/list returns the complete list, not a cursor
					     page, so the rows are rendered as a plain array. -->
					<div
						v-for="antenna in antennas"
						:key="antenna.id"
						class="antenna _panel"
					>
						<CfA
							class="main"
							:to="`/timeline/antenna/${antenna.id}`"
						>
							<i class="ph-flying-saucer ph-bold ph-lg"></i>
							<div class="body">
								<div class="name">{{ antenna.name }}</div>
								<div class="meta">
									<span>{{ sourceLabel(antenna) }}</span>
									<span v-if="keywordCount(antenna) > 0">
										· {{ keywordCount(antenna) }}
										{{ t("antennaKeywords") }}
									</span>
								</div>
							</div>
						</CfA>
						<CfA
							class="edit"
							:to="`/my/antennas/${antenna.id}`"
							:aria-label="t('settings')"
							:title="t('settings')"
						>
							<i class="ph-pencil-simple ph-bold ph-lg"></i>
						</CfA>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish lists the caller's antennas with MkPagination and a 15-second
// refresh timer. `antennas/list` has no pagination and `hasUnreadNote` is
// always false, so the page fetches once and renders a plain array without the
// unread dot. Clicking a row opens its timeline; the pencil opens the editor.
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { Antenna } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyAntennasIndex" });

const router = useRouter();

const antennas = ref<Antenna[]>([]);
const loading = ref(true);
const loadError = ref<string | null>(null);

const headerTabs = computed(() => []);

const headerActions = computed(() => [
	{
		icon: "ph-plus ph-bold ph-lg",
		text: t("add"),
		handler: () => create(),
	},
]);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

function create(): void {
	void router.push("/my/antennas/create");
}

function keywordCount(antenna: Antenna): number {
	return antenna.keywords.reduce((sum, group) => sum + group.length, 0);
}

function sourceLabel(antenna: Antenna): string {
	switch (antenna.src) {
		case "home":
			return t("_antennaSources.homeTimeline");
		case "users":
			return t("_antennaSources.users");
		case "users_blacklist":
			return `${t("_antennaSources.users")} (excluded)`;
		case "list":
			return t("_antennaSources.userList");
		case "all":
		default:
			return t("_antennaSources.all");
	}
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		antennas.value = await api(ENDPOINTS.antennasList, {});
	} catch (cause) {
		antennas.value = [];
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

onMounted(load);

definePageMetadata(
	computed(() => ({
		title: t("manageAntennas"),
		icon: "ph-flying-saucer ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-my-antennas {
	.description {
		margin: 0 0 12px;
		white-space: pre-line;
	}

	.antenna {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-block-end: 8px;
		overflow: hidden;

		.main {
			display: flex;
			align-items: center;
			gap: 12px;
			flex: 1;
			min-width: 0;
			padding: 14px 16px;

			> i {
				flex: 0 0 auto;
				color: var(--fgTransparentWeak);
			}

			.body {
				min-width: 0;
			}

			.name {
				font-weight: bold;
				overflow-wrap: anywhere;
			}

			.meta {
				display: flex;
				flex-wrap: wrap;
				gap: 4px;
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}
		}

		.edit {
			display: flex;
			align-items: center;
			justify-content: center;
			flex: 0 0 auto;
			padding: 10px 14px;
			color: var(--fgTransparentWeak);
			border-radius: 6px;

			&:hover {
				color: var(--fg);
				background: var(--listItemHoverBg);
			}
		}
	}
}
</style>
