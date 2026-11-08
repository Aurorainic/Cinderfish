<!-- Ported from firefish@b425f61:packages/client/src/pages/channel-editor.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" :tabs="headerTabs" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-channel-editor">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn class="_formBlock">
					{{ loadError }}
				</CfInfo>
				<template v-else>
					<CfInput v-model="name" class="_formBlock">
						<template #label>{{ t("name") }}</template>
					</CfInput>

					<CfTextarea v-model="description" class="_formBlock">
						<template #label>{{ t("description") }}</template>
						<template #caption>{{ t("optional") }}</template>
					</CfTextarea>

					<div class="_formBlock bannerBlock">
						<div class="label">{{ t("banner") }}</div>
						<div v-if="bannerUrl !== null" class="preview">
							<img :src="bannerUrl" alt="" />
							<CfButton :disabled="saving" @click="removeBanner">
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("_channel.removeBanner") }}
							</CfButton>
						</div>
						<CfButton v-else :disabled="saving" @click="pickBanner">
							<i class="ph-plus ph-bold ph-lg"></i>
							{{ t("_channel.setBanner") }}
						</CfButton>
					</div>

					<div class="_formBlock colorBlock">
						<CfInput v-model="color" :placeholder="'#RRGGBB'">
							<template #label>Color</template>
						</CfInput>
						<span
							class="swatch"
							:style="{
								backgroundColor:
									color !== '' ? color : 'transparent',
							}"
						></span>
					</div>

					<CfSwitch v-model="isSensitive" class="_formBlock">
						{{ t("sensitive") }}
					</CfSwitch>

					<CfSwitch v-model="allowRenoteToExternal" class="_formBlock">
						Allow renoting outside of the channel
					</CfSwitch>

					<div class="actions _formBlock">
						<CfButton
							primary
							:disabled="saving || name.trim() === ''"
							@click="save"
						>
							<i class="ph-floppy-disk-back ph-bold ph-lg"></i>
							{{ isEdit ? t("save") : t("_channel.create") }}
						</CfButton>
						<CfButton
							v-if="isEdit && !isArchived"
							danger
							:disabled="saving || archiving"
							@click="archive"
						>
							<i class="ph-archive ph-bold ph-lg"></i>
							Archive
						</CfButton>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's channel editor: name, description and banner. The port adds the
// remaining writable fields the backend implements (color, sensitive flag and
// the renote-outside switch) and turns "delete" into the archive update —
// there is no `channels/delete`, archiving through `channels/update` is the
// deletion. Clearing the banner sends `bannerId: ""`, which the backend treats
// as an explicit clear.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { router } from "@/router";
import * as os from "@/os";
import { selectFile } from "@/scripts/select-file";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageChannelEditor" });

const route = useRoute();

const channelId = computed<string | null>(() => {
	const value = route.params.channelId;
	return typeof value === "string" && value !== "" ? value : null;
});
const isEdit = computed(() => channelId.value != null);

const loading = ref(false);
const loadError = ref<string | null>(null);
const saving = ref(false);
const archiving = ref(false);

const name = ref("");
const description = ref("");
const bannerId = ref<string | null>(null);
const bannerUrl = ref<string | null>(null);
const color = ref("");
const isSensitive = ref(false);
const allowRenoteToExternal = ref(true);
const isArchived = ref(false);

const headerTabs = computed(() => []);
const headerActions = computed(() => []);

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function load(): Promise<void> {
	if (channelId.value == null) {
		loading.value = false;
		return;
	}
	loading.value = true;
	loadError.value = null;
	try {
		const channel = await api(ENDPOINTS.channelsShow, {
			channelId: channelId.value,
		});
		name.value = channel.name;
		description.value = channel.description ?? "";
		bannerId.value = channel.bannerId;
		bannerUrl.value = channel.bannerUrl;
		color.value = channel.color;
		isSensitive.value = channel.isSensitive;
		allowRenoteToExternal.value = channel.allowRenoteToExternal;
		isArchived.value = channel.isArchived;
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function pickBanner(event: MouseEvent): Promise<void> {
	const file = await selectFile({ label: t("banner"), src: event });
	if (file == null) return;
	bannerId.value = file.id;
	bannerUrl.value = file.url;
}

function removeBanner(): void {
	bannerId.value = null;
	bannerUrl.value = null;
}

async function save(): Promise<void> {
	const trimmed = name.value.trim();
	if (saving.value || trimmed === "") return;
	const trimmedColor = color.value.trim();
	// `channels/update` rejects an empty color, and `channels/create` defaults
	// it, so a blank field is omitted instead of sent as "".
	const colorParam =
		trimmedColor === "" ? {} : { color: trimmedColor };
	saving.value = true;
	try {
		if (channelId.value != null) {
			await api(ENDPOINTS.channelsUpdate, {
				channelId: channelId.value,
				name: trimmed,
				// An empty string clears the stored description.
				description: description.value.trim(),
				// "" clears the banner on the backend (upstream silently kept it).
				bannerId: bannerId.value ?? "",
				...colorParam,
				isSensitive: isSensitive.value,
				allowRenoteToExternal: allowRenoteToExternal.value,
			});
			os.success();
		} else {
			const created = await api(ENDPOINTS.channelsCreate, {
				name: trimmed,
				description:
					description.value.trim() === ""
						? null
						: description.value.trim(),
				...(bannerId.value != null
					? { bannerId: bannerId.value }
					: {}),
				...colorParam,
				isSensitive: isSensitive.value,
				allowRenoteToExternal: allowRenoteToExternal.value,
			});
			os.success();
			await router.replace(`/channels/${created.id}`);
		}
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function archive(): Promise<void> {
	if (channelId.value == null || archiving.value) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("removeAreYouSure", { x: name.value }),
	});
	if (canceled) return;
	archiving.value = true;
	try {
		await api(ENDPOINTS.channelsUpdate, {
			channelId: channelId.value,
			isArchived: true,
		});
		os.success();
		await router.push("/channels");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		archiving.value = false;
	}
}

watch(channelId, () => void load(), { immediate: true });

definePageMetadata(
	computed(() => ({
		title: isEdit.value ? t("_channel.edit") : t("_channel.create"),
		icon: "ph-television ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-channel-editor {
	.bannerBlock {
		> .label {
			margin-block-end: 8px;
			font-weight: bold;
		}

		> .preview {
			display: flex;
			flex-direction: column;
			align-items: flex-start;
			gap: 8px;

			> img {
				inline-size: 100%;
				max-block-size: 220px;
				object-fit: cover;
				border-radius: var(--radius);
			}
		}
	}

	.colorBlock {
		display: flex;
		align-items: flex-end;
		gap: 8px;

		:deep(.matxzzsk) {
			flex: 1;
			min-width: 0;
		}

		> .swatch {
			flex: 0 0 auto;
			inline-size: 34px;
			block-size: 34px;
			margin-block-end: 1px;
			border: solid 1px var(--divider);
			border-radius: 6px;
		}
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
}
</style>
