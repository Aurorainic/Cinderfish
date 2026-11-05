<!-- Ported from firefish@b425f61:packages/client/src/components/MkFormula.vue (AGPL-3.0-only) -->
<template>
	<pre v-if="block" class="mk-formula block"><code>{{ formula }}</code></pre>
	<code v-else class="mk-formula inline">{{ formula }}</code>
</template>

<script setup lang="ts">
// Degraded versus Firefish: katex is not in the Cinderfish dependency set
// (wave-2 contract 5), so formulas render as code text instead of typeset
// math. The props match MkFormulaCore so the component can be upgraded
// without touching call sites.
withDefaults(
	defineProps<{
		formula: string;
		block: boolean;
	}>(),
	{},
);
</script>

<style lang="scss" scoped>
.mk-formula {
	&.block {
		display: block;
		overflow: auto;
		padding: 8px;
		text-align: center;
		border-radius: var(--radius);
		background: var(--panelHighlight);
	}

	&.inline {
		padding: 0 4px;
		font-size: 0.9em;
	}
}
</style>
