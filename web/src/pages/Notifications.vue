<!-- Ported from firefish@b425f61:packages/client/src/pages/notifications.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tab="tab"
				:tabs="headerTabs"
				@update:tab="onTabUpdate"
			/>
		</template>
		<MkSpacer :content-max="800">
			<MkNotifications
				v-if="tab !== 'mentions'"
				:key="`${tab}:${filterKey}`"
				ref="notificationsComponent"
				class="notifications"
				:include-types="includeTypes"
				:unread-only="tab === 'unread'"
			/>
			<MkNotes v-else :pagination="mentionsPagination" />
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's notification page tabs were unread / mentions / directNotes. The
// backend now has `unreadOnly` and `notes/mentions`, so the first two are back;
// directNotes stays out because there is no chat scope in this port.
import { computed, ref } from "vue";
import type { NotificationType } from "@/api/types";
import MkNotifications from "@/components/MkNotifications.vue";
import MkNotes from "@/components/MkNotes.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import type { Paging } from "@/components/ui";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os/types";
import { definePageMetadata } from "@/scripts/page-metadata";

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

type Tab = "all" | "unread" | "mentions";
const tab = ref<Tab>("all");

const headerTabs = computed(() => [
	{ key: "all", title: t("all"), icon: "ph-bell ph-bold ph-lg" },
	{ key: "unread", title: t("unread"), icon: "ph-envelope-open ph-bold ph-lg" },
	{ key: "mentions", title: t("mentions"), icon: "ph-at ph-bold ph-lg" },
]);

const mentionsPagination: Paging = {
	endpoint: "notes/mentions",
	limit: 10,
	params: { following: false },
};

function onTabUpdate(key: string): void {
	if (key !== "all" && key !== "unread" && key !== "mentions") return;
	tab.value = key;
}

// Reading the list must not swallow the "unread" tab: individual notifications
// are marked read when they are displayed (MkNotifications), and this action
// stays explicit.
function readAll(): void {
	void notificationsComponent.value?.readAllNotifications();
}

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
					...typeItems,
					{ separator: true },
					{
						text: "显示全部通知",
						icon: "ph-list-bullets ph-bold ph-lg",
						action: () => {
							includeTypes.value = null;
						},
					},
				]
			: typeItems;

	void os.popupMenu(items, ev);
}

const headerActions = computed(() => [
	{
		icon: "ph-check-square ph-bold ph-lg",
		text: "全部标为已读",
		handler: readAll,
	},
	{
		icon: "ph-funnel ph-bold ph-lg",
		text: t("filter"),
		handler: setFilter,
	},
]);

definePageMetadata({
	title: t("notifications"),
	icon: "ph-bell ph-bold ph-lg",
});
</script>
