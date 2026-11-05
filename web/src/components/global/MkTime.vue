<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkTime.vue (AGPL-3.0-only) -->
<template>
	<CfTime :time="time" :origin="origin" :mode="mode" />
</template>

<script setup lang="ts">
import CfTime from "../ui/CfTime.vue";

// Cinderfish renders dates through the wave-1 atom; MkTime keeps the Firefish
// component name and prop surface so ported templates stay unchanged.
withDefaults(
	defineProps<{
		time: Date | string | number | null;
		origin?: Date | null;
		mode?: "relative" | "absolute" | "detail";
	}>(),
	{
		origin: null,
		mode: "relative",
	},
);
</script>
