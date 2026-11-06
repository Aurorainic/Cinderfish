<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<Favorites />
</template>

<script lang="ts" setup>
// Alias for /my/favorites (wave-4 contract §3.6): renders the real Favorites
// page so there is exactly one implementation, and no route or logic duplicate.
import Favorites from "./Favorites.vue";

defineOptions({ name: "CfPageMyFavorites" });
</script>
