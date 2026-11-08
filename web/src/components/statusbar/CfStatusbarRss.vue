<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/statusbar-rss.vue (AGPL-3.0-only) -->
<template>
	<span class="cf-statusbar-rss">
		<span v-if="fetching" class="state">读取中…</span>
		<span v-else-if="failed" class="state" :title="errorText">
			无法读取订阅源
		</span>
		<span v-else-if="items.length === 0" class="state">
			订阅源没有条目
		</span>
		<MkMarquee
			v-else
			class="track"
			:duration="marqueeDuration"
			:reverse="marqueeReverse"
		>
			<span
				v-for="(item, index) in items"
				:key="`${item.link}-${index}`"
				class="item"
			>
				<a
					class="link"
					:href="item.link"
					rel="nofollow noopener"
					target="_blank"
					:title="item.title"
					>{{ item.title }}</a
				>
				<span class="divider"></span>
			</span>
		</MkMarquee>
	</span>
</template>

<script lang="ts" setup>
// The feed is fetched through the server (`fetch-rss`) for the same reason the
// rss widget is: browsers cannot read a cross-origin feed, and the endpoint
// applies the SSRF guard.
import { onMounted, ref, watch } from "vue";
import MkMarquee from "@/components/MkMarquee.vue";
import { apiGet } from "@/api/client";
import type { FetchRssFeed, FetchRssItem } from "@/api/types";
import { useInterval } from "@/composables/use-interval";

const props = withDefaults(
	defineProps<{
		url?: string;
		shuffle?: boolean;
		marqueeDuration?: number;
		marqueeReverse?: boolean;
		refreshIntervalSec?: number;
	}>(),
	{
		url: "",
		shuffle: false,
		marqueeDuration: 40,
		marqueeReverse: false,
		refreshIntervalSec: 300,
	},
);

const MAX_ITEMS = 50;

const items = ref<FetchRssItem[]>([]);
const fetching = ref(true);
const failed = ref(false);
const errorText = ref("");

function shuffleInPlace(list: FetchRssItem[]): void {
	for (let i = list.length - 1; i > 0; i--) {
		const j = Math.floor(Math.random() * (i + 1));
		[list[i], list[j]] = [list[j], list[i]];
	}
}

/**
 * A feed can be huge (Codeberg's release feeds carry hundreds of entries) and
 * some generators emit blank titles. A ticker of empty dividers helps nobody,
 * and thousands of nodes in a 26px bar would only make the marquee fly by, so
 * blank titles are dropped and the list is capped.
 */
function selectItems(feed: FetchRssFeed): FetchRssItem[] {
	const titles = (feed.items ?? []).filter(
		(item) => (item.title ?? "").trim() !== "",
	);
	if (props.shuffle) shuffleInPlace(titles);
	return titles.slice(0, MAX_ITEMS);
}

function load(): void {
	const url = props.url.trim();
	if (url === "") {
		items.value = [];
		fetching.value = false;
		failed.value = false;
		return;
	}
	fetching.value = true;
	failed.value = false;
	apiGet<FetchRssFeed>("fetch-rss", { url })
		.then((feed) => {
			items.value = selectItems(feed);
		})
		.catch((error: unknown) => {
			failed.value = true;
			errorText.value =
				error instanceof Error && error.message !== ""
					? error.message
					: "无法读取订阅源";
		})
		.finally(() => {
			fetching.value = false;
		});
}

onMounted(load);

useInterval(
	load,
	Math.max(30, props.refreshIntervalSec) * 1000,
);

watch(
	() => props.url,
	() => {
		items.value = [];
		load();
	},
);
</script>

<style lang="scss" scoped>
.cf-statusbar-rss {
	display: flex;
	align-items: center;
	height: 100%;
	overflow: hidden;
	white-space: nowrap;

	> .state {
		padding-inline: 10px;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}

	> .track {
		flex: 1;
		min-width: 0;
	}

	.item {
		display: inline-flex;
		align-items: center;
		height: 100%;

		> .link {
			padding-inline: 8px;
			color: inherit;
			text-decoration: none;

			&:hover {
				text-decoration: underline;
			}
		}

		> .divider {
			inline-size: 1px;
			block-size: 0.9em;
			background: currentColor;
			opacity: 0.4;
		}
	}
}
</style>
