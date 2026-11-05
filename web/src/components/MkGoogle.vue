<!-- Ported from firefish@b425f61:packages/client/src/components/MkGoogle.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-google" @click.stop>
		<input v-model="query" type="search" :placeholder="q" />
		<button @click="search">
			<i class="ph-magnifying-glass ph-bold ph-lg"></i>
			{{ t("searchByGoogle") }}
		</button>
	</div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { t } from "../i18n";

// Firefish opens a self-hosted search front-end; the port targets Google, which
// is the documented fallback in the wave-2 contract (the "search" MFM function
// has no backend support in Cinderfish).
const props = defineProps<{
	q: string;
}>();

const query = ref(props.q);

function search(): void {
	window.open(
		"https://www.google.com/search?q=" + encodeURIComponent(query.value),
		"_blank",
		"noopener",
	);
}
</script>

<style lang="scss" scoped>
.mk-google {
	display: flex;
	margin: 8px 0;

	> input {
		flex-shrink: 1;
		padding: 10px;
		width: 100%;
		height: 40px;
		font-size: 16px;
		border: solid 1px var(--divider);
		border-radius: 4px 0 0 4px;
		-webkit-appearance: none;
		-webkit-border-radius: 4px 0 0 4px;
	}

	> button {
		flex-shrink: 0;
		margin: 0;
		padding: 0 16px;
		border: solid 1px var(--divider);
		border-left: none;
		border-radius: 0 4px 4px 0;

		&:active {
			box-shadow: 0 2px 4px rgba(#000, 0.15) inset;
		}
	}
}
</style>
