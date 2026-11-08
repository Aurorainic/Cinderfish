<!-- Ported from firefish@b425f61:packages/client/src/components/MkTimeline.vue (AGPL-3.0-only) -->
<template>
	<CfInfo v-if="streamUnavailable" :warn="true" class="_gap">
		{{ t("disconnectedFromServer") }}
	</CfInfo>
	<CfInfo
		v-if="hint != null && !hintClosed"
		:closeable="true"
		class="_gap"
		@close="closeHint"
	>
		{{ hint }}
	</CfInfo>
	<div v-if="queueCount > 0" class="new">
		<button
			class="_buttonPrimary _shadow"
			:class="{ instant: !defaultStore.reactiveState.animation.value }"
			@click="onNewNoteButton"
		>
			{{ t("newNoteRecived") }}
			<i class="ph-arrow-up ph-bold"></i>
		</button>
	</div>
	<MkNotes
		ref="tlComponent"
		:key="paginationKey"
		:no-gap="!defaultStore.reactiveState.showGapBetweenNotesInTimeline.value"
		:pagination="pagination"
	/>
</template>

<script setup lang="ts">
// Home / local / social / global / list / antenna / channel timeline.
//
// Firefish branches on `src` (antenna, list, channel, recommended, mentions,
// directs). Cinderfish has no endpoints for most of those (contract §5); the
// four real timelines, the user-list timeline, the antenna timeline and the
// channel timeline are ported. `list`, `antenna` and `channel` are local
// extensions of the prop type, deliberately not part of the persisted
// `TimelineSource` union in store/base.ts, so none of them can become
// someone's saved `tl.src` preference.
//
// Streaming notes are prepended here: when the reader is scrolled down they are
// queued (Firefish's MkPagination behaviour) and flushed when the timeline is
// scrolled back to the top or the "new note" button is pressed. Reaction and
// deletion updates stay with `useNoteCapture` — this component only inserts.
import { computed, onUnmounted, ref } from "vue";
import MkNotes from "@/components/MkNotes.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import type { Paging } from "@/components/ui/pagination";
import { stream } from "@/stream";
import type { StreamChannelName } from "@/api/stream";
import type { NoteDetailed } from "@/api/types";
import * as sound from "@/scripts/sound";
import { getScrollPosition } from "@/scripts/scroll";
import { me } from "@/account";
import { t } from "@/i18n";
import { defaultStore, type TimelineSource } from "@/store/base";

/** The subset of MkNotes' exposed surface this component relies on. */
interface NotesPaging {
	prepend: (note: NoteDetailed) => void;
	reload?: () => void;
}
interface NotesExposed {
	pagingComponent?: NotesPaging | null;
}

/** A persisted timeline source, or the non-persisted list/antenna/channel ones. */
export type TimelineSourceProp =
	| TimelineSource
	| "list"
	| "antenna"
	| "channel";

type HintClosedKey =
	| "tlHomeHintClosed"
	| "tlLocalHintClosed"
	| "tlSocialHintClosed"
	| "tlGlobalHintClosed";

interface TimelineDefinition {
	endpoint: string;
	channel: StreamChannelName;
	/** REST query; `withReplies` is a real backend parameter on the four timelines. */
	restParams: Record<string, unknown>;
	/** Channel subscription params (docs/research/api-contract/streaming.md). */
	channelParams: Record<string, unknown>;
	/** `null` means the timeline has no hint to render. */
	hintKey: string | null;
	/** `null` when `hintKey` is null; no store key is read or written then. */
	hintClosedKey: HintClosedKey | null;
}

const props = withDefaults(
	defineProps<{
		src: TimelineSourceProp;
		/** Required when `src` is `"list"`; ignored otherwise. */
		listId?: string;
		/** Required when `src` is `"antenna"`; ignored otherwise. */
		antennaId?: string;
		/** Required when `src` is `"channel"`; ignored otherwise. */
		channelId?: string;
		sound?: boolean;
	}>(),
	{
		listId: undefined,
		antennaId: undefined,
		channelId: undefined,
		sound: false,
	},
);

const emit = defineEmits<{
	note: [];
	queue: [count: number];
}>();

const showTimelineReplies = defaultStore.reactiveState.showTimelineReplies.value;

// The channel defaults differ from upstream Misskey: only `homeTimeline` ships
// pure renotes by default, the other three drop them unless the client asks for
// them explicitly (contract: streaming.md line ~602). Firefish's timelines show
// renotes, so ask for them.
function definitionFor(
	src: TimelineSourceProp,
	listId?: string,
	antennaId?: string,
	channelId?: string,
): TimelineDefinition {
	switch (src) {
		case "list":
			return {
				endpoint: "notes/user-list-timeline",
				channel: "userList",
				restParams: {
					listId,
					withRenotes: true,
					withFiles: false,
				},
				channelParams: { listId, withRenotes: true },
				hintKey: null,
				hintClosedKey: null,
			};
		case "antenna":
			return {
				endpoint: "antennas/notes",
				channel: "antenna",
				restParams: { antennaId },
				channelParams: { antennaId },
				hintKey: null,
				hintClosedKey: null,
			};
		case "channel":
			return {
				endpoint: "channels/timeline",
				channel: "channel",
				restParams: { channelId },
				channelParams: { channelId },
				hintKey: null,
				hintClosedKey: null,
			};
		case "local":
			return {
				endpoint: "notes/local-timeline",
				channel: "localTimeline",
				restParams: { withReplies: showTimelineReplies },
				channelParams: {
					withReplies: showTimelineReplies,
					withRenotes: true,
				},
				hintKey: "_tutorial.step5_4",
				hintClosedKey: "tlLocalHintClosed",
			};
		case "social":
			return {
				endpoint: "notes/hybrid-timeline",
				channel: "hybridTimeline",
				restParams: { withReplies: showTimelineReplies },
				channelParams: {
					withReplies: showTimelineReplies,
					withRenotes: true,
				},
				hintKey: "_tutorial.step5_5",
				hintClosedKey: "tlSocialHintClosed",
			};
		case "global":
			return {
				endpoint: "notes/global-timeline",
				channel: "globalTimeline",
				restParams: { withReplies: showTimelineReplies },
				// globalTimeline has no `withReplies` parameter.
				channelParams: { withRenotes: true },
				hintKey: "_tutorial.step5_7",
				hintClosedKey: "tlGlobalHintClosed",
			};
		case "home":
		default:
			return {
				endpoint: "notes/timeline",
				channel: "homeTimeline",
				restParams: { withReplies: showTimelineReplies },
				// homeTimeline has no `withReplies` parameter and defaults
				// `withRenotes` to true.
				channelParams: {},
				hintKey: "_tutorial.step5_3",
				hintClosedKey: "tlHomeHintClosed",
			};
	}
}

// `listId` / `antennaId` / `channelId` participate in the definition and in
// the pagination key so a changed list, antenna or channel reloads the
// timeline (and its streaming channel) instead of showing the previous
// source's notes.
const definition = computed(() =>
	definitionFor(props.src, props.listId, props.antennaId, props.channelId),
);

const paginationKey = computed(
	() =>
		`${props.src}:${props.listId ?? ""}:${props.antennaId ?? ""}:${props.channelId ?? ""}`,
);

const pagination = computed<Paging>(() => ({
	endpoint: definition.value.endpoint,
	limit: 10,
	params: definition.value.restParams,
}));

const tlComponent = ref<NotesExposed | null>(null);

// ----- queue --------------------------------------------------------------

/**
 * Notes that arrived from Streaming while the reader was scrolled down. They
 * are handed to the list in arrival order when the timeline is back at the top.
 */
const queue = ref<NoteDetailed[]>([]);
const queueCount = computed(() => queue.value.length);
let scrollFlushHandler: (() => void) | null = null;

function isAtTop(): boolean {
	return getScrollPosition(null) <= 16;
}

function pagingComponent(): NotesPaging | null {
	return tlComponent.value?.pagingComponent ?? null;
}

function isUsableNote(note: NoteDetailed): boolean {
	// Remote notes arrive as partial objects (`{ id, userId }`) when the streaming
	// hub has not hydrated them; contract: streaming.md "已知缺口" 6.
	const candidate = note as Partial<NoteDetailed> | null | undefined;
	return candidate != null && candidate.user != null && candidate.createdAt != null;
}

function appendToTimeline(note: NoteDetailed): void {
	const paging = pagingComponent();
	if (paging == null) return;
	paging.prepend(note);
	emit("note");
}

function drainQueue(): void {
	if (queue.value.length === 0) return;
	const pending = queue.value;
	queue.value = [];
	emit("queue", 0);
	if (scrollFlushHandler != null) {
		window.removeEventListener("scroll", scrollFlushHandler);
		scrollFlushHandler = null;
	}
	for (const note of pending) {
		appendToTimeline(note);
	}
}

function armScrollFlush(): void {
	if (scrollFlushHandler != null) return;
	scrollFlushHandler = () => {
		if (isAtTop()) drainQueue();
	};
	window.addEventListener("scroll", scrollFlushHandler, { passive: true });
}

function onNewNoteButton(): void {
	drainQueue();
	// The wave-1 shell scrolls the window (`#maincontent` is not a scroll
	// container), so MkNotes.scrollTop() would scroll a non-scrolling element.
	window.scrollTo({ top: 0, behavior: "smooth" });
}

// ----- streaming ----------------------------------------------------------

function prepend(note: NoteDetailed): void {
	if (!isUsableNote(note)) return;

	if (props.sound) {
		const isMyNote = me.value != null && note.userId === me.value.id;
		sound.play(isMyNote ? "noteMy" : "note");
	}

	if (isAtTop()) {
		appendToTimeline(note);
		return;
	}

	queue.value.push(note);
	emit("queue", queue.value.length);
	armScrollFlush();
}

const connection = stream.useChannel(
	definition.value.channel,
	definition.value.channelParams,
);
connection.on("note", prepend);

const streamUnavailable = ref(false);
connection.on("connected", () => {
	streamUnavailable.value = false;
});
connection.on("unavailable", () => {
	streamUnavailable.value = true;
});

// Firefish also listens on `main` for follow/unfollow: the home timeline's
// content set changes when the follow graph does, so refetch it. The `main`
// channel is a per-connection singleton shared with other components, so its
// handlers are detached on unmount but the subscription itself is left alone.
let mainConnection: ReturnType<typeof stream.useChannel> | null = null;
function onChangeFollowing(): void {
	pagingComponent()?.reload?.();
}
if (props.src === "home") {
	mainConnection = stream.useChannel("main");
	mainConnection.on("follow", onChangeFollowing);
	mainConnection.on("unfollow", onChangeFollowing);
}

// ----- hint ---------------------------------------------------------------

// A list or antenna timeline has no hint. `hintClosedKey` is null for both, so
// no store entry is read or written and the template's `hint != null` guard
// keeps the banner out of the page.
const hintClosedKey = definition.value.hintClosedKey;
const hintClosed = ref(
	hintClosedKey == null
		? true
		: defaultStore.reactiveState[hintClosedKey].value,
);
// Firefish renders the hint through <i18n> with an inline icon; `{icon}` has no
// component here, so the placeholder is dropped and the text is shown verbatim.
const hint = computed<string | null>(() => {
	const key = definition.value.hintKey;
	if (key == null) return null;
	return t(key).split("{icon}").join("").replace(/\s+/g, " ").trim();
});

function closeHint(): void {
	const key = definition.value.hintClosedKey;
	if (key == null) return;
	hintClosed.value = true;
	defaultStore.set(key, true);
}

onUnmounted(() => {
	connection.dispose();
	mainConnection?.off("follow", onChangeFollowing);
	mainConnection?.off("unfollow", onChangeFollowing);
	if (scrollFlushHandler != null) {
		window.removeEventListener("scroll", scrollFlushHandler);
		scrollFlushHandler = null;
	}
});
</script>
<style lang="scss" scoped>
@keyframes slideUp {
	to {
		transform: translateY(-100%);
		opacity: 0;
	}
}
.new {
	position: sticky;
	display: flex;
	justify-content: center;
	top: calc(var(--stickyTop, 0px) - 60px);
	width: 600px;
	max-width: 100%;
	height: 60px;
	pointer-events: none;
	margin: auto;
	margin-top: -60px;
	z-index: 1001;
	box-shadow: 0 24px 24px -20px var(--accentedBg);
	&::after {
		content: "";
		position: absolute;
		inset: -2px 0;
		border: 2px solid var(--accentDarken);
		mask: linear-gradient(
			to right,
			transparent,
			black 40% 60%,
			transparent
		);
		-webkit-mask: linear-gradient(
			to right,
			transparent,
			black 40% 60%,
			transparent
		);
	}
	> button {
		display: flex;
		position: absolute;
		top: 120%;
		margin-inline: auto;
		border-radius: 2em;
		padding: 0.5em 1.2em;
		background: var(--accentedBg);
		border: 0;
		color: var(--accent);
		overflow: hidden;
		pointer-events: all;
		transform: translateY(-100%);
		opacity: 0;
		animation:
			reset 0.4s forwards cubic-bezier(0, 0.4, 0, 1.1),
			slideUp 1s 5s forwards cubic-bezier(1, 0, 1, 1);
		&::before {
			content: "";
			position: absolute;
			inset: 0;
			background: var(--bg);
			z-index: -1;
		}
		i {
			margin-left: 0.7em;
			border-left: 1px solid var(--accentedBg);
			padding-left: 0.4em;
		}
	}
}
</style>
