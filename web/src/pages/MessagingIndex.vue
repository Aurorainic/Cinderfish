<!-- Ported from firefish@b425f61:packages/client/src/pages/messaging/index.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="cf-messaging-index">
				<CfPagination
					ref="pagingComponent"
					:pagination="pagination"
					class="list"
				>
					<template #empty>
						<div class="_fullinfo">
							<i class="ph-chats-teardrop ph-bold ph-lg"></i>
							<div>{{ t("noMessagesYet") }}</div>
						</div>
					</template>
					<template #default="{ items }">
						<MkChatPreview
							v-for="message in sortedMessages(items)"
							:key="message.id"
							:message="message"
						/>
					</template>
				</CfPagination>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's messaging index with the groups tab and every `users/groups/*`
// call removed: group messaging is a recorded backend deferral, so the DM tab
// is the whole page. `messaging/history { group: false }` returns one latest
// message per conversation and embeds the recipient, which is what
// MkChatPreview renders.
//
// Live updates ride `messagingIndex`: `message` refreshes the list (the event
// payload omits `recipient`, so re-fetching is more correct than inserting it),
// `read` clears the unread styling of conversations read on this account.
import { computed, onMounted, onUnmounted, ref } from "vue";
import { me, updateMe } from "@/account";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { MessagingMessage } from "@/api/types";
import type { MessagingReadEvent, StreamChannel } from "@/api/stream";
import MkChatPreview from "@/components/MkChatPreview.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import type { Paging } from "@/components/ui";
import { t } from "@/i18n";
import { success, toast } from "@/os";
import { stream } from "@/stream";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMessagingIndex" });

interface ExposedPagination {
	items: MessagingMessage[];
	refresh: () => Promise<void>;
	reload: () => Promise<unknown>;
}

const pagination: Paging = {
	endpoint: ENDPOINTS.messagingHistory,
	params: { group: false },
	// `messaging/history` ignores cursor params and caps at 100, so one page
	// is the whole conversation list; a larger limit avoids a dead "load more".
	limit: 100,
};

const pagingComponent = ref<ExposedPagination | null>(null);
const marking = ref(false);

const headerActions = computed(() => [
	{
		icon: "ph-check ph-bold ph-lg",
		text: t("markAllAsRead"),
		handler: () => {
			void markAllAsRead();
		},
	},
]);

/**
 * CfPagination's slot is generic over `{ id: string }`; this endpoint is always
 * messages. The backend orders `messaging/history` by peer id (the DISTINCT ON
 * key), so the newest conversation is not necessarily first; sort for display.
 */
function sortedMessages(items: { id: string }[]): MessagingMessage[] {
	return (items as MessagingMessage[])
		.slice()
		.sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt));
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

function markAllReadLocally(): void {
	const items = pagingComponent.value?.items;
	if (items == null) return;
	for (const message of items) message.isRead = true;
}

async function markAllAsRead(): Promise<void> {
	if (marking.value) return;
	marking.value = true;
	try {
		await api(ENDPOINTS.iReadAllMessagingMessages, {});
		markAllReadLocally();
		updateMe({ hasUnreadMessagingMessage: false });
		success();
	} catch (cause) {
		toast(errorText(cause), "error");
	} finally {
		marking.value = false;
	}
}

function onMessage(): void {
	// The live payload has no embedded `recipient`; refresh instead so a
	// conversation started by the viewer keeps its peer header.
	void pagingComponent.value?.refresh();
}

function onRead(event: MessagingReadEvent): void {
	if (me.value == null || event.readerId !== me.value.id) return;
	const items = pagingComponent.value?.items;
	if (items == null) return;
	// The event carries only the reader, not the conversation. Clearing every
	// incoming row matches Firefish's "conversation was read" styling.
	for (const message of items) {
		if (message.userId !== me.value.id) message.isRead = true;
	}
}

let connection: StreamChannel | null = null;
const offHandlers: (() => void)[] = [];

onMounted(() => {
	const channel = stream.useChannel("messagingIndex");
	connection = channel;
	offHandlers.push(channel.on("message", onMessage));
	offHandlers.push(channel.on("read", onRead));
});

onUnmounted(() => {
	for (const off of offHandlers.splice(0)) off();
	connection?.dispose();
	connection = null;
});

definePageMetadata(
	computed(() => ({
		title: t("messaging"),
		icon: "ph-chats-teardrop ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-messaging-index {
	> .list {
		display: block;
	}
}
</style>
