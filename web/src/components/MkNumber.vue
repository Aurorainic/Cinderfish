<!-- Ported from firefish@b425f61:packages/client/src/components/MkNumber.vue (AGPL-3.0-only) -->
<template>
	<span>{{ number(Math.floor(value)) }}</span>
</template>

<script lang="ts" setup>
// Degradation (contract §2): Firefish animates the counter with gsap. gsap is
// not part of the Cinderfish dependency set, so the value is rendered as-is.
import number from "@/filters/number";

defineProps<{
	value: number;
}>();
</script>
