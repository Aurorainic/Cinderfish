<!-- Ported from firefish@b425f61:packages/client/src/pages/antenna-timeline.vue (AGPL-3.0-only) -->
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
			<div class="cf-antenna-timeline">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_gap">
					{{ loadError }}
				</CfInfo>
				<template v-else-if="antenna != null">
					<div class="antennaHeader _block">
						<div class="name">
							<i class="ph-flying-saucer ph-bold"></i>
							<span>{{ antenna.name }}</span>
						</div>
						<div class="meta">{{ sourceLabel }}</div>
					</div>
					<div class="tl _block">
						<MkTimeline
							:key="antennaId"
							src="antenna"
							:antenna-id="antennaId"
							:sound="true"
						/>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's antenna timeline: load the antenna header, then render MkTimeline
// with src="antenna". The notes come from `antennas/notes` and the `antenna`
// streaming channel (there is no notes/antennas-timeline endpoint). Firefish's
// date jump (timetravel) action is dropped because MkTimeline has no timetravel
// support; the settings action opens the antenna editor Firefish links to.
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { Antenna } from "@/api/types";
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

defineOptions({ name: "CfPageAntennaTimeline" });

const route = useRoute();
const router = useRouter();

const antennaId = computed(() => String(route.params.antennaId ?? ""));
const antenna = ref<Antenna | null>(null);
const loading = ref(true);
const loadError = ref<string | null>(null);

const headerTabs = computed(() => []);

// Map the Misskey src values onto the locale keys Firefish uses. `home` is not
// offered by the editor but the backend accepts it, so a packed antenna can
// carry it; `users_blacklist` has no key and gets a literal suffix.
const sourceLabel = computed(() => {
	const src = antenna.value?.src;
	switch (src) {
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
});

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		antenna.value = await api(ENDPOINTS.antennasShow, {
			antennaId: antennaId.value,
		});
	} catch (cause) {
		antenna.value = null;
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

watch(antennaId, load, { immediate: true });

function openMenu(ev: MouseEvent): void {
	const id = antenna.value?.id ?? antennaId.value;
	const items: MenuItem[] = [
		{
			icon: "ph-sliders-horizontal ph-bold ph-lg",
			text: t("manageAntennas"),
			action: () => {
				void router.push(`/my/antennas/${id}`);
			},
		},
	];
	void os.popupMenu(items, ev);
}

const headerActions = computed(() =>
	antenna.value == null
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
		title: antenna.value?.name ?? t("antennas"),
		icon: "ph-flying-saucer ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-antenna-timeline {
	.antennaHeader {
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

		.meta {
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
