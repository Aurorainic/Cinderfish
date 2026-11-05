<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotificationSettingWindow.vue (AGPL-3.0-only) -->
<template>
	<CfModal v-if="open" @closed="close">
		<div class="_panel msdskfuq">
			<div class="header">{{ t("notificationSetting") }}</div>
			<div class="_monolithic_">
				<div v-if="showGlobalToggle" class="_section">
					<CfSwitch v-model="useGlobalSetting">
						{{ t("useGlobalSetting") }}
						<template #caption>{{
							t("useGlobalSettingDesc")
						}}</template>
					</CfSwitch>
				</div>
				<div v-if="!useGlobalSetting" class="_section">
					<CfInfo>{{ t("notificationSettingDesc") }}</CfInfo>
					<CfButton inline @click="disableAll">{{
						t("disableAll")
					}}</CfButton>
					<CfButton inline @click="enableAll">{{
						t("enableAll")
					}}</CfButton>
					<CfSwitch
						v-for="ntype in notificationTypes"
						:key="ntype"
						v-model="typesMap[ntype]"
						>{{ t(`_notification._types.${ntype}`) }}</CfSwitch
					>
				</div>
			</div>
			<div class="footer">
				<CfButton primary @click="ok">{{ t("ok") }}</CfButton>
			</div>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import type { NotificationType } from "@/api/types";
import { CfButton, CfInfo, CfSwitch } from "@/components/ui";
import { t } from "@/i18n";
import { CfModal } from "@/os";

// Upstream builds this list from `firefish-js`'s notificationTypes. Cinderfish
// produces only the five kinds below, and its API has no `unreadOnly` switch.
const notificationTypes: NotificationType[] = [
	"mention",
	"reply",
	"renote",
	"quote",
	"reaction",
];

const emit = defineEmits<{
	done: [value: { includingTypes: string[] | null }];
	closed: [];
}>();

const props = withDefaults(
	defineProps<{
		includingTypes?: string[] | null;
		showGlobalToggle?: boolean;
	}>(),
	{
		includingTypes: () => [],
		showGlobalToggle: true,
	},
);

const open = ref(true);

const includingTypes = props.includingTypes || [];
const useGlobalSetting = ref(
	(includingTypes === null || includingTypes.length === 0) &&
		props.showGlobalToggle,
);

const typesMap = reactive<Record<NotificationType, boolean>>({
	mention: false,
	reply: false,
	renote: false,
	quote: false,
	reaction: false,
});

for (const ntype of notificationTypes) {
	typesMap[ntype] = includingTypes.includes(ntype);
}

function close(): void {
	open.value = false;
	emit("closed");
}

function ok(): void {
	if (useGlobalSetting.value) {
		emit("done", { includingTypes: null });
	} else {
		emit("done", {
			includingTypes: notificationTypes.filter((type) => typesMap[type]),
		});
	}

	close();
}

function disableAll(): void {
	for (const type of notificationTypes) typesMap[type] = false;
}

function enableAll(): void {
	for (const type of notificationTypes) typesMap[type] = true;
}
</script>

<style lang="scss" scoped>
.msdskfuq {
	display: flex;
	flex-direction: column;
	width: min(400px, calc(100vw - 32px));
	max-height: min(450px, calc(100vh - 64px));
	overflow: auto;
	background: var(--panel);
	border-radius: var(--radius);

	> .header {
		padding: 16px;
		font-weight: bold;
		border-bottom: solid 1px var(--divider);
	}

	> ._monolithic_ {
		flex: 1;
		overflow: auto;
	}

	> .footer {
		display: flex;
		justify-content: flex-end;
		padding: 12px;
		border-top: solid 1px var(--divider);
	}
}
</style>
