<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkStickyContainer.vue (AGPL-3.0-only) -->
<template>
	<div class="sticky-container">
		<div ref="headerEl">
			<slot name="header"></slot>
		</div>
		<div ref="bodyEl" :data-sticky-container-header-height="headerHeight">
			<slot></slot>
		</div>
	</div>
</template>

<script lang="ts">
// なんか動かない
// const CURRENT_STICKY_TOP = Symbol('CURRENT_STICKY_TOP');
const CURRENT_STICKY_TOP = "CURRENT_STICKY_TOP";
</script>

<script setup lang="ts">
import type { Ref } from "vue";
import { inject, onMounted, onUnmounted, provide, ref, watch } from "vue";

// Firefish's reactivity transform ($ref / $$) is not enabled in Cinderfish, so
// the same state is written with ref() and a provide/inject handshake.
const headerEl = ref<HTMLElement | null>(null);
const bodyEl = ref<HTMLElement | null>(null);
const headerHeight = ref<string | undefined>(undefined);
const childStickyTop = ref(0);
const parentStickyTop = inject<Ref<number>>(CURRENT_STICKY_TOP, ref(0));
provide(CURRENT_STICKY_TOP, childStickyTop);

function calc(): void {
	if (headerEl.value == null) return;
	childStickyTop.value = parentStickyTop.value + headerEl.value.offsetHeight;
	headerHeight.value = headerEl.value.offsetHeight.toString();
}

const observer = new ResizeObserver(() => {
	window.setTimeout(() => {
		calc();
	}, 100);
});

onMounted(() => {
	calc();

	watch(parentStickyTop, calc);

	watch(
		childStickyTop,
		() => {
			if (bodyEl.value == null) return;
			bodyEl.value.style.setProperty(
				"--stickyTop",
				childStickyTop.value + "px",
			);
		},
		{
			immediate: true,
		},
	);

	if (headerEl.value != null) observer.observe(headerEl.value);
});

onUnmounted(() => {
	observer.disconnect();
});
</script>

<style lang="scss">
.sticky-container {
	display: flex;
	flex-direction: column;
	> div:first-child {
		position: sticky;
		top: var(--stickyTop, 0);
		z-index: 1000;
	}
	> div:last-child {
		display: flex;
		flex-direction: column;
		flex-grow: 1;
	}
}
</style>
