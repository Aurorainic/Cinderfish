<!-- Ported from firefish@b425f61:packages/client/src/components/form/folder.vue (AGPL-3.0-only) -->
<template>
	<details class="dwzlatin" :open="opened" @toggle="onToggle">
		<summary class="header _button">
			<span class="icon"><slot name="icon" /></span>
			<span class="text"><slot name="label" /></span>
			<span class="right">
				<span class="suffix"><slot name="suffix" /></span>
				<i
					class="ph-bold ph-lg icon"
					:class="opened ? 'ph-caret-up' : 'ph-caret-down'"
				></i>
			</span>
		</summary>
		<div class="body">
			<MkSpacer :margin-min="14" :margin-max="22">
				<slot />
			</MkSpacer>
		</div>
	</details>
</template>

<script setup lang="ts">
// `<details>` gives the collapse behaviour for free; `opened` only drives the
// caret, since the browser owns the actual open state. Upstream's template reads
// an undefined `opened` (it was lost in the Vue 3 rewrite), so the caret never
// flipped — this is the fix, not a behaviour change.
import { ref } from "vue";
import MkSpacer from "@/components/global/MkSpacer.vue";

const props = withDefaults(
	defineProps<{
		defaultOpen?: boolean;
	}>(),
	{
		defaultOpen: false,
	},
);

const opened = ref(props.defaultOpen);

function onToggle(event: Event): void {
	opened.value = (event.currentTarget as HTMLDetailsElement).open;
}
</script>

<style lang="scss" scoped>
.dwzlatin {
	display: block;
	overflow: clip;
	border-radius: 6px;

	> .header {
		display: flex;
		align-items: center;
		inline-size: 100%;
		box-sizing: border-box;
		padding: 12px 14px;
		background: var(--buttonBg);
		cursor: pointer;
		list-style: none;

		&::-webkit-details-marker {
			display: none;
		}

		&:hover {
			text-decoration: none;
			background: var(--buttonHoverBg);
		}

		> .icon {
			margin-inline-end: 0.75em;
			flex-shrink: 0;
			text-align: center;
			opacity: 0.8;

			&:empty {
				display: none;

				& + .text {
					padding-inline-start: 4px;
				}
			}
		}

		> .text {
			white-space: nowrap;
			text-overflow: ellipsis;
			overflow: hidden;
			padding-inline-end: 12px;
		}

		> .right {
			margin-inline-start: auto;
			color: var(--fgTransparentWeak);
			white-space: nowrap;

			> .suffix:not(:empty) {
				margin-inline-end: 0.75em;
			}
		}
	}

	> .body {
		background: var(--panel);
		border-radius: 0 0 6px 6px;
	}
}
</style>
