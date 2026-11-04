<template>
	<div
		class="mk-loading"
		:class="{ inline, colored, mini, em }"
		:style="em ? { '--size': `${sizeEm}em` } : undefined"
	>
		<div class="container" aria-hidden="true">
			<svg class="spinner" viewBox="0 0 50 50" xmlns="http://www.w3.org/2000/svg">
				<circle class="path" cx="25" cy="25" r="20" fill="none" stroke-width="6px"></circle>
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

<style lang="scss" scoped>
@keyframes cf-spin {
	0% {
		transform: rotate(0deg);
	}
	100% {
		transform: rotate(360deg);
	}
}

@keyframes cf-dash {
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

.mk-loading {
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
	}
}

.container {
	position: relative;
	inline-size: var(--size);
	block-size: var(--size);
	margin-inline: auto;
}

.spinner {
	position: absolute;
	inset: 0;
	z-index: 1;
	inline-size: var(--size);
	block-size: var(--size);
	animation: cf-spin 2s linear infinite;
}

.path {
	stroke: var(--accent);
	stroke-linecap: round;
	animation: cf-dash 1.2s ease-in-out infinite;
}
</style>
