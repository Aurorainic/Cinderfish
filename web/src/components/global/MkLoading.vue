<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkLoading.vue (AGPL-3.0-only) -->
<template>
	<div class="root" :class="{ inline, colored, mini, em }">
		<div class="container" aria-hidden="true">
			<svg
				class="spinner"
				viewBox="0 0 50 50"
				xmlns="http://www.w3.org/2000/svg"
			>
				<circle
					class="path"
					cx="25"
					cy="25"
					r="20"
					fill="none"
					stroke-width="6px"
					style="fill: none; stroke: currentColor; stroke-width: 6px"
				></circle>
			</svg>
		</div>
	</div>
</template>

<script setup lang="ts">
withDefaults(
	defineProps<{
		inline?: boolean;
		colored?: boolean;
		mini?: boolean;
		em?: boolean;
	}>(),
	{
		inline: false,
		colored: true,
		mini: false,
		em: false,
	},
);
</script>

<style lang="scss" scoped>
/* Credit to https://codepen.io/supah/pen/BjYLdW */

@keyframes spin {
	0% {
		transform: rotate(0deg);
	}
	100% {
		transform: rotate(360deg);
	}
}

@keyframes dash {
	0% {
		stroke-dasharray: 1, 150;
		stroke-dashoffset: 0;
	}
	50% {
		stroke-dasharray: 90, 150;
		stroke-dashoffset: -35;
	}
	100% {
		stroke-dasharray: 90, 150;
		stroke-dashoffset: -124;
	}
}

.root {
	padding: 32px;
	text-align: center;
	cursor: wait;

	--size: 40px;

	&.colored {
		color: var(--accent);
	}

	&.inline {
		display: inline;
		padding: 0;
		--size: 32px;
	}

	&.mini {
		padding: 16px;
		--size: 32px;
	}

	&.em {
		display: inline-block;
		vertical-align: middle;
		padding: 0;
		--size: 1em;
	}
}

.container {
	position: relative;
	width: var(--size);
	height: var(--size);
	margin: 0 auto;
}

.spinner {
	position: absolute;
	top: 0;
	left: 0;
	z-index: 999;
	width: var(--size);
	height: var(--size);
	animation: spin 2s linear infinite;
}

.path {
	stroke: var(--accent);
	stroke-linecap: round;
	animation: dash 1.2s ease-in-out infinite;
}
</style>
