<!-- Ported from firefish@b425f61:packages/client/src/components/MkSubNoteContent.vue (AGPL-3.0-only) -->
<template>
	<p v-if="note.cw != null" class="cw">
		<MkA
			v-if="conversation && note.renoteId == parentId"
			:to="
				detailedView ? `#${parentId}` : `${notePage(note)}#${parentId}`
			"
			behavior="browser"
			class="reply-icon"
			@click.stop
		>
			<i class="ph-quotes ph-bold ph-lg"></i>
		</MkA>
		<MkA
			v-else-if="!detailed && note.replyId"
			:to="
				detailedView
					? `#${note.replyId}`
					: `${notePage(note)}#${note.replyId}`
			"
			behavior="browser"
			v-tooltip="t('jumpToPrevious')"
			class="reply-icon"
			@click.stop
		>
			<i class="ph-arrow-bend-left-up ph-bold ph-lg"></i>
		</MkA>
		<Mfm
			v-if="note.cw != ''"
			class="text"
			:text="note.cw"
			:author="note.user"
			:custom-emojis="note.emojis"
		/>
	</p>
	<div class="wrmlmaau">
		<div
			class="content"
			:class="{
				collapsed,
				isLong,
				manyImages: note.files.length > 4,
				showContent: note.cw && !showContent,
				animatedMfm: !disableMfm,
			}"
		>
			<button
				ref="showMoreButton"
				v-if="isLong && collapsed"
				class="_button cf-show-more is-fade"
				@click.stop="collapsed = false"
				v-on:keydown="focusFooter"
			>
				<span>{{ t("showMore") }}</span>
			</button>
			<MkCwButton
				ref="cwButton"
				v-if="note.cw && !showContent"
				v-model="showContent"
				:note="note"
				v-on:keydown="focusFooter"
				v-on:update:model-value="(val) => emit('expanded', val)"
			/>
			<div
				class="body"
				v-bind="{
					'aria-hidden': note.cw && !showContent ? 'true' : undefined,
					tabindex: !showContent ? '-1' : undefined,
				}"
			>
				<template v-if="!note.cw">
					<MkA
						v-if="conversation && note.renoteId == parentId"
						:to="
							detailedView
								? `#${parentId}`
								: `${notePage(note)}#${parentId}`
						"
						behavior="browser"
						class="reply-icon"
						@click.stop
					>
						<i class="ph-quotes ph-bold ph-lg"></i>
					</MkA>
					<MkA
						v-else-if="!detailed && note.replyId"
						:to="
							detailedView
								? `#${note.replyId}`
								: `${notePage(note)}#${note.replyId}`
						"
						behavior="browser"
						v-tooltip="t('jumpToPrevious')"
						class="reply-icon"
						@click.stop
					>
						<i class="ph-arrow-bend-left-up ph-bold ph-lg"></i>
					</MkA>
				</template>
				<Mfm
					v-if="note.text"
					:text="note.text"
					:author="note.user"
					:custom-emojis="note.emojis"
				/>
				<MkA
					v-if="!detailed && note.renoteId"
					class="rp"
					:to="`/notes/${note.renoteId}`"
					>{{ t("quoteAttached") }}: ...</MkA
				>
				<MkMediaList
					v-if="note.files.length > 0"
					:media-list="note.files"
				/>
				<MkPoll v-if="note.poll" :note="note" class="poll" />
				<template v-if="detailed">
					<MkUrlPreview
						v-for="url in urls"
						:key="url"
						:url="url"
						:compact="true"
						:detail="false"
						class="url-preview"
					/>
					<div
						v-if="note.renote"
						class="renote"
						@click.stop="emit('push', note.renote)"
					>
						<MkNoteSimple :note="note.renote" />
					</div>
				</template>
				<div
					v-if="(note.cw && !showContent) || collapsed"
					tabindex="0"
					v-on:focus="
						cwButton?.focus();
						showMoreButton?.focus();
					"
				></div>
			</div>
			<button
				v-if="isLong && !collapsed"
				class="_button cf-show-more is-show-less"
				@click.stop="collapsed = true"
			>
				<span>{{ t("showLess") }}</span>
			</button>
			<MkCwButton
				v-if="note.cw && showContent"
				v-model="showContent"
				:note="note"
			/>
		</div>
		<CfButton
			v-if="hasMfm && defaultStore.state.animatedMfm"
			@click.stop="toggleMfm"
			mini
			rounded
		>
			<template v-if="disableMfm">
				<i class="ph-play ph-bold"></i> {{ t("_mfm.play") }}
			</template>
			<template v-else>
				<i class="ph-stop ph-bold"></i> {{ t("_mfm.stop") }}
			</template>
		</CfButton>
	</div>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * `MkShowMoreButton` is not part of this wave's component scope, so the
//     show-more / show-less control is inlined (same DOM shape, own class names);
//   * `extractUrlFromMfm()` from `scripts/extract-url-from-mfm` is not in the
//     port scope either, so the URL extraction lives here;
//   * `note.deletedAt` does not exist on Cinderfish notes (deleted notes are
//     removed from timelines instead);
//   * `:i` is not a prop of our Mfm port, so it is not passed.
import { computed, ref } from "vue";
import * as mfm from "mfm-js";
import type { Note, NoteDetailed } from "@/api/types";
import MkNoteSimple from "@/components/MkNoteSimple.vue";
import MkMediaList from "@/components/MkMediaList.vue";
import MkPoll from "@/components/MkPoll.vue";
import MkUrlPreview from "@/components/MkUrlPreview.vue";
import MkCwButton from "@/components/MkCwButton.vue";
import CfButton from "@/components/ui/CfButton.vue";
import { notePage } from "@/filters/note";
import { extractMfmWithAnimation } from "@/scripts/extract-mfm";
import { defaultStore } from "@/store/base";
import { t } from "@/i18n";
import * as os from "@/os";

type ContentNote = Note &
	Partial<
		Pick<NoteDetailed, "reply" | "renote" | "poll" | "myReaction" | "hasPoll">
	>;

const props = defineProps<{
	note: ContentNote;
	parentId?: string;
	conversation?: boolean;
	detailed?: boolean;
	detailedView?: boolean;
}>();

const emit = defineEmits<{
	(ev: "push", v: ContentNote): void;
	(ev: "focusfooter"): void;
	(ev: "expanded", v: boolean): void;
}>();

const cwButton = ref<{ focus: () => void } | null>(null);
const showMoreButton = ref<HTMLElement | null>(null);

const isLong =
	!props.detailedView &&
	props.note.cw == null &&
	((props.note.text != null &&
		(props.note.text.split("\n").length > 10 ||
			props.note.text.length > 800)) ||
		props.note.files.length > 4);
const collapsed = ref(props.note.cw == null && isLong);

function removeHash(x: string): string {
	return x.replace(/#[^#]*$/, "");
}

/** Port of `extractUrlFromMfm()` without dragging in another scope's module. */
function extractUrlFromMfm(nodes: mfm.MfmNode[]): string[] {
	const urlNodes = mfm.extract(
		nodes,
		(node) => node.type === "url" || node.type === "link",
	);
	const all = urlNodes.map((node) =>
		node.type === "url" || node.type === "link" ? node.props.url : "",
	);
	const unique: string[] = [];
	for (const url of all) {
		if (url === "") continue;
		if (!unique.map((x) => removeHash(x)).includes(removeHash(url))) {
			unique.push(url);
		}
	}
	return unique;
}

const urls = computed<string[]>(() =>
	props.note.text
		? extractUrlFromMfm(mfm.parse(props.note.text)).slice(0, 5)
		: [],
);

const showContent = ref(false);

const mfms = props.note.text
	? extractMfmWithAnimation(mfm.parse(props.note.text))
	: [];
const hasMfm = mfms.length > 0;

const disableMfm = ref(defaultStore.state.animatedMfm);

async function toggleMfm(): Promise<void> {
	if (disableMfm.value) {
		if (!defaultStore.state.animatedMfmWarnShown) {
			const { canceled } = await os.confirm({
				type: "warning",
				text: t("_mfm.warn"),
			});
			if (canceled) return;

			defaultStore.set("animatedMfmWarnShown", true);
		}

		disableMfm.value = false;
	} else {
		disableMfm.value = true;
	}
}

function focusFooter(ev: KeyboardEvent): void {
	if (ev.key === "Tab" && !ev.getModifierState("Shift")) {
		emit("focusfooter");
	}
}
</script>

<style lang="scss" scoped>
:deep(a),
:deep(button) {
	position: relative;
	z-index: 2;
}
.reply-icon {
	display: inline-block;
	border-radius: 6px;
	padding: 0.2em 0.2em;
	margin-right: 0.2em;
	color: var(--accent);
	transition: background 0.2s;
	&:hover,
	&:focus {
		background: var(--buttonHoverBg);
	}
}
.cw {
	cursor: default;
	display: block;
	margin: 0;
	padding: 0;
	margin-bottom: 10px;
	overflow-wrap: break-word;
	> .text {
		margin-right: 8px;
	}
}

.wrmlmaau {
	.content {
		overflow-wrap: break-word;
		> .body {
			transition: filter 0.1s;
			> .rp {
				margin-left: 4px;
				font-style: oblique;
				color: var(--renote);
			}
			.reply-icon {
				display: inline-block;
				border-radius: 6px;
				padding: 0.2em 0.2em;
				margin-right: 0.2em;
				color: var(--accent);
				transition: background 0.2s;
				&:hover,
				&:focus {
					background: var(--buttonHoverBg);
				}
			}
			> :deep(.files) {
				margin-top: 0.4em;
				margin-bottom: 0.4em;
			}
			> .url-preview {
				margin-top: 8px;
			}

			> .poll {
				font-size: 80%;
			}

			> .renote {
				padding-top: 8px;
				> * {
					padding: 16px;
					border: solid 1px var(--renote);
					border-radius: 8px;
					transition: background 0.2s;
					&:hover,
					&:focus-within {
						background-color: var(--panelHighlight);
					}
				}
			}
		}

		&.collapsed,
		&.showContent {
			position: relative;
			max-height: calc(15em + 100px);
			> .body {
				max-height: inherit;
				mask: linear-gradient(black calc(100% - 64px), transparent);
				-webkit-mask: linear-gradient(
					black calc(100% - 64px),
					transparent
				);
				padding-inline: 100px;
				margin-inline: -100px;
				margin-top: -100px;
				padding-top: 100px;
				overflow: hidden;
				user-select: none;
				-webkit-user-select: none;
				-moz-user-select: none;
			}
		}
		&.collapsed {
			&.manyImages {
				max-height: calc(15em + 250px);
			}
			> .body {
				box-sizing: border-box;
			}
		}
		&.showContent {
			> .body {
				min-height: 2em;
				max-height: 5em;
				filter: blur(4px);
				:deep(span) {
					animation: none !important;
					transform: none !important;
				}
				:deep(img) {
					filter: blur(12px);
				}
			}
			:deep(.fade) {
				inset: 0;
				top: 90px;
			}
		}

		&:not(.animatedMfm) :deep(span) {
			animation: none !important;
		}
	}
	> :deep(button) {
		margin-top: 10px;
		margin-left: 0;
		margin-right: 0.4rem;
	}
	> .fade {
		position: absolute;
		inset: 0;
		bottom: -400px;
		display: flex;
		align-items: flex-end;
		z-index: 4;
		pointer-events: none;
		&::before {
			content: "";
			display: block;
			height: 100px;
			position: sticky;
			bottom: 0;
			width: 100%;
			background: var(--panel);
			mask: linear-gradient(to top, var(--gradient));
			-webkit-mask: linear-gradient(to top, var(--gradient));
			transition: background 0.2s;
		}
	}
}
</style>

<!-- Styles for the inlined show-more control (MkShowMoreButton is not part of this wave). -->
<style lang="scss" scoped>
.cf-show-more {
	&.is-fade {
		display: block;
		position: absolute;
		bottom: 0;
		left: 0;
		width: 100%;
		padding: 20px;
		margin-bottom: -10px;
		z-index: 5;
		> span {
			display: inline-block;
			background: var(--panel);
			padding: 0.4em 1em;
			font-size: 0.8em;
			border-radius: 999px;
			box-shadow: 0 2px 6px rgb(0 0 0 / 20%);
		}
		&:hover {
			> span {
				background: var(--panelHighlight);
			}
		}
	}
	&.is-show-less {
		width: 100%;
		position: sticky;
		bottom: calc(var(--stickyBottom) - 1em);
		padding: 20px;
		z-index: 5;

		> span {
			display: inline-block;
			background: var(--panel);
			padding: 6px 10px;
			font-size: 0.8em;
			border-radius: 999px;
			box-shadow: 0 0 7px 7px var(--bg);
		}
	}
}
</style>
