<!-- Ported from firefish@b425f61:packages/client/src/pages/timeline.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<!-- Firefish uses v-model:tab here; the explicit prop/event pair keeps
			     the store-backed source typed as TimelineSource. -->
			<MkPageHeader
				:tab="src"
				:actions="headerActions"
				:tabs="headerTabs"
				:display-my-avatar="true"
				@update:tab="onTabUpdate"
			/>
		</template>
		<MkSpacer :content-max="800">
			<div class="cmuxhskf">
				<MkPostForm
					v-if="showFixedPostForm"
					class="post-form _block"
					fixed
				/>
				<div class="tl _block">
					<MkTimeline :key="src" :src="src" :sound="true" />
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's timeline page branches into swiper/tutorial/antenna/list menus and
// a fixed PostForm. Cinderfish drops the swiper, tutorial and antennas (no
// endpoints) and keeps the user-list menu as a header action:
//   * no swiper — only the selected tab's MkTimeline is rendered;
//   * no antenna or recommended timeline (no endpoints);
//   * the fixed PostForm is ported as-is (top of the column, gated by
//     defaultStore.state.showFixedPostForm). Like Firefish, `posted` is not
//     handled here: MkTimeline prepends the new note from Streaming.
// `src` stays persisted in defaultStore.state.tl, exactly like Firefish; a
// list timeline is reached through /timeline/list/:listId and never persisted.
import { computed } from "vue";
import MkTimeline from "@/components/MkTimeline.vue";
import MkPostForm from "@/components/MkPostForm.vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { t } from "@/i18n";
import { instance } from "@/instance";
import { me } from "@/account";
import * as os from "@/os";
import type { MenuItem } from "@/os";
import { router } from "@/router";
import { defaultStore, type TimelineSource } from "@/store/base";
import { definePageMetadata } from "@/scripts/page-metadata";

type TabDefinition = {
	key: TimelineSource;
	title: string;
	icon: string;
	iconOnly: boolean;
};

// meta.features.localTimeline is always true on Cinderfish; globalTimeline
// reflects the configured federation mode (docs/research/api-contract/
// meta-federation.md). Admins/moderators keep the tab even when federation is
// off, as in Firefish. Both are read through computed refs because `instance`
// is filled asynchronously by the first meta response.
const isLocalTimelineAvailable = computed(
	() => instance.features?.localTimeline !== false,
);
const isGlobalTimelineAvailable = computed(
	() =>
		instance.features?.globalTimeline !== false ||
		me.value?.isAdmin === true ||
		me.value?.isModerator === true,
);

const headerTabs = computed<TabDefinition[]>(() => [
	{
		key: "home",
		title: t("_timelines.home"),
		icon: "ph-house ph-bold ph-lg",
		iconOnly: true,
	},
	...(isLocalTimelineAvailable.value
		? [
				{
					key: "local" as const,
					title: t("_timelines.local"),
					icon: "ph-users ph-bold ph-lg",
					iconOnly: true,
				},
			]
		: []),
	...(isLocalTimelineAvailable.value
		? [
				{
					key: "social" as const,
					title: t("_timelines.social"),
					icon: "ph-handshake ph-bold ph-lg",
					iconOnly: true,
				},
			]
		: []),
	...(isGlobalTimelineAvailable.value
		? [
				{
					key: "global" as const,
					title: t("_timelines.global"),
					icon: "ph-planet ph-bold ph-lg",
					iconOnly: true,
				},
			]
		: []),
]);

const availableSources = computed<TimelineSource[]>(() =>
	headerTabs.value.map((tab) => tab.key),
);

const src = computed<TimelineSource>(() => {
	const stored = defaultStore.reactiveState.tl.value.src;
	return availableSources.value.includes(stored) ? stored : "home";
});

const showFixedPostForm = computed(
	() => defaultStore.reactiveState.showFixedPostForm.value,
);

function onTabUpdate(key: string): void {
	const next = key as TimelineSource;
	if (!availableSources.value.includes(next)) return;
	if (next === src.value) return;
	saveSrc(next);
}

function saveSrc(newSrc: TimelineSource): void {
	defaultStore.set("tl", {
		...defaultStore.state.tl,
		src: newSrc,
	});
}

// Firefish's list menu: the caller's lists navigate to /timeline/list/:listId,
// with a trailing entry for the list manager. Loading is deferred until the
// action is pressed so an unused menu costs no request.
async function openListsMenu(ev: MouseEvent): Promise<void> {
	const items: MenuItem[] = [];
	try {
		const lists = await api(ENDPOINTS.usersListsList, {});
		if (lists.length === 0) {
			items.push({ type: "label", text: t("noLists") });
		} else {
			for (const list of lists) {
				items.push({
					icon: "ph-list-bullets ph-bold ph-lg",
					text: list.name,
					action: () => {
						void router.push(`/timeline/list/${list.id}`);
					},
				});
			}
		}
	} catch {
		items.push({ type: "label", text: t("somethingHappened") });
	}
	items.push(
		{ separator: true },
		{
			icon: "ph-sliders-horizontal ph-bold ph-lg",
			text: t("manageLists"),
			action: () => {
				void router.push("/my/lists");
			},
		},
	);
	void os.popupMenu(items, ev);
}

const headerActions = computed<
	{
		text: string;
		icon: string;
		handler: (ev: MouseEvent) => void;
	}[]
>(() =>
	me.value == null
		? []
		: [
				{
					text: t("lists"),
					icon: "ph-list-bullets ph-bold ph-lg",
					handler: (ev: MouseEvent) => {
						void openListsMenu(ev);
					},
				},
			],
);

function iconForSource(source: TimelineSource): string {
	switch (source) {
		case "local":
			return "ph-users ph-bold ph-lg";
		case "social":
			return "ph-handshake ph-bold ph-lg";
		case "global":
			return "ph-planet ph-bold ph-lg";
		case "home":
		default:
			return "ph-house ph-bold ph-lg";
	}
}

definePageMetadata(
	computed(() => ({
		title: t("timeline"),
		icon: iconForSource(src.value),
	})),
);
</script>

<style lang="scss" scoped>
.cmuxhskf {
	--swiper-theme-color: var(--accent);
	> .tl {
		background: none;
	}
}
</style>
