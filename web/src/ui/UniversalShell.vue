<!-- Ported from firefish@b425f61:packages/client/src/ui/universal.vue (AGPL-3.0-only) -->
<template>
	<UniversalLayout />
</template>

<script setup lang="ts">
// `main.ts` mounts one of four shells (universal / deck / zen / visitor).
// Firefish's `ui/universal.vue` is the universal one; the shell entry exists
// only because the registry is keyed by shell name, so this file stays a
// one-line delegation to the layout.
import UniversalLayout from "./layouts/UniversalLayout.vue";
</script>
