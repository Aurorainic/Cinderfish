// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkTimeline.vue
//
// Adaptations:
// - Firefish builds the pagination config from an endpoint name; the Cinderfish
//   api client exposes typed methods, so the paging config takes a fetcher
//   function over web/src/api/client.ts.
// - MkPullToRefresh and the tlHint MkInfo are dropped: TODO no pull-to-refresh
//   runtime and no timeline hint copy in Cinderfish.
// - The channel filter/antenna/list variants are dropped: the backend only
//   serves home/local/hybrid/global (see the StreamChannelName union in
//   stores/streaming.ts).
// - Streamed notes are prepended as-is; Firefish's withReplies/withRenotes
//   client-side replay checks have no backend counterpart.
<template>
	<div v-if="needsAuth && !isLoggedIn" class="_panel _fullinfo auth-gate">
		<AppIcon class="ghost" name="globe" />
		<h2>需要登录</h2>
		<p>这个频道只对已登录用户开放。</p>
		<MkButton link to="/login" primary>登录</MkButton>
	</div>

	<div v-else class="timeline-feed">
		<!-- The section's own composer. Replies and quotes are opened by
		     NoteFooter itself through os.openPostForm, so there is only ever
		     one post dialog. Deck columns pass `:composer="false"`: a 350px
		     column is not the place for it, and the rail carries the global
		     post entry there. -->
		<PostForm v-if="isLoggedIn && composer && showFixedPostForm" fixed />

		<!-- Firefish's "new notes" button (MkTimeline's `.new`): streamed notes
		     queue up inside MkPagination while the list is scrolled away from
		     the top; the button scrolls back so the queue flushes. -->
		<div v-if="queue > 0" class="new">
			<button type="button" class="new-button" @click="scrollTop">
				{{ queue }} 条新笔记
				<AppIcon name="caretUp" />
			</button>
		</div>

		<MkPagination
			:key="`${source}:${tag}`"
			ref="paginationEl"
			:pagination="pagination"
			@queue="(count) => (queue = count)"
		>
			<template #default="{ items }">
				<MkDateSeparatedList :items="items" no-gap>
					<template #default="{ item }">
						<NoteItem :note="item" />
					</template>
				</MkDateSeparatedList>
			</template>
			<template #empty>
				<div class="_fullinfo empty-placeholder">
					<AppIcon class="ghost" name="feather" />
					<p>这里还没有笔记。</p>
				</div>
			</template>
		</MkPagination>
	</div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api } from "../api/client";
import type { Note, TimelineParams } from "../api/types";
import { isLoggedIn } from "../stores/session";
import { defaultStore } from "../store/firefish-store";
import { useChannel } from "../stores/streaming";
import AppIcon from "./AppIcon.vue";
import MkButton from "./ui/MkButton.vue";
import PostForm from "./composer/PostForm.vue";
import MkDateSeparatedList from "./ui/MkDateSeparatedList.vue";
import MkPagination from "./ui/MkPagination.vue";
import NoteItem from "./note/Note.vue";

const props = withDefaults(
	defineProps<{ source: "local" | "global" | "hybrid" | "home"; tag?: string; composer?: boolean }>(),
	{ tag: "", composer: true },
);

const needsAuth = computed(() => props.source === "hybrid" || props.source === "home");

// Firefish's timeline only pins the post form when the user turns that on
// (deviceStore.showFixedPostForm, default false); the navbar's post button is
// the normal entry point. The registry is reactive, so flipping the setting in
// the widget/settings UI shows or hides the form without a reload.
const showFixedPostForm = computed(() => defaultStore.reactiveState.showFixedPostForm.value);

const PAGE_SIZE = 20;

const timelineApi = {
	local: api.localTimeline,
	global: api.globalTimeline,
	hybrid: api.hybridTimeline,
	home: api.homeTimeline,
};

const streamChannels = {
	local: "localTimeline",
	global: "globalTimeline",
	hybrid: "hybridTimeline",
	home: "homeTimeline",
} as const;

// When a tag filter is active the fetcher applies it client-side; MkPagination
// derives its cursor from the last item it received, which would skip
// unfiltered notes between pages. The fetcher therefore keeps its own raw
// cursor and ignores MkPagination's untilId while filtering. The raw page is
// over-fetched so a heavily filtered page still yields items.
let rawCursor = "";

async function fetcher(params: Record<string, unknown>): Promise<Note[]> {
	const untilId = typeof params.untilId === "string" ? params.untilId : "";
	const request: TimelineParams = { limit: props.tag ? PAGE_SIZE * 4 : PAGE_SIZE };
	if (props.tag) {
		if (!untilId) rawCursor = "";
		request.untilId = rawCursor || undefined;
	} else if (untilId) {
		request.untilId = untilId;
	}
	const page = await timelineApi[props.source](request);
	if (page.length) rawCursor = page[page.length - 1].id;
	if (!props.tag) return page;
	const needle = `#${props.tag}`.toLowerCase();
	return page.filter((n) => (n.text ?? "").toLowerCase().includes(needle));
}

// A fresh config object per key: the template keys MkPagination on
// `${source}:${tag}`, so a change remounts it and init() runs again.
const pagination = computed(() => ({
	endpoint: fetcher,
	limit: PAGE_SIZE,
}));

type PaginationApi = {
	items: Note[];
	prepend: (...items: Note[]) => void;
};

const paginationEl = ref<PaginationApi | null>(null);
const queue = ref(0);

// Streamed notes arrive on the channel matching the source and are prepended
// into MkPagination; the component queues them while the top is off screen
// and flushes on scroll, mirroring Firefish's connection.on("note", ...).
let channel: ReturnType<typeof useChannel> | null = null;

function onStreamNote(body: unknown) {
	if (!body || typeof body !== "object") return;
	paginationEl.value?.prepend(body as Note);
}

function watchSource(source: string) {
	channel?.dispose();
	channel = useChannel(streamChannels[source as keyof typeof streamChannels]);
	channel.on("note", onStreamNote);
}

watch(() => props.source, watchSource);

onMounted(() => watchSource(props.source));

onBeforeUnmount(() => {
	channel?.dispose();
	channel = null;
});

function scrollTop() {
	window.scrollTo({ top: 0, behavior: "smooth" });
}
</script>

<style scoped>
.new {
	position: sticky;
	top: 52px;
	z-index: 5;
	display: flex;
	justify-content: center;
	padding-block: 8px;
}

.new-button {
	display: inline-flex;
	align-items: center;
	gap: 6px;
	padding: 7px 18px;
	border: 1px solid var(--accent);
	border-radius: var(--radius-full);
	background: var(--accent-soft);
	color: var(--accent);
	font: inherit;
	font-size: var(--fs-sm);
	font-weight: 700;
	cursor: pointer;
	transition: background 0.15s ease, color 0.15s ease;
}

.new-button .icon {
	width: 14px;
	height: 14px;
}

.new-button:hover,
.new-button:focus-visible {
	background: var(--accent);
	color: var(--text-on-accent);
}

/* Borderless rows carry their own hairline, so they must not be spaced. */
.timeline-feed :deep(.cinderfish-date-separated-list.noGap) {
	gap: 0;
}

/* Firefish's empty state (MkPagination's _fullinfo slot): the muted badge and
   its label sit centred in the column instead of hugging the top-left. */
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
   its empty state; the panel only appears when the feed has nothing else. */
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
</style>
