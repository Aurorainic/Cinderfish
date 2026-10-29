<script setup lang="ts">
import AppShell from "./layouts/AppShell.vue";
import { OsOverlay } from "./os";
import { shellMode } from "./stores/shell";
</script>

<template>
  <RouterView v-slot="{ Component, route }">
    <component :is="Component" v-if="route.meta.bare" />
    <AppShell v-else :mode="shellMode">
      <component :is="Component" />
    </AppShell>
  </RouterView>
  <OsOverlay />
</template>
