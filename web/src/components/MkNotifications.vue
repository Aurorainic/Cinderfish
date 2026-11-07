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
				v-slot="{ item: row }"
				class="elsfgstc"
				:items="notificationRows(notifications)"
				:no-gap="true"
			>
				<MkNote
					v-if="
						row.notification.note != null &&
						['reply', 'quote', 'mention'].includes(
							row.notification.type,
						)
					"
					:key="row.id"
					:note="row.notification.note"
					:collapsedReply="
						row.notification.type === 'reply' ||
						(row.notification.type === 'mention' &&
							row.notification.note.replyId != null)
					"
				/>
				<MkNotification
					v-else
					:key="row.id"
					:notification="row.notification"
					:ids="row.ids"
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
import { defaultStore } from "@/store/base";
import { CfDateSeparatedList, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import MkNotification from "@/components/MkNotification.vue";
import MkNote from "@/components/MkNote.vue";

const props = defineProps<{
	includeTypes?: NotificationType[] | null;
	/** Only unread notifications (`i/notifications`'s unreadOnly). */
	unreadOnly?: boolean;
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
		...(props.unreadOnly === true ? { unreadOnly: true } : {}),
	},
};

// CfPagination is generic over `{ id: string }` when instantiated from a
// template, so the slot payload is re-typed here instead of relying on
// inference that cannot see through the `pagination` prop.
interface NotificationRow {
	id: string;
	createdAt: string;
	notification: Notification;
	/** Every notification id the row stands for, newest first. */
	ids: string[];
}

/**
 * Fold notifications that describe the same thing on the same note: five
 * people reacting with 🎉 to one post become one row with a `×5` badge. Only
 * types that can legitimately repeat are folded; a reply or a mention happens
 * once per note and stays on its own.
 *
 * Folding happens here rather than on the server because it is a display
 * preference (`foldNotification`) and the unread list still needs every row to
 * be marked read, which the row does with its whole `ids` group.
 */
function foldKey(n: Notification): string | null {
	if (n.note == null) return null;
	if (n.type !== "reaction" && n.type !== "renote") return null;
	return `${n.type}:${n.note.id}:${n.reaction ?? ""}`;
}

function notificationRows(items: readonly { id: string }[]): NotificationRow[] {
	const notifications = items as Notification[];
	const rows: NotificationRow[] = [];
	if (!defaultStore.reactiveState.foldNotification.value) {
		for (const n of notifications) {
			rows.push({
				id: n.id,
				createdAt: n.createdAt,
				notification: n,
				ids: [n.id],
			});
		}
		return rows;
	}

	const byKey = new Map<string, NotificationRow>();
	for (const n of notifications) {
		const key = foldKey(n);
		const existing = key == null ? undefined : byKey.get(key);
		if (existing != null) {
			existing.ids.push(n.id);
			continue;
		}
		const row: NotificationRow = {
			id: n.id,
			createdAt: n.createdAt,
			notification: n,
			ids: [n.id],
		};
		rows.push(row);
		if (key != null) byKey.set(key, row);
	}
	return rows;
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
