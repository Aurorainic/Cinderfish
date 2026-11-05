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
		:no-gap="!defaultStore.reactiveState.showGapBetweenNotesInTimeline.value"
		:pagination="pagination"
	/>
</template>

<script setup lang="ts">
// Home / local / social / global timeline.
//
// Firefish branches on `src` (antenna, list, channel, recommended, mentions,
// directs). Cinderfish has no endpoints for those (contract §5), so only the
// four real timelines are ported and `src` is limited to them.
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

interface TimelineDefinition {
	endpoint: string;
	channel: "homeTimeline" | "localTimeline" | "hybridTimeline" | "globalTimeline";
	/** REST query; `withReplies` is a real backend parameter on all four. */
	restParams: Record<string, unknown>;
	/** Channel subscription params (docs/research/api-contract/streaming.md). */
	channelParams: Record<string, unknown>;
	hintKey: string | null;
	hintClosedKey:
		| "tlHomeHintClosed"
		| "tlLocalHintClosed"
		| "tlSocialHintClosed"
		| "tlGlobalHintClosed";
}

const props = withDefaults(
	defineProps<{
		src: TimelineSource;
		sound?: boolean;
	}>(),
	{
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
function definitionFor(src: TimelineSource): TimelineDefinition {
	switch (src) {
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

const definition = definitionFor(props.src);

const pagination: Paging = {
	endpoint: definition.endpoint,
	limit: 10,
	params: definition.restParams,
};

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
	definition.channel,
	definition.channelParams,
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

const hintClosed = ref(
	defaultStore.reactiveState[definition.hintClosedKey].value,
);
// Firefish renders the hint through <i18n> with an inline icon; `{icon}` has no
// component here, so the placeholder is dropped and the text is shown verbatim.
const hint = computed<string | null>(() => {
	if (definition.hintKey == null) return null;
	return t(definition.hintKey).split("{icon}").join("").replace(/\s+/g, " ").trim();
});

function closeHint(): void {
	hintClosed.value = true;
	defaultStore.set(definition.hintClosedKey, true);
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
