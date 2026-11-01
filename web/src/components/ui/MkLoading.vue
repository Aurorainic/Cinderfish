// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkLoading.vue
<template>
	<div
		class="cinderfish-loading"
		:class="{ inline, colored, mini, em }"
		:style="em ? { '--size': `${sizeEm}em` } : undefined"
	>
		<div class="container" aria-hidden="true">
			<svg class="spinner" viewBox="0 0 50 50" xmlns="http://www.w3.org/2000/svg">
				<circle
					class="path"
					cx="25"
					cy="25"
					r="20"
					fill="none"
					stroke="currentColor"
					stroke-width="6px"
				></circle>
			</svg>
		</div>
	</div>
</template>

<script lang="ts" setup>
withDefaults(
	defineProps<{
		inline?: boolean;
		colored?: boolean;
		mini?: boolean;
		em?: boolean;
		sizeEm?: number;
	}>(),
	{
		inline: false,
		colored: true,
		mini: false,
		em: false,
		sizeEm: 1,
	},
);
</script>

<style scoped>
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

.cinderfish-loading {
	padding: 32px;
	text-align: center;
	cursor: wait;

	--size: 40px;
}

.cinderfish-loading.colored {
	color: var(--accent);
}

.cinderfish-loading.inline {
	display: inline;
	padding: 0;
	--size: 32px;
}

.cinderfish-loading.mini {
	padding: 16px;
	--size: 32px;
}

.cinderfish-loading.em {
	display: inline-block;
	vertical-align: middle;
	padding: 0;
}

.cinderfish-loading > .container {
	position: relative;
	inline-size: var(--size);
	block-size: var(--size);
	margin-block: 0;
	margin-inline: auto;
}

.cinderfish-loading > .container > .spinner {
	position: absolute;
	inset: 0;
	z-index: 999;
	inline-size: var(--size);
	block-size: var(--size);
	animation: spin 2s linear infinite;
}

.cinderfish-loading > .container > .spinner > .path {
	stroke: var(--accent);
	stroke-linecap: round;
	animation: dash 1.2s ease-in-out infinite;
}
</style>
