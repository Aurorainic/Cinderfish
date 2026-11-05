<!-- Ported from firefish@b425f61:packages/client/src/components/MkCode.vue (AGPL-3.0-only) -->
<template>
	<code v-if="inline" class="mk-code inline" :class="languageClass">{{ code }}</code>
	<pre v-else class="mk-code block" :class="languageClass"><code :class="languageClass">{{ code }}</code></pre>
</template>

<script setup lang="ts">
import { computed } from "vue";

// Degraded versus Firefish: prismjs is not in the Cinderfish dependency set, so
// code blocks keep their structure and language class but are not highlighted.
// Firefish loads MkCode.core.vue asynchronously; the reduced renderer is small
// enough to stay inline.
const props = defineProps<{
	code: string;
	lang?: string;
	inline?: boolean;
}>();

const languageClass = computed<string>(
	() => "language-" + (props.lang ?? "plaintext"),
);
</script>

<style lang="scss" scoped>
.mk-code {
	&.block {
		display: block;
		overflow: auto;
		padding: 8px;
		border-radius: var(--radius);
		background: var(--panelHighlight);

		> code {
			font-size: 0.8em;
		}
	}

	&.inline {
		padding: 4px 6px;
		font-size: 0.8em;
		word-break: break-all;
	}
}
</style>
