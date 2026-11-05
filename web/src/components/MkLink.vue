<!-- Ported from firefish@b425f61:packages/client/src/components/MkLink.vue (AGPL-3.0-only) -->
<template>
	<MkA
		v-if="self"
		class="xlcxczvw _link"
		:to="path"
		:rel="rel ?? undefined"
		:title="url"
		@click.stop
	>
		<slot></slot>
	</MkA>
	<a
		v-else
		class="xlcxczvw _link"
		:href="url"
		:rel="rel ?? undefined"
		target="_blank"
		:title="url"
		@click.stop
	>
		<slot></slot>
		<i class="ph-arrow-square-out ph-bold ph-lg icon"></i>
	</a>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { url as local } from "../config";
import MkA from "./global/MkA.vue";

// Degraded versus Firefish: the URL preview popup (MkUrlPreviewPopup) is not
// ported because there is no url-preview endpoint.
const props = withDefaults(
	defineProps<{
		url: string;
		rel?: string | null;
	}>(),
	{
		rel: null,
	},
);

const self = computed<boolean>(() => props.url.startsWith(local));

const path = computed<string>(() => {
	const suffix = props.url.substring(local.length);
	if (suffix === "") return "/";
	return suffix.startsWith("/") ? suffix : "/" + suffix;
});
</script>

<style lang="scss" scoped>
.xlcxczvw {
	word-break: break-all;

	> .icon {
		padding-left: 2px;
		font-size: 0.9em;
	}
}
</style>
