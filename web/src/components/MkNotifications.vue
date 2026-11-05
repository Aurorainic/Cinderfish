<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotifications.vue (AGPL-3.0-only) -->
<template>
	<CfPagination ref="pagingComponent" :pagination="pagination">
		<template #empty>
			<div class="_fullinfo">
				<img
					src="/static-assets/badges/info.svg"
					class="_ghost"
					alt="Info"
				/>
				<div>{{ t("noNotifications") }}</div>
			</div>
		</template>

		<template #default="{ items: notifications }">
			<CfDateSeparatedList
				v-slot="{ item: notification }"
				class="elsfgstc"
				:items="notificationItems(notifications)"
				:no-gap="true"
			>
				<MkNote
					v-if="
						notification.note != null &&
						['reply', 'quote', 'mention'].includes(notification.type)
					"
					:key="notification.id"
					:note="notification.note"
					:collapsedReply="
						notification.type === 'reply' ||
						(notification.type === 'mention' &&
							notification.note.replyId != null)
					"
				/>
				<MkNotification
					v-else
					:key="notification.id"
					:notification="notification"
					:with-time="true"
					:full="true"
					class="_panel notification"
				/>
			</CfDateSeparatedList>
		</template>
	</CfPagination>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { api } from "@/api/client";
import type { Notification, NotificationType } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import { stream } from "@/stream";
import { CfDateSeparatedList, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import MkNotification from "@/components/MkNotification.vue";
import MkNote from "@/components/MkNote.vue";

const props = defineProps<{
	includeTypes?: NotificationType[] | null;
}>();

interface NotificationsPaging {
	items: Notification[];
	reload: () => Promise<Notification[]>;
	prepend: (...items: Notification[]) => void;
}

const pagingComponent = ref<NotificationsPaging | null>(null);

// The backend only filters by `includeTypes` / `excludeTypes`; explicit empty
// arrays mean "everything" for include and "nothing" for exclude, so absent
// filters are omitted instead of sent empty (docs/research/api-contract/
// timelines-notifications.md).
const mutedTypes: string[] = me.value?.mutingNotificationTypes ?? [];

const pagination: Paging = {
	endpoint: "i/notifications",
	limit: 10,
	params: {
		// An explicit empty `includeTypes` would select nothing, so it is treated
		// as "no filter" instead.
		includeTypes:
			props.includeTypes != null && props.includeTypes.length > 0
				? props.includeTypes
				: undefined,
		excludeTypes: props.includeTypes
			? undefined
			: mutedTypes.length > 0
				? mutedTypes
				: undefined,
	},
};

// CfPagination is generic over `{ id: string }` when instantiated from a
// template, so the slot payload is re-typed here instead of relying on
// inference that cannot see through the `pagination` prop.
function notificationItems(items: readonly { id: string }[]): Notification[] {
	return items as Notification[];
}

const isMuted = computed(
	() => (type: NotificationType): boolean =>
		props.includeTypes != null && props.includeTypes.length > 0
			? !props.includeTypes.includes(type)
			: mutedTypes.includes(type),
);

function onNotification(notification: Notification): void {
	if (isMuted.value(notification.type) || document.visibilityState === "visible") {
		stream.send("readNotification", { id: notification.id });
	}

	if (!isMuted.value(notification.type)) {
		pagingComponent.value?.prepend({
			...notification,
			isRead: document.visibilityState === "visible",
		});
	}
}

function markAllAsReadLocally(): void {
	const items = pagingComponent.value?.items;
	if (items == null) return;
	for (const item of items) item.isRead = true;
}

/**
 * Mark every notification read: the REST endpoint persists it and the
 * `readNotification` streaming frame does the same thing without an extra
 * round-trip (the server treats both identically).
 */
async function readAllNotifications(): Promise<void> {
	try {
		await api("notifications/mark-all-as-read");
	} catch (error) {
		console.error(error);
	}
	markAllAsReadLocally();
	stream.readNotification();
}

// `main` is a per-connection singleton, so listeners are detached one by one
// instead of disposing the shared subscription.
const offHandlers: (() => void)[] = [];

onMounted(() => {
	const channel = stream.useChannel("main");
	offHandlers.push(channel.on("notification", onNotification));
	offHandlers.push(channel.on("readAllNotifications", markAllAsReadLocally));
});

onUnmounted(() => {
	for (const off of offHandlers.splice(0)) off();
});

defineExpose({
	readAllNotifications,
	reload: () => pagingComponent.value?.reload(),
});
</script>

<style lang="scss" scoped>
.elsfgstc {
	background: var(--panel);
	border-radius: var(--radius);
}
</style>
