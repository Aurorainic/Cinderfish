<!-- Cinderfish: replaces firefish@b425f61:packages/client/src/components/MkContainer.vue (AGPL-3.0-only) -->
<!--
	A widget card: title row (icon + label + actions) and a content area. Firefish
	uses MkContainer, which carries ad slots, "show more" footers and a naked
	mode; none of those are needed for a widgets column, so this is the reduced
	shape every widget shares.
-->
<template>
	<section
		class="card"
		:class="{ naked, bare }"
		:style="bodyStyle"
	>
		<header v-if="showHeader && !bare">
			<div class="title">
				<i v-if="icon" :class="icon"></i>
				<span class="text">{{ title }}</span>
			</div>
			<div v-if="$slots.actions" class="actions">
				<slot name="actions"></slot>
			</div>
		</header>
		<div class="body" :class="{ scrollable }">
			<slot></slot>
		</div>
	</section>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
	defineProps<{
		title?: string;
		icon?: string;
		showHeader?: boolean;
		/** No panel background or radius (Firefish's `naked`). */
		naked?: boolean;
		/** Render only the content area, without the title row. */
		bare?: boolean;
		/** Fixed content height in pixels; unset means "grow with content". */
		height?: number | null;
		scrollable?: boolean;
	}>(),
	{
		title: "",
		icon: "",
		showHeader: true,
		naked: false,
		bare: false,
		height: null,
		scrollable: false,
	},
);

const bodyStyle = computed<Record<string, string>>(() => {
	const style: Record<string, string> = {};
	if (props.height != null) {
		style["--widget-body-height"] = `${props.height}px`;
	}
	return style;
});
</script>

<style lang="scss" scoped>
.card {
	display: flex;
	flex-direction: column;
	background: var(--panel);
	border-radius: var(--radius);
	overflow: clip;

	&.naked {
		background: none;
	}

	> header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		font-size: 0.9em;
		font-weight: bold;
		color: var(--fg);
		border-bottom: solid 0.5px var(--divider);

		> .title {
			display: flex;
			align-items: center;
			gap: 6px;
			min-width: 0;

			> .text {
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
			}
		}

		> .actions {
			display: flex;
			align-items: center;
			margin-inline-start: auto;

			> :deep(button) {
				padding: 0 4px;
			}
		}
	}

	> .body {
		min-width: 0;

		&.scrollable {
			height: var(--widget-body-height, 300px);
			overflow: auto;
			overscroll-behavior: contain;
		}
	}
}
</style>
