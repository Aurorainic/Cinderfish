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
				<div class="tl _block">
					<MkTimeline :key="src" :src="src" :sound="true" />
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's timeline page branches into swiper/tutorial/antenna/list menus and
// a fixed PostForm. Cinderfish has none of those in this wave (contract §5):
//   * no swiper — only the selected tab's MkTimeline is rendered;
//   * no tutorial dialog, no list/antenna pickers (no endpoints);
//   * no recommended timeline (no endpoint);
//   * no PostForm — the fixed form block is omitted entirely rather than stubbed.
// `src` stays persisted in defaultStore.state.tl, exactly like Firefish.
import { computed } from "vue";
import MkTimeline from "@/components/MkTimeline.vue";
import { t } from "@/i18n";
import { instance } from "@/instance";
import { me } from "@/account";
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

// Lists and antennas have no backend endpoints in this wave, so Firefish's two
// header actions are not rendered. The array is kept for the page structure.
const headerActions = computed<
	{ text: string; icon: string; handler: () => void }[]
>(() => []);

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
