<!-- Ported from firefish@b425f61:packages/client/src/pages/my-antennas/editor.vue (AGPL-3.0-only) -->
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
			<div class="_formRoot cf-antenna-create">
				<CfInfo v-if="error !== null" warn class="_formBlock">
					{{ error }}
				</CfInfo>
				<AntennaEditor :saving="saving" @submit="save" />
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish renders the blank editor inside a dedicated create page. Saving
// posts the whole required field set to `antennas/create` and returns to the
// antenna manager.
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { api, ApiError } from "@/api/client";
import { ENDPOINTS, type AntennaInput } from "@/api/endpoints";
import AntennaEditor from "@/components/AntennaEditor.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyAntennasCreate" });

const router = useRouter();

const saving = ref(false);
const error = ref<string | null>(null);

const headerTabs = computed(() => []);
const headerActions = computed(() => []);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function save(payload: AntennaInput): Promise<void> {
	if (saving.value) return;
	saving.value = true;
	error.value = null;
	try {
		await api(ENDPOINTS.antennasCreate, payload);
		os.toast(t("saved"), "success");
		void router.push("/my/antennas");
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

definePageMetadata(
	computed(() => ({
		title: t("antennas"),
		icon: "ph-flying-saucer ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-antenna-create {
	max-inline-size: 640px;
}
</style>
