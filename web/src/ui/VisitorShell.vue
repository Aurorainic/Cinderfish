<!-- Ported from firefish@b425f61:packages/client/src/ui/visitor.vue (AGPL-3.0-only) -->
<template>
	<VisitorLayout />
</template>

<script setup lang="ts">
// `main.ts` mounts one of four shells (universal / deck / zen / visitor).
// The visitor chrome lives in `layouts/VisitorLayout.vue`; this entry only
// delegates, matching UniversalShell.vue and DeckShell.vue.
import VisitorLayout from "./layouts/VisitorLayout.vue";
</script>
