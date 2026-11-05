<!-- Ported from firefish@b425f61:packages/client/src/components/form/slot.vue (AGPL-3.0-only) -->
<template>
	<div class="adhpbeou">
		<div class="label"><slot name="label" /></div>
		<div class="content"><slot /></div>
		<div class="caption"><slot name="caption" /></div>
	</div>
</template>

<script setup lang="ts">
// A labelled container for content that is not a single input (a nested form,
// a message, a list of buttons). No Cf equivalent exists because wave 2 had no
// settings page to host one.
</script>

<style lang="scss" scoped>
.adhpbeou {
	> .label {
		font-size: 0.85em;
		padding: 0 0 8px 0;
		user-select: none;

		&:empty {
			display: none;
		}
	}

	> .caption {
		font-size: 0.85em;
		padding: 8px 0 0 0;
		color: var(--fgTransparentWeak);

		&:empty {
			display: none;
		}
	}
}
</style>
