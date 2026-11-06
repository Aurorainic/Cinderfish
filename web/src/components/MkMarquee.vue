<!-- Ported from firefish@b425f61:packages/client/src/components/MkMarquee.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-marquee">
		<div
			class="track"
			:style="{
				animationDuration: `${duration}s`,
				animationDirection: reverse ? 'reverse' : 'normal',
			}"
		>
			<div class="copy"><slot /></div>
			<div class="copy" aria-hidden="true"><slot /></div>
		</div>
	</div>
</template>

<script setup lang="ts">
// Firefish scrolls the slot with a Web Animation loop. The CSS version keeps
// the same parameters (duration in seconds, reverse) and duplicates the content
// so the loop has no gap; it pauses when the user prefers reduced motion.
defineProps<{ duration?: number; reverse?: boolean }>();
</script>

<style lang="scss" scoped>
.mk-marquee {
	overflow: hidden;
	white-space: nowrap;

	> .track {
		display: inline-flex;
		width: max-content;
		animation-name: mk-marquee;
		animation-timing-function: linear;
		animation-iteration-count: infinite;

		> .copy {
			display: inline-flex;
			flex-shrink: 0;
		}
	}
}

@keyframes mk-marquee {
	from {
		transform: translateX(0);
	}
	to {
		transform: translateX(-50%);
	}
}

@media (prefers-reduced-motion: reduce) {
	.mk-marquee > .track {
		animation: none;
	}
}
</style>
