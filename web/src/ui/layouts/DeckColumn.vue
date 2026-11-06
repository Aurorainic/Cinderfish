<!-- Ported from firefish@b425f61:packages/client/src/ui/deck/column.vue (AGPL-3.0-only) -->
<!--
  This file also absorbs the column kinds upstream splits into
  `ui/deck/column-core.vue`, `main-column.vue`, `tl-column.vue`,
  `notifications-column.vue` and `widgets-column.vue`: a deck column is a header
  plus one body, so one component covers them. The `favorites` kind is a
  Cinderfish addition (`i/favorites` exists), and `mentions` uses the existing
  `notes/mentions` endpoint; list/antenna/channel/direct have no backend
  endpoints (they are outside the first-release scope) and render the "not
  provided" notice.
-->
<template>
	<section
		class="deck-column"
		:class="{ stacked: isStacked, collapsed: !active, active }"
	>
		<header class="header" :class="{ indicated }" @click="goTop">
			<button
				v-if="isStacked && !isMainColumn"
				class="icon-button toggle _button"
				@click.stop="toggleActive"
			>
				<i
					:class="
						active
							? 'ph-caret-up ph-bold ph-lg'
							: 'ph-caret-down ph-bold ph-lg'
					"
				></i>
			</button>
			<span class="title">
				<i v-if="headerIcon != null" :class="headerIcon"></i>
				<span class="name">{{ title }}</span>
			</span>
			<button
				v-if="!virtual"
				class="icon-button menu _button"
				:aria-label="t('settings')"
				@click.stop="openMenu"
			>
				<i class="ph-dots-three-outline-vertical ph-bold ph-lg"></i>
			</button>
		</header>

		<div v-show="active" ref="bodyEl" class="body">
			<RouterView v-if="column.type === 'main'" />

			<MkTimeline
				v-else-if="column.type === 'tl' && timelineSource != null"
				:key="timelineSource"
				:src="timelineSource"
				@queue="onQueue"
				@note="onNote"
			/>

			<MkNotifications
				v-else-if="column.type === 'notifications'"
				:include-types="column.includingTypes"
			/>

			<MkWidgets v-else-if="column.type === 'widgets'" />

			<CfPagination
				v-else-if="column.type === 'favorites'"
				class="favorites"
				:pagination="favoritesPagination"
			>
				<template #empty>
					<div class="_fullinfo">
						<div>{{ t("noNotes") }}</div>
					</div>
				</template>
				<template #default="{ items }">
					<MkNote
						v-for="favorite in favoriteItems(items)"
						:key="favorite.id"
						:note="favorite.note"
					/>
				</template>
			</CfPagination>

			<CfPagination
				v-else-if="column.type === 'mentions'"
				class="mentions"
				:pagination="mentionsPagination"
			>
				<template #empty>
					<div class="_fullinfo">
						<div>{{ t("noNotes") }}</div>
					</div>
				</template>
				<template #default="{ items }">
					<MkNote
						v-for="note in noteItems(items)"
						:key="note.id"
						:note="note"
					/>
				</template>
			</CfPagination>

			<div v-else class="unsupported">
				<CfInfo :warn="true">
					<div class="unsupported-title">
						{{ columnLabel(column.type) }}
					</div>
					<div>未提供：Cinderfish 后端暂无该列所需的接口。</div>
				</CfInfo>
			</div>
		</div>
	</section>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from "vue";
import type { Note, NoteFavorite } from "@/api/types";
import {
	CfInfo,
	CfPagination,
	type Paging,
} from "@/components/ui";
import MkNote from "@/components/MkNote.vue";
import MkNotifications from "@/components/MkNotifications.vue";
import MkTimeline from "@/components/MkTimeline.vue";
import MkWidgets from "@/components/MkWidgets.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import type { TimelineSource } from "@/store/base";
import {
	popRightColumn,
	removeColumn,
	stackLeftColumn,
	swapDownColumn,
	swapLeftColumn,
	swapRightColumn,
	swapUpColumn,
	updateColumn,
	type Column,
} from "@/store/deck";

const props = withDefaults(
	defineProps<{
		column: Column;
		isStacked?: boolean;
		/** The layout's unpersisted main column: no column menu, no config. */
		virtual?: boolean;
	}>(),
	{
		isStacked: false,
		virtual: false,
	},
);

// Firefish measures the collapsed column by its header; the deck layout uses
// the same height, so only the body depends on `active`.
const bodyEl = ref<HTMLElement | null>(null);
const indicated = ref(false);

const isMainColumn = computed(() => props.column.type === "main");
const active = computed(() => props.column.active !== false);

const TIMELINE_SOURCES: TimelineSource[] = [
	"home",
	"local",
	"social",
	"global",
];

const TIMELINE_ICONS: Record<TimelineSource, string> = {
	home: "ph-house ph-bold ph-lg",
	local: "ph-chats-circle ph-bold ph-lg",
	social: "ph-share-network ph-bold ph-lg",
	global: "ph-planet ph-bold ph-lg",
};

const timelineSource = computed<TimelineSource | null>(() => {
	const source = props.column.tl;
	if (source == null) return null;
	return (TIMELINE_SOURCES as string[]).includes(source)
		? (source as TimelineSource)
		: null;
});

const headerIcon = computed<string | null>(() => {
	switch (props.column.type) {
		case "tl":
			return TIMELINE_ICONS[timelineSource.value ?? "home"];
		case "notifications":
			return "ph-bell ph-bold ph-lg";
		case "widgets":
			return "ph-browser ph-bold ph-lg";
		case "favorites":
			return "ph-star ph-bold ph-lg";
		case "mentions":
			return "ph-at ph-bold ph-lg";
		default:
			return null;
	}
});

/** Static key per kind; building the key dynamically breaks the i18n gate. */
function columnLabel(type: Column["type"]): string {
	switch (type) {
		case "main":
			return t("_deck._columns.main");
		case "tl":
			return t("_deck._columns.tl");
		case "notifications":
			return t("_deck._columns.notifications");
		case "widgets":
			return t("_deck._columns.widgets");
		case "favorites":
			return t("favorites");
		case "antenna":
			return t("_deck._columns.antenna");
		case "list":
			return t("_deck._columns.list");
		case "channel":
			return t("_deck._columns.channel");
		case "mentions":
			return t("_deck._columns.mentions");
		case "direct":
			return t("_deck._columns.direct");
	}
}

function timelineLabel(source: TimelineSource): string {
	switch (source) {
		case "home":
			return t("_timelines.home");
		case "local":
			return t("_timelines.local");
		case "social":
			return t("_timelines.social");
		case "global":
			return t("_timelines.global");
	}
}

const title = computed(() => props.column.name ?? columnLabel(props.column.type));

// `i/favorites` returns NoteFavorite[] (docs/research/api-contract/
// notes-core.md); CfPagination is generic over `{ id: string }` in templates,
// so the slot payload is re-typed here.
const favoritesPagination: Paging = {
	endpoint: "i/favorites",
	limit: 10,
};

const mentionsPagination: Paging = {
	endpoint: "notes/mentions",
	limit: 10,
};

function noteItems(items: readonly { id: string }[]): Note[] {
	return items as Note[];
}

function favoriteItems(items: readonly { id: string }[]): NoteFavorite[] {
	return items as NoteFavorite[];
}

function goTop(): void {
	bodyEl.value?.scrollTo({ top: 0, behavior: "smooth" });
}

function toggleActive(): void {
	if (!props.isStacked) return;
	const expand = props.column.active === false;
	updateColumn(props.column.id, { active: expand });
	if (expand) indicated.value = false;
}

function onQueue(count: number): void {
	if (!active.value) indicated.value = count !== 0;
}

function onNote(): void {
	if (!active.value) indicated.value = true;
}

/** 350 / 420 px, the two widths the deck column header toggles between. */
function toggleWidth(): void {
	updateColumn(props.column.id, {
		width: props.column.width === 420 ? 350 : 420,
	});
}

async function renameColumn(): Promise<void> {
	const { canceled, result } = await os.inputText({
		title: t("_deck.configureColumn"),
		default: props.column.name ?? "",
		minLength: 1,
	});
	if (canceled) return;
	const name = result.trim();
	if (name === "") return;
	updateColumn(props.column.id, { name });
}

function changeTimeline(source: TimelineSource): void {
	updateColumn(props.column.id, { tl: source });
	indicated.value = false;
}

function openNotificationSettings(): void {
	os.popup(
		defineAsyncComponent(
			() => import("@/components/MkNotificationSettingWindow.vue"),
		),
		{ includingTypes: props.column.includingTypes ?? null },
		{
			done: (result: { includingTypes: string[] | null }) => {
				updateColumn(props.column.id, {
					includingTypes:
						result.includingTypes == null ||
						result.includingTypes.length === 0
							? undefined
							: (result.includingTypes as Column["includingTypes"]),
				});
			},
		},
		"closed",
	);
}

function getMenu(): MenuItem[] {
	const items: MenuItem[] = [
		{
			icon: "ph-arrow-up ph-bold",
			text: t("_menuDisplay.top"),
			action: goTop,
		},
		{
			icon: "ph-arrows-horizontal ph-bold",
			text: t("width"),
			caption: `${props.column.width}px → ${
				props.column.width === 420 ? 350 : 420
			}px`,
			action: toggleWidth,
		},
		{
			icon: "ph-pencil ph-bold",
			text: t("_deck.configureColumn"),
			action: () => void renameColumn(),
		},
		{ separator: true },
	];

	if (props.column.type === "tl") {
		items.push(
			{ type: "label", text: t("timeline") },
			...TIMELINE_SOURCES.map((source) => ({
				icon: TIMELINE_ICONS[source],
				text: timelineLabel(source),
				active: props.column.tl === source,
				action: () => changeTimeline(source),
			})),
			{ separator: true },
		);
	}

	if (props.column.type === "notifications") {
		items.push(
			{
				icon: "ph-pencil ph-bold",
				text: t("notificationSetting"),
				action: openNotificationSettings,
			},
			{ separator: true },
		);
	}

	items.push(
		{
			icon: "ph-caret-left ph-bold",
			text: t("_deck.swapLeft"),
			action: () => swapLeftColumn(props.column.id),
		},
		{
			icon: "ph-caret-right ph-bold",
			text: t("_deck.swapRight"),
			action: () => swapRightColumn(props.column.id),
		},
	);

	if (props.isStacked) {
		items.push(
			{
				icon: "ph-caret-up ph-bold",
				text: t("_deck.swapUp"),
				action: () => swapUpColumn(props.column.id),
			},
			{
				icon: "ph-caret-down ph-bold",
				text: t("_deck.swapDown"),
				action: () => swapDownColumn(props.column.id),
			},
		);
	}

	items.push({
		icon: "ph-copy ph-bold",
		text: t("_deck.stackLeft"),
		action: () => stackLeftColumn(props.column.id),
	});

	if (props.isStacked) {
		items.push({
			icon: "ph-browser ph-bold",
			text: t("_deck.popRight"),
			action: () => popRightColumn(props.column.id),
		});
	}

	items.push(
		{ separator: true },
		{
			icon: "ph-trash ph-bold",
			text: t("remove"),
			danger: true,
			action: () => removeColumn(props.column.id),
		},
	);

	return items;
}

function openMenu(ev: MouseEvent): void {
	void os.popupMenu(getMenu(), ev);
}
</script>

<style lang="scss" scoped>
$header-height: 42px;

.deck-column {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	min-block-size: 0;
	background: var(--panel);
	border-radius: var(--radius);
	overflow: clip;

	&.stacked.collapsed {
		block-size: $header-height;
		min-block-size: $header-height;
	}

	> .header {
		position: relative;
		display: flex;
		align-items: center;
		flex: 0 0 $header-height;
		gap: 2px;
		padding: 0 6px;
		block-size: $header-height;
		box-sizing: border-box;
		color: var(--panelHeaderFg);
		background: var(--panelHeaderBg);
		box-shadow: 0 1px 0 0 var(--panelHeaderDivider);
		cursor: pointer;
		user-select: none;

		&.indicated {
			box-shadow: inset 0 -3px 0 var(--accent);
		}

		> .title {
			display: flex;
			align-items: center;
			flex: 1;
			gap: 8px;
			min-inline-size: 0;
			font-size: 0.9em;
			font-weight: 600;

			> .name {
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}
		}

		> .icon-button {
			display: flex;
			align-items: center;
			justify-content: center;
			flex: 0 0 32px;
			block-size: 32px;
			border-radius: 8px;
			color: var(--fgTransparentWeak);

			&:hover {
				background: var(--buttonBg);
				color: var(--fg);
			}
		}
	}

	> .body {
		flex: 1;
		min-block-size: 0;
		overflow-y: auto;
		overflow-x: clip;
		overscroll-behavior: contain;

		> .favorites {
			padding-block: 4px;
		}

		> .unsupported {
			padding: 16px;
		}
	}
}

.unsupported-title {
	margin-block-end: 6px;
	font-weight: 700;
}
</style>
