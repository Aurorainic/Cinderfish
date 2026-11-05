<!-- Ported from firefish@b425f61:packages/client/src/widgets/notifications.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('notifications')"
		icon="ph-bold ph-bell"
		:show-header="widgetProps.showHeader === true"
		:height="height"
		:scrollable="true"
	>
		<template #actions>
			<button class="_button" :aria-label="t('markAllAsRead')" @click="markAllAsRead">
				<i class="ph-bold ph-check"></i>
			</button>
			<button class="_button" :aria-label="t('notificationSetting')" @click="configureNotification">
				<i class="ph-bold ph-gear-six"></i>
			</button>
		</template>
		<MkNotifications :include-types="includeTypes" />
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish forwards `includingTypes` to MkNotificationSettingWindow and stores
// the selection in the widget's own props. Cinderfish has that window, so the
// flow is unchanged; only the container is WidgetCard instead of MkContainer.
import { computed } from "vue";
import WidgetCard from "./WidgetCard.vue";
import MkNotifications from "@/components/MkNotifications.vue";
import MkNotificationSettingWindow from "@/components/MkNotificationSettingWindow.vue";
import type { NotificationType } from "@/api/types";
import { apiWithDialog, popup } from "@/os";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "notifications";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
	{ key: "height", label: "高度 (px)", type: "number", default: 300, min: 120, max: 800, step: 10 },
];

const { widgetProps, configure, save } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const height = computed(() => {
	const value = widgetProps.height;
	return typeof value === "number" ? value : 300;
});

/** An empty `includeTypes` means "everything", which MkNotifications expects as null. */
const includeTypes = computed<NotificationType[] | null>(() => {
	const value = widgetProps.includingTypes;
	return Array.isArray(value) && value.length > 0
		? (value as NotificationType[])
		: null;
});

function markAllAsRead(): void {
	void apiWithDialog("notifications/mark-all-as-read");
}

function configureNotification(): void {
	popup(
		MkNotificationSettingWindow,
		{ includingTypes: widgetProps.includingTypes ?? null },
		{
			done: (result: { includingTypes: string[] | null }) => {
				widgetProps.includingTypes = result.includingTypes;
				save();
			},
		},
		"closed",
	);
}

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>
