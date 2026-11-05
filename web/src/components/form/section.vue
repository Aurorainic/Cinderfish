<!-- Ported from firefish@b425f61:packages/client/src/components/form/section.vue (AGPL-3.0-only) -->
<template>
	<section class="vrtktovh _formBlock">
		<h3 class="label"><slot name="label" /></h3>
		<div class="main _formRoot">
			<slot />
		</div>
	</section>
</template>

<script setup lang="ts">
// Sectioned forms were not used by any wave-2 page, so there was no Cf
// equivalent to wrap; this is a direct port.
</script>

<style lang="scss" scoped>
.vrtktovh {
	border-top: solid 0.5px var(--divider);
	border-bottom: solid 0.5px var(--divider);

	& + .vrtktovh {
		border-top: none;
	}

	&:first-child {
		border-top: none;
	}

	&:last-child {
		border-bottom: none;
	}

	> .label {
		font-weight: bold;
		margin: 1.5em 0 16px 0;
		font-size: 1em;

		&:empty {
			display: none;
		}
	}

	> .main {
		margin: 1.5em 0;
	}
}
</style>
