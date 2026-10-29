<script setup lang="ts">
import AppShell from "./layouts/AppShell.vue";
</script>

<template>
  <RouterView v-slot="{ Component, route }">
    <component :is="Component" v-if="route.meta.bare" />
    <AppShell v-else>
      <component :is="Component" />
    </AppShell>
  </RouterView>
</template>
