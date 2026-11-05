<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkSpacer.vue (AGPL-3.0-only) -->
<template>
	<div ref="root" class="root" :style="{ padding: margin + 'px' }">
		<div ref="content" class="content">
			<slot></slot>
		</div>
	</div>
</template>

<script setup lang="ts">
import { inject, onMounted, onUnmounted, ref } from "vue";

// Firefish reads deviceKind from scripts/device-kind and the classic-UI flag
// from config; the port derives the same smartphone decision from the viewport
// so the component does not depend on another workstream's script.
const props = withDefaults(
	defineProps<{
		contentMax?: number | null;
		marginMin?: number;
		marginMax?: number;
	}>(),
	{
		contentMax: null,
		marginMin: 12,
		marginMax: 24,
	},
);

const root = ref<HTMLElement | null>(null);
const content = ref<HTMLElement | null>(null);
const margin = ref(0);
const shouldSpacerMin = inject<boolean>("shouldSpacerMin", false);

let ro: ResizeObserver | null = null;

function isSmartphone(): boolean {
	return window.matchMedia("(max-width: 500px)").matches;
}

function adjust(rect: { width: number; height: number }): void {
	if (shouldSpacerMin || isSmartphone()) {
		margin.value = props.marginMin;
		return;
	}

	if (
		rect.width > (props.contentMax ?? 0) ||
		(rect.width > 360 && window.innerWidth > 400)
	) {
		margin.value = props.marginMax;
	} else {
		margin.value = props.marginMin;
	}
}

onMounted(() => {
	ro = new ResizeObserver(() => {
		if (root.value == null) return;
		adjust({
			width: root.value.offsetWidth,
			height: root.value.offsetHeight,
		});
	});
	if (root.value != null) ro.observe(root.value);

	if (props.contentMax && content.value != null) {
		content.value.style.maxWidth = props.contentMax + "px";
	}
});

onUnmounted(() => {
	if (ro != null) ro.disconnect();
});
</script>

<style lang="scss" scoped>
.root {
	box-sizing: border-box;
	width: 100%;
}

.content {
	margin: 0 auto;
}
</style>
