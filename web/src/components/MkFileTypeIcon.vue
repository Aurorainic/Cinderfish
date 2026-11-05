<!-- Ported from firefish@b425f61:packages/client/src/components/MkFileTypeIcon.vue (AGPL-3.0-only) -->
<template>
	<span class="mk-file-type-icon">
		<template v-if="kind == 'image'"
			><i class="ph-file-image ph-bold ph-lg"></i
		></template>
		<template v-else-if="kind == 'video'"
			><i class="ph-file-video ph-bold ph-lg"></i
		></template>
		<template v-else-if="kind == 'audio'"
			><i class="ph-file-audio ph-bold ph-lg"></i
		></template>
		<template v-else><i class="ph-file ph-bold ph-lg"></i></template>
	</span>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
	type: string;
}>();

const kind = computed(() => props.type.split("/")[0]);
</script>
