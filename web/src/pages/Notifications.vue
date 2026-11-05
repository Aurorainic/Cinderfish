<!-- Ported from firefish@b425f61:packages/client/src/pages/notifications.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<MkNotifications
				:key="filterKey"
				ref="notificationsComponent"
				class="notifications"
				:include-types="includeTypes"
			/>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import type { NotificationType } from "@/api/types";
import MkNotifications from "@/components/MkNotifications.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os/types";
import { definePageMetadata } from "@/scripts/page-metadata";

// Cinderfish produces only these five notification kinds (contract §5), so the
// upstream tab set (unread / mentions / directNotes) is dropped: the backend
// has neither `unreadOnly` nor `notes/mentions`.
const notificationTypes: NotificationType[] = [
	"mention",
	"reply",
	"renote",
	"quote",
	"reaction",
];

const notificationsComponent = ref<{
	readAllNotifications: () => Promise<void>;
} | null>(null);

const includeTypes = ref<NotificationType[] | null>(null);
const filterKey = computed(() => includeTypes.value?.join(",") ?? "all");

onMounted(() => {
	void notificationsComponent.value?.readAllNotifications();
});

function setFilter(ev: MouseEvent): void {
	const typeItems: MenuItem[] = notificationTypes.map((type) => ({
		text: t(`_notification._types.${type}`),
		active: includeTypes.value?.includes(type) ?? false,
		action: () => {
			includeTypes.value = [type];
		},
	}));

	const items: MenuItem[] =
		includeTypes.value != null
			? [
					{
						icon: "ph-x ph-bold ph-lg",
						text: t("clear"),
						action: () => {
							includeTypes.value = null;
						},
					},
					{ separator: true },
					...typeItems,
				]
			: typeItems;

	void os.popupMenu(items, ev);
}

const headerActions = computed(() => [
	{
		text: t("filter"),
		icon: "ph-funnel ph-bold ph-lg",
		handler: setFilter,
	},
	{
		text: t("markAllAsRead"),
		icon: "ph-check ph-bold ph-lg",
		handler: () => {
			void notificationsComponent.value?.readAllNotifications();
		},
	},
]);

definePageMetadata(
	computed(() => ({
		title: t("notifications"),
		icon: "ph-bell ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.notifications {
	background: var(--panel);
	border-radius: var(--radius);
}
</style>
