<!-- Ported from firefish@b425f61:packages/client/src/pages/_empty_.vue (AGPL-3.0-only) -->
<template>
	<div></div>
</template>

<script lang="ts" setup>
// Rendered as the index child of /settings and /admin: those shells redirect to
// a concrete tab, so this route only exists to keep the parent mounted.
defineOptions({ name: "CfPageEmpty" });
</script>
