// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/notifications.vue
//
// Adaptations:
// - Firefish's MkPageHeader is PageHeader (ui/PageHeader.vue); the swiper is
//   dropped and tab switching remounts the feed component instead of sliding.
// - The type filter (reactions/mentions/directNotes tabs plus the per-type
//   includeTypes popover) maps onto the backend's includeTypes parameter;
//   notes/mentions and notes/direct-notes based tabs have no endpoint here,
//   so the mention tab filters the notifications feed instead (TODO: switch
//   to notes/mentions when the endpoint exists).
// - MkNotification becomes components/NotificationItem.vue; groupInvited and
//   app types are dropped where the backend never emits them (app is kept in
//   the item component but not offered as a tab).
// - The "mark all as read" action goes through api.markAllNotificationsRead;
//   unread state is shared with the shell through stores/streaming.ts.
<template>
	<PageHeader v-model:tab="tab" :title="i18.ts.notifications ?? '通知'" :tabs="session.me ? tabs : null">
		<template v-if="session.me" #actions>
			<button
				class="mark"
				type="button"
				:disabled="marking || unreadNotifications === 0"
				:title="marking ? i18.ts.processing : i18.ts.markAllAsRead"
				:aria-label="marking ? i18.ts.processing : i18.ts.markAllAsRead"
				@click="markAllRead"
			>
				<AppIcon name="check" />
			</button>
		</template>
	</PageHeader>

	<div v-if="!session.me" class="_panel _fullinfo auth-gate">
		<AppIcon class="ghost" name="bell" />
		<h2>需要登录</h2>
		<p>登录后可以查看回复、反应、关注与提及。</p>
		<MkButton link to="/login" primary>登录</MkButton>
	</div>

	<div v-else class="notifications-page">
		<!-- MkPagination is keyed on the tab: a switch remounts it and starts
		     from an empty, freshly fetched page for the new type instead of
		     paginating on top of the previous type's pages. -->
		<MkPagination :key="tab" ref="paginationEl" :pagination="pagination">
			<template #default="{ items }">
				<MkDateSeparatedList :items="items" no-gap>
					<template #default="{ item }">
						<NotificationItem :notification="item" />
					</template>
				</MkDateSeparatedList>
			</template>
			<template #empty>
				<div class="_fullinfo empty-placeholder">
					<AppIcon class="ghost" name="bell" />
					<p>{{ emptyText }}</p>
				</div>
			</template>
		</MkPagination>
	</div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api } from "../api/client";
import type { Notification } from "../api/types";
import { i18n as i18 } from "../i18n";
import { session } from "../stores/session";
import { setUnreadNotifications, streaming, unreadNotifications } from "../stores/streaming";
import NotificationItem from "../components/NotificationItem.vue";
import AppIcon from "../components/AppIcon.vue";
import MkButton from "../components/ui/MkButton.vue";
import MkDateSeparatedList from "../components/ui/MkDateSeparatedList.vue";
import MkPagination from "../components/ui/MkPagination.vue";
import PageHeader from "../components/ui/PageHeader.vue";

type Filter = "all" | "reaction" | "mention" | "renote";

// Type tabs are pushed down to the endpoint as includeTypes, so every page
// the feed receives is already of the selected type and an empty result means
// the type really has no notifications. "follow" is not offered: the backend
// never writes follow notification rows (following/create only emits a stream
// event), so the tab would always be empty.
const FILTERS: readonly Filter[] = ["all", "reaction", "mention", "renote"];

const LABELS: Record<Filter, string> = {
	all: i18.ts._notification._types.all,
	reaction: i18.ts._notification._types.reaction,
	mention: i18.ts._notification._types.mention,
	renote: i18.ts._notification._types.renote,
};

const tabs = FILTERS.map((key) => ({ key, label: LABELS[key] }));

const tab = defineModel<string>("tab", { default: "all" });

function isFilter(value: string): value is Filter {
	return (FILTERS as readonly string[]).includes(value);
}

if (!isFilter(tab.value)) tab.value = "all";

const filter = computed<Filter>(() => (isFilter(tab.value) ? tab.value : "all"));

function includeTypesFor(type: Filter): string[] | undefined {
	return type === "all" ? undefined : [type];
}

const emptyText = computed(() => {
	switch (filter.value) {
		case "reaction":
			return "还没有反应通知。";
		case "mention":
			return "还没有提及通知。";
		case "renote":
			return "还没有转贴通知。";
		default:
			return "还没有通知。";
	}
});

// A fresh config object per tab: the template keys MkPagination on the tab,
// so a switch remounts it and init() runs again.
const pagination = computed(() => ({
	endpoint: (params: Record<string, unknown>) =>
		api.notifications({
			limit: 20,
			untilId: typeof params.untilId === "string" ? params.untilId : undefined,
			includeTypes: includeTypesFor(filter.value),
			// The backend marks only the returned page read, and marking a
			// filtered page would leave the other types unread while the badge
			// already shows zero; the explicit "mark all as read" button is
			// the only read path.
			markAsRead: false,
		}),
	limit: 20,
}));

// The unread badge lives in the store, shared with the shell navigation. Seed
// it from the session once the account is loaded; live notifications bump it
// in the store itself.
let seeded = false;
watch(
	() => session.me?.unreadNotificationsCount,
	(count) => {
		if (seeded || typeof count !== "number") return;
		seeded = true;
		setUnreadNotifications(count);
	},
	{ immediate: true },
);

// Streamed notifications arrive on the main channel and are prepended into
// MkPagination, matching the original's onNotification (the isMuted check
// maps to the active includeTypes filter). The exposed handle is typed
// loosely because defineExpose does not surface the component's generic.
type PaginationApi = {
	items: Notification[];
	prepend: (...items: Notification[]) => void;
};

const paginationEl = ref<PaginationApi | null>(null);

function onNotification(body: unknown) {
	if (!body || typeof body !== "object") return;
	const notification = body as Notification;
	const types = includeTypesFor(filter.value);
	if (types && !types.includes(notification.type)) return;
	paginationEl.value?.prepend(notification);
}

function markLocalRead() {
	// The exposed items array is reactive; marking in place updates the rows
	// without a refetch, matching the original's local read sweep.
	for (const n of paginationEl.value?.items ?? []) n.isRead = true;
	setUnreadNotifications(0);
	if (session.me) {
		session.me.unreadNotificationsCount = 0;
		session.me.hasUnreadNotification = false;
	}
}

const marking = ref(false);

async function markAllRead() {
	if (marking.value || unreadNotifications.value === 0) return;
	marking.value = true;
	try {
		await api.markAllNotificationsRead();
		markLocalRead();
	} catch (err) {
		console.error("mark-all-as-read failed", err);
	} finally {
		marking.value = false;
	}
}

// "Mark all as read" is announced on the main channel, so a tab opened in a
// second client clears its dots without polling.
let subscription = "";
let offStream: (() => void) | null = null;

onMounted(() => {
	subscription = streaming.subscribe("main");
	offStream = streaming.on((channel, type, body) => {
		if (channel === "main" && type === "readAllNotifications") markLocalRead();
		if (channel === "main" && type === "notification") onNotification(body);
	});
});

onBeforeUnmount(() => {
	if (subscription) streaming.unsubscribe(subscription);
	subscription = "";
	offStream?.();
	offStream = null;
});
</script>

<style scoped>
.notifications-page {
	/* Firefish wraps the list in MkSpacer (12–24px); --margin gives the same
	   gutter as the other ported pages. */
	padding-block: var(--margin);
}

/* MkPagination's empty slot: Firefish centres the muted badge and its label in
   the column (._fullinfo already supplies the 64px padding). */
.empty-placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 12px;
	color: var(--fgTransparentWeak);
}

.empty-placeholder > .ghost {
	inline-size: 64px;
	block-size: 64px;
	opacity: 0.45;
}

.empty-placeholder > p {
	margin: 0;
	font-size: 0.95em;
	line-height: 1.5;
}

/* Firefish's unauthenticated state uses the same centred badge and label as
   its empty state; the panel only appears when the page has nothing else. */
.auth-gate {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 12px;
	color: var(--fgTransparentWeak);
}

.auth-gate > .ghost {
	inline-size: 64px;
	block-size: 64px;
	opacity: 0.45;
}

.auth-gate > h2 {
	margin: 0;
	font-size: 1.05em;
	color: var(--fg);
}

.auth-gate > p {
	margin: 0;
	font-size: 0.95em;
	line-height: 1.5;
}

.mark {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	width: 34px;
	height: 34px;
	padding: 0;
	border: 0;
	border-radius: 50%;
	background: transparent;
	color: var(--muted);
	cursor: pointer;
	transition: color 0.15s ease, background 0.15s ease;
}

.mark .icon {
	width: 16px;
	height: 16px;
}

.mark:hover:not(:disabled) {
	color: var(--text);
	background: var(--panel-2);
}

.mark:focus-visible {
	outline: 2px solid var(--accent);
	outline-offset: 2px;
}

.mark:disabled {
	opacity: 0.5;
	cursor: default;
}
</style>
