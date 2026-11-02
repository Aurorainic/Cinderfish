// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/timeline.vue
//
// Adaptations:
// - Firefish's MkPageHeader is PageHeader (ui/PageHeader.vue) with the same
//   v-model:tab contract; the swiper is dropped and tab switching remounts
//   the feed component instead of sliding.
// - The basic/home/local/social/global union is reduced to what the Cinderfish
//   api client serves (home/local/hybrid/global); tab labels come from
//   i18n.ts._timelines. The "recommended" and channel/antenna/list sources
//   have no backend counterpart (TODO when they exist).
// - Notes render through MkTimeline (TimelineFeed.vue) fed by api fetchers.
<template>
	<PageHeader v-model:tab="tab" :title="i18n.ts.timeline" :tabs="tabs" />
	<TimelineFeed :source="source" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import TimelineFeed from "../components/TimelineFeed.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import { usePageTitle } from "../composables/use-page-title";
import { i18n } from "../i18n";
import { isLoggedIn } from "../stores/session";

usePageTitle(i18n.ts.timeline);

type Source = "home" | "local" | "hybrid" | "global";

// Misskey channel names: home reads as 首页, hybrid as 社交. Deck columns use
// the same labels (stores/deck.ts), so both shells name a channel alike — the
// icons are shared with the deck column definitions for the same reason.
const CHANNELS: readonly Source[] = ["home", "local", "hybrid", "global"];
// Local mirror of PageHeader's tab shape (the interface is not exported).
type Tab = { key: string; label: string; icon?: "home" | "globe" | "feather" | "compass" };

const TAB_META: Record<Source, { label: string; icon: Tab["icon"] }> = {
	home: { label: i18n.ts._timelines.home, icon: "home" },
	local: { label: i18n.ts._timelines.local, icon: "globe" },
	hybrid: { label: i18n.ts._timelines.social, icon: "feather" },
	global: { label: i18n.ts._timelines.global, icon: "compass" },
};

const AUTH_CHANNELS: readonly string[] = ["home", "hybrid"];

function isChannel(value: string): value is Source {
	return (CHANNELS as readonly string[]).includes(value);
}

function isAvailable(value: string): value is Source {
	return isChannel(value) && (isLoggedIn.value || !AUTH_CHANNELS.includes(value));
}

const tabs = computed(() =>
	CHANNELS.filter((key) => isAvailable(key)).map((key) => ({
		key,
		label: TAB_META[key].label,
		icon: TAB_META[key].icon,
	})),
);

// The route ?tab= handling of the original is kept: the selected channel
// survives reload and sharing, and falls back to 首页 when signed in and
// 本地 otherwise.
const tab = defineModel<string>("tab", { default: "" });

const source = computed<Source>(() =>
	isAvailable(tab.value) ? (tab.value as Source) : isLoggedIn.value ? "home" : "local",
);

if (!tab.value) tab.value = source.value;
</script>
