<!-- Ported from firefish@b425f61:packages/client/src/ui/deck.vue (AGPL-3.0-only) -->
<template>
	<div class="deck" :class="{ mobile: isMobile }">
		<XSidebar v-if="!isMobile" />

		<div class="main">
			<div v-if="phase === 'loading'" class="state">
				<CfLoading />
			</div>

			<div v-else-if="phase === 'error'" class="state">
				<CfInfo :warn="true">{{ t("somethingHappened") }}</CfInfo>
				<CfButton class="retry" @click="reload">{{ t("retry") }}</CfButton>
			</div>

			<div
				v-else
				ref="columnsEl"
				class="columns"
				:class="align"
				@contextmenu.self.prevent="openAddMenu"
				@wheel.self="onWheel"
			>
				<section
					v-for="(group, index) in groups"
					:key="index"
					class="column"
					:class="{ folder: group.columns.length > 1 }"
					:style="groupStyle(group)"
				>
					<DeckColumn
						v-for="column in group.columns"
						:key="column.id"
						:column="column"
						:is-stacked="group.columns.length > 1"
					/>
				</section>

				<section
					v-if="!hasMainColumn"
					class="column"
					:style="{ '--column-width': '350px' }"
				>
					<DeckColumn :column="fallbackMain" virtual />
				</section>

				<div v-if="groups.length === 0" class="intro _panel">
					<div>{{ t("_deck.introduction") }}</div>
					<CfButton primary class="add" @click="openAddMenu">{{
						t("_deck.addColumn")
					}}</CfButton>
					<div>{{ t("_deck.introduction2") }}</div>
				</div>

				<div class="side-menu">
					<div class="top">
						<button
							class="side-button _button"
							:aria-label="t('_deck.profile')"
							@click="changeProfile"
						>
							<i class="ph-caret-down ph-bold ph-lg"></i>
						</button>
						<button
							v-if="profile !== 'default'"
							class="side-button _button"
							:aria-label="t('_deck.renameProfile')"
							@click="renameProfile"
						>
							<i class="ph-pencil ph-bold ph-lg"></i>
						</button>
						<button
							v-if="profile !== 'default'"
							class="side-button _button"
							:aria-label="t('_deck.deleteProfile')"
							@click="deleteProfile"
						>
							<i class="ph-trash ph-bold ph-lg"></i>
						</button>
					</div>
					<div class="middle">
						<button
							class="side-button new _button"
							:aria-label="t('_deck.addColumn')"
							@click="openAddMenu"
						>
							<i class="ph-plus ph-bold ph-lg"></i>
						</button>
					</div>
					<div class="bottom">
						<button
							class="side-button _button"
							:aria-label="t('settings')"
							@click="openSettings"
						>
							<i class="ph-gear-six ph-bold ph-lg"></i>
						</button>
					</div>
				</div>
			</div>
		</div>

		<nav v-if="isMobile" class="mobile-buttons">
			<RouterLink
				to="/"
				class="mobile-button"
				active-class="active"
				:aria-label="t('home')"
			>
				<i class="ph-house ph-bold ph-lg"></i>
			</RouterLink>
			<RouterLink
				to="/my/notifications"
				class="mobile-button"
				active-class="active"
				:aria-label="t('notifications')"
			>
				<i class="ph-bell ph-bold ph-lg"></i>
			</RouterLink>
			<button
				class="mobile-button _button"
				:aria-label="t('note')"
				@click="compose"
			>
				<i class="ph-pencil ph-bold ph-lg"></i>
			</button>
			<RouterLink
				to="/settings/deck"
				class="mobile-button"
				active-class="active"
				:aria-label="t('settings')"
			>
				<i class="ph-gear-six ph-bold ph-lg"></i>
			</RouterLink>
		</nav>
	</div>
</template>

<script setup lang="ts">
// Deck shell surface: a horizontally scrolling row of columns plus the profile
// / add-column controls Firefish keeps in `ui/deck.vue`'s right-hand side menu.
//
// DEGRADED (contract §2): column drag-and-drop reordering is not ported (menus
// move columns instead); `os.select` / `os.form` do not exist in Cinderfish, so
// the timeline source is picked from the column menu and the column "form" is
// reduced to renaming.
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { CfButton, CfInfo, CfLoading } from "@/components/ui";
import { t } from "@/i18n";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { useLayout } from "@/router/layout";
import {
	addColumn as addColumnToStore,
	createDeckId,
	deckStore,
	deleteProfile as deleteProfileRemote,
	getProfiles,
	loadDeck,
	renameProfile as renameProfileRemote,
	type Column,
} from "@/store/deck";
import DeckColumn from "./DeckColumn.vue";
import XSidebar from "./XSidebar.vue";

interface ColumnGroup {
	ids: string[];
	columns: Column[];
}

type Phase = "loading" | "ready" | "error";

const { isMobile } = useLayout();
const router = useRouter();

const phase = ref<Phase>("loading");
const columnsEl = ref<HTMLElement | null>(null);

const columns = computed(() => deckStore.reactiveState.columns.value);
const layout = computed(() => deckStore.reactiveState.layout.value);
const align = computed(() => deckStore.reactiveState.columnAlign.value);
const profile = computed(() => deckStore.reactiveState.profile.value);

/** Layout ids resolved to columns; ids whose column vanished are dropped. */
const groups = computed<ColumnGroup[]>(() =>
	layout.value
		.map((ids) => ({
			ids,
			columns: ids
				.map((id) => columns.value.find((column) => column.id === id))
				.filter((column): column is Column => column != null),
		}))
		.filter((group) => group.columns.length > 0),
);

// Without a "main" column a route would have nowhere to render, so the layout
// appends an unpersisted main column (Firefish opens a page window instead).
const hasMainColumn = computed(() =>
	groups.value.some((group) =>
		group.columns.some((column) => column.type === "main"),
	),
);

const fallbackMain: Column = {
	id: "deck-fallback-main",
	type: "main",
	name: null,
	width: 350,
};

const DEFAULT_WIDTH = 350;

function groupStyle(group: ColumnGroup): Record<string, string> {
	if (group.columns.some((column) => column.flexible)) {
		return { flex: "1", "--column-width": `${DEFAULT_WIDTH}px` };
	}
	const width = Math.max(
		DEFAULT_WIDTH,
		...group.columns.map((column) => column.width),
	);
	return { "--column-width": `${width}px` };
}

async function reload(): Promise<void> {
	phase.value = "loading";
	try {
		await loadDeck();
		phase.value = "ready";
	} catch (error) {
		console.error(error);
		phase.value = "error";
	}
}

onMounted(() => {
	void reload();
});

/** Firefish's wheel-by-wheel horizontal scroll on the column area. */
function onWheel(ev: WheelEvent): void {
	const element = columnsEl.value;
	if (element == null || ev.deltaX !== 0) return;
	element.scrollLeft += ev.deltaY;
}

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
	}
	return type;
}

function addColumnOfType(type: Column["type"]): void {
	const column: Column = {
		id: createDeckId(),
		type,
		name: columnLabel(type),
		width: DEFAULT_WIDTH,
	};
	if (type === "tl") column.tl = "home";
	addColumnToStore(column);
}

function openAddMenu(ev: MouseEvent): void {
	const types: Column["type"][] = [
		"main",
		"tl",
		"notifications",
		"widgets",
		"favorites",
	];
	const icons: Partial<Record<Column["type"], string>> = {
		main: "ph-columns ph-bold",
		tl: "ph-house ph-bold",
		notifications: "ph-bell ph-bold",
		widgets: "ph-browser ph-bold",
		favorites: "ph-star ph-bold",
	};
	const items: MenuItem[] = [
		{ type: "label", text: t("_deck.addColumn") },
		...types.map((type) => ({
			icon: icons[type],
			text: columnLabel(type),
			action: () => addColumnOfType(type),
		})),
	];
	void os.popupMenu(items, ev);
}

function openSettings(): void {
	void router.push("/settings/deck");
}

function compose(): void {
	void os.post();
}

async function switchProfile(name: string): Promise<void> {
	deckStore.set("profile", name);
	await reload();
}

async function changeProfile(ev: MouseEvent): Promise<void> {
	let profiles: string[] = [];
	try {
		profiles = await getProfiles();
	} catch (error) {
		console.error(error);
	}
	const items: MenuItem[] = [
		{ type: "label", text: t("_deck.profile") },
		{ text: profile.value, active: true },
		...profiles
			.filter((name) => name !== profile.value)
			.map((name) => ({
				text: name,
				action: () => void switchProfile(name),
			})),
		{ separator: true },
		{
			icon: "ph-plus ph-bold",
			text: t("_deck.newProfile"),
			action: () => void createProfile(),
		},
	];
	void os.popupMenu(items, ev);
}

async function createProfile(): Promise<void> {
	const { canceled, result } = await os.inputText({
		title: t("_deck.newProfile"),
		minLength: 1,
	});
	if (canceled) return;
	const name = result.trim();
	if (name === "") return;
	await switchProfile(name);
}

async function renameProfile(): Promise<void> {
	const { canceled, result } = await os.inputText({
		title: t("_deck.renameProfile"),
		default: profile.value,
		minLength: 1,
	});
	if (canceled) return;
	const name = result.trim();
	if (name === "" || name === profile.value) return;
	try {
		const profiles = await getProfiles();
		if (profiles.includes(name)) {
			void os.alert({ type: "error", text: t("_deck.nameAlreadyExists") });
			return;
		}
		await renameProfileRemote(profile.value, name);
	} catch (error) {
		console.error(error);
		void os.alert({ type: "error" });
		return;
	}
	await reload();
}

async function deleteProfile(): Promise<void> {
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteAreYouSure", { x: profile.value }),
	});
	if (canceled) return;
	try {
		await deleteProfileRemote(profile.value);
	} catch (error) {
		console.error(error);
	}
	deckStore.set("profile", "default");
	await reload();
}
</script>

<style lang="scss" scoped>
$side-menu-width: 44px;
$mobile-bar-height: 68px;

.deck {
	display: flex;
	block-size: 100dvh;
	overflow: hidden;
	gap: var(--margin);
	padding: var(--margin);
	box-sizing: border-box;
	background: var(--bg);

	&.mobile {
		padding: 0;
		gap: 0;
		padding-block-end: var(--margin);
	}

	> .main {
		flex: 1;
		min-inline-size: 0;
		display: flex;
		flex-direction: column;
	}

	> .main > .state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		margin: auto;
		padding: 32px;
	}

	> .main > .columns {
		flex: 1;
		display: flex;
		gap: 8px;
		min-block-size: 0;
		overflow-x: auto;
		overflow-y: clip;
		scrollbar-width: thin;

		&.center {
			> .column:first-of-type {
				margin-inline-start: auto;
			}

			> .column:last-of-type {
				margin-inline-end: auto;
			}
		}

		&.right {
			justify-content: flex-end;
		}

		> .column {
			flex: 0 0 auto;
			inline-size: var(--column-width, 350px);
			min-block-size: 0;
			max-block-size: 100%;

			&.folder {
				display: flex;
				flex-direction: column;
				gap: 8px;

				> .deck-column {
					min-block-size: 0;

					&:not(.collapsed) {
						flex: 1;
					}
				}
			}
		}

		> .intro {
			margin: auto;
			padding: 32px;
			max-inline-size: 420px;
			text-align: center;
			color: var(--fgTransparentWeak);

			> .add {
				margin: 1em auto;
			}
		}
	}
}

.side-menu {
	display: flex;
	flex: 0 0 $side-menu-width;
	flex-direction: column;
	inline-size: $side-menu-width;

	> .top,
	> .middle,
	> .bottom {
		display: flex;
		flex-direction: column;
	}

	> .top {
		margin-block-end: auto;
	}

	> .middle {
		margin-block: auto;

		> .new {
			color: var(--fgOnAccent);
			background: var(--accent);
			border-radius: 10px;
		}
	}

	> .bottom {
		margin-block-start: auto;
	}
}

.side-button {
	display: flex;
	align-items: center;
	justify-content: center;
	inline-size: 100%;
	aspect-ratio: 1;
	border-radius: 10px;
	color: var(--fgTransparentWeak);

	&:hover {
		color: var(--fg);
		background: var(--buttonBg);
	}
}

.mobile-buttons {
	position: fixed;
	z-index: 1000;
	inset-block-end: 0;
	inset-inline: 0;
	display: flex;
	gap: 8px;
	padding: 8px 12px calc(8px + env(safe-area-inset-bottom));
	box-sizing: border-box;
	block-size: $mobile-bar-height;
	background: var(--header);
	-webkit-backdrop-filter: var(--blur);
	backdrop-filter: var(--blur);
	border-block-start: solid 0.5px var(--divider);

	> .mobile-button {
		display: flex;
		flex: 1;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		block-size: 100%;
		color: var(--fg);
		background: var(--panel);

		&.active {
			color: var(--accent);
		}

		&:active {
			background: var(--buttonBg);
		}
	}
}

// Narrow screens show one column at a time and swipe horizontally between
// them (Firefish's deck does the same through its horizontally scrolling row).
@media (max-width: 700px) {
	.deck > .main > .columns {
		scroll-snap-type: x mandatory;
		padding-inline: var(--margin);
		padding-block-end: $mobile-bar-height;

		> .column {
			inline-size: calc(100vw - var(--margin) * 2);
			scroll-snap-align: start;
		}
	}
}
</style>
