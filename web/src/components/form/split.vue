<!-- Ported from firefish@b425f61:packages/client/src/components/form/split.vue (AGPL-3.0-only) -->
<template>
	<div
		class="terlnhxf _formBlock"
		:style="{
			gridTemplateColumns: `repeat(auto-fill, minmax(${minWidth}, 1fr))`,
		}"
	>
		<slot />
	</div>
</template>

<script setup lang="ts">
// No Cf equivalent: wave 2 had no use for the two-column form grid. The grid
// template is applied inline instead of through `v-bind()` in SCSS, so the
// component needs no dynamic style module.
//
// Upstream also forced `margin: 0` on every child with `!important` to cancel
// the margins of its nested form blocks; the ported form wrappers have no outer
// margin (only `._formBlock` does, and those are never children here), so the
// rule is dropped rather than re-implemented.
import { computed } from "vue";

const props = withDefaults(
	defineProps<{
		minWidth?: number;
	}>(),
	{
		minWidth: 210,
	},
);

const minWidth = computed(() => `${props.minWidth}px`);
</script>

<style lang="scss" scoped>
.terlnhxf {
	display: grid;
	grid-gap: 12px;
}
</style>
