<!-- Ported from firefish@b425f61:packages/client/src/pages/my-antennas/edit.vue (AGPL-3.0-only) -->
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
			<div class="_formRoot cf-antenna-edit">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<template v-else-if="antenna != null">
					<CfInfo v-if="saveError !== null" warn class="_formBlock">
						{{ saveError }}
					</CfInfo>
					<AntennaEditor
						:key="antenna.id"
						:initial="antenna"
						:saving="saving"
						@submit="save"
					/>
					<div class="dangerRow _formBlock">
						<CfButton danger :disabled="deleting" @click="remove">
							<i class="ph-trash ph-bold ph-lg"></i>
							{{ t("delete") }}
						</CfButton>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's edit page loads `antennas/show`, renders the shared editor and
// returns to /my/antennas after an update. Cinderfish keeps the delete button
// in the page (the editor is shared with create, where delete does not apply).
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS, type AntennaInput } from "@/api/endpoints";
import type { Antenna } from "@/api/types";
import AntennaEditor from "@/components/AntennaEditor.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyAntennasEdit" });

const route = useRoute();
const router = useRouter();

const antennaId = computed(() => String(route.params.antennaId ?? ""));
const antenna = ref<Antenna | null>(null);
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const deleting = ref(false);
const saveError = ref<string | null>(null);

const headerTabs = computed(() => []);
const headerActions = computed(() => []);

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

async function save(payload: AntennaInput): Promise<void> {
	if (antenna.value == null || saving.value) return;
	saving.value = true;
	saveError.value = null;
	try {
		// Omitting a field would leave it unchanged; the editor emits the whole
		// set, which is equivalent and keeps create/edit on one payload type.
		await api(ENDPOINTS.antennasUpdate, {
			antennaId: antenna.value.id,
			...payload,
		});
		os.toast(t("saved"), "success");
		void router.push("/my/antennas");
	} catch (cause) {
		saveError.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

async function remove(): Promise<void> {
	if (antenna.value == null || deleting.value) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("removeAreYouSure", { x: antenna.value.name }),
	});
	if (canceled) return;
	deleting.value = true;
	try {
		await api(ENDPOINTS.antennasDelete, {
			antennaId: antenna.value.id,
		});
		void router.push("/my/antennas");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		deleting.value = false;
	}
}

definePageMetadata(
	computed(() => ({
		title: antenna.value?.name ?? t("manageAntennas"),
		icon: "ph-flying-saucer ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-antenna-edit {
	max-inline-size: 640px;

	.dangerRow {
		display: flex;
		justify-content: flex-end;
	}
}
</style>
