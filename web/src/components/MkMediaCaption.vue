<!-- Ported from firefish@b425f61:packages/client/src/components/MkMediaCaption.vue (AGPL-3.0-only) -->
<template>
	<CfModal ref="modal" @click="onBgClick" @closed="emit('closed')">
		<div class="container">
			<div class="fullwidth top-caption">
				<div class="mk-dialog">
					<header>
						<Mfm v-if="title" class="title" :text="title" />
						<span
							class="text-count"
							:class="{ over: remainingLength < 0 }"
							>{{ remainingLength }}</span
						>
						<br />
					</header>
					<textarea
						id="captioninput"
						v-model="inputValue"
						autofocus
						:placeholder="input.placeholder ?? ''"
						@keydown="onInputKeydown"
					></textarea>
					<div
						v-if="
							showOkButton ||
							showCaptionButton ||
							showCancelButton
						"
						class="buttons"
					>
						<CfButton
							inline
							primary
							:disabled="remainingLength < 0"
							@click="ok"
							>{{ t("ok") }}</CfButton
						>
						<CfButton inline @click="cancel">{{
							t("cancel")
						}}</CfButton>
					</div>
				</div>
			</div>
			<div class="hdrwpsaf fullwidth">
				<header>{{ image.name }}</header>
				<img
					id="imgtocaption"
					:src="imageUrl"
					:alt="image.comment ?? ''"
					:title="image.comment ?? undefined"
					@click="close()"
				/>
				<footer>
					<span>{{ image.type }}</span>
					<span>{{ bytes(image.size) }}</span>
				</footer>
			</div>
		</div>
	</CfModal>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { DriveFile } from "@/api/types";
import { resolveMediaUrl } from "@/api/client";
import { CfModal } from "@/os";
import { instance } from "@/instance";
import { CfButton } from "@/components/ui";
import { t } from "@/i18n";

/**
 * Ported from Firefish. Degradations:
 *   - the AI "caption" button called drive/files/caption-image, which does not
 *     exist in Cinderfish (docs/research/api-contract/drive-media.md), so it is
 *     not rendered;
 *   - stringz and insert-text-at-cursor are not installed, so the remaining
 *     length counts code points with Intl.Segmenter-free spreading;
 *   - /api/meta has no maxCaptionTextLength: the 512-character backend limit is
 *     used;
 *   - the image width/height read `image.properties`, which the Cinderfish
 *     DriveFile does not have, so the footer omits the dimensions.
 */
const props = withDefaults(
	defineProps<{
		image: DriveFile;
		title?: string;
		input: { default?: string | null; placeholder?: string | null };
		showOkButton?: boolean;
		showCaptionButton?: boolean;
		showCancelButton?: boolean;
		cancelableByBgClick?: boolean;
	}>(),
	{
		showOkButton: true,
		showCaptionButton: false,
		showCancelButton: true,
		cancelableByBgClick: true,
	},
);

const emit = defineEmits<{
	done: [result: { canceled: boolean; result?: string }];
	closed: [];
}>();

const modal = ref<{ close: () => void } | null>(null);
const inputValue = ref<string>(props.input.default ?? "");
/** Synthesised by @/instance from the backend caption limit (default 512). */
const maxCaptionLength = computed(() => instance.maxCaptionTextLength);

const imageUrl = computed(() => resolveMediaUrl(props.image.url));

const remainingLength = computed(
	() => maxCaptionLength.value - [...inputValue.value].length,
);

function bytes(v: number | null | undefined, digits = 0): string {
	if (v == null) return "?";
	if (v === 0) return "0";
	const sizes = ["B", "KB", "MB", "GB", "TB"];
	const negative = v < 0;
	const value = Math.abs(v);
	const i = Math.min(
		sizes.length - 1,
		Math.floor(Math.log(value) / Math.log(1024)),
	);
	const scaled = (value / Math.pow(1024, i)).toFixed(digits).replace(/\.0+$/, "");
	return `${negative ? "-" : ""}${scaled}${sizes[i]}`;
}

function close(): void {
	modal.value?.close();
}

function done(canceled: boolean, result?: string): void {
	emit("done", { canceled, result });
	close();
}

function ok(): void {
	if (!props.showOkButton) return;
	if (remainingLength.value < 0) return;
	done(false, inputValue.value);
}

function cancel(): void {
	done(true);
}

function onBgClick(): void {
	if (props.cancelableByBgClick) cancel();
}

function onInputKeydown(ev: KeyboardEvent): void {
	if (ev.key === "Enter" && ev.ctrlKey) {
		ev.preventDefault();
		ev.stopPropagation();
		ok();
	}
}
</script>

<style lang="scss" scoped>
.container {
	display: flex;
	width: 100%;
	height: 100%;
	flex-direction: row;
	overflow: scroll;
	position: fixed;
	left: 0;
	top: 0;
}
@media (max-width: 850px) {
	.container {
		flex-direction: column;
	}
	.top-caption {
		padding-bottom: 8px;
	}
}
.fullwidth {
	width: 100%;
	margin: auto;
}
.mk-dialog {
	position: relative;
	padding: 32px;
	min-width: 320px;
	max-width: 480px;
	box-sizing: border-box;
	text-align: center;
	background: var(--panel);
	border-radius: var(--radius);
	margin: auto;

	> header {
		margin: 0 0 8px 0;
		position: relative;

		> .title {
			font-weight: bold;
			font-size: 20px;
		}

		> .text-count {
			color: var(--fgTransparentWeak);
			position: absolute;
			right: 0;
		}
	}

	> .buttons {
		margin-top: 16px;

		> * {
			margin: 0 8px;
		}
	}

	> textarea {
		display: block;
		box-sizing: border-box;
		padding: 0 24px;
		margin: 0;
		width: 100%;
		font-size: 16px;
		border: none;
		border-radius: 0;
		background: transparent;
		color: var(--fg);
		font-family: inherit;
		max-width: 100%;
		min-width: 100%;
		min-height: 90px;

		&:focus-visible {
			outline: none;
		}

		&:disabled {
			opacity: 0.5;
		}
	}
}
.hdrwpsaf {
	display: flex;
	flex-direction: column;
	height: 100%;

	> header,
	> footer {
		align-self: center;
		display: inline-block;
		padding: 6px 9px;
		font-size: 90%;
		background: rgba(0, 0, 0, 0.5);
		border-radius: 6px;
		color: #fff;
	}

	> header {
		margin-bottom: 8px;
		opacity: 0.9;
	}

	> img {
		display: block;
		flex: 1;
		min-height: 0;
		object-fit: contain;
		width: 100%;
		cursor: zoom-out;
		image-orientation: from-image;
	}

	> footer {
		margin-top: 8px;
		color: var(--fgTransparentWeak);

		> span + span {
			margin-left: 0.5em;
			padding-left: 0.5em;
			border-left: solid 1px rgba(255, 255, 255, 0.5);
		}
	}
}
</style>