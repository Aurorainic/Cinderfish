<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import AppShell from "./layouts/AppShell.vue";
import DeckShell from "./layouts/DeckShell.vue";
import { OsOverlay } from "./os";
import { shellMode } from "./stores/shell";

// Deck mode takes over the main area for the three surfaces it renders as
// columns. Note and profile pages keep routing (a direct link to /notes/:id or
// /@user must still work), and so do settings, admin and the auth pages; deck
// columns open notes and profiles in the overlay instead of navigating.
const DECK_ROUTES = new Set(["home", "notifications", "explore"]);

const route = useRoute();
const showsDeck = computed(
  () => shellMode.value === "deck" && DECK_ROUTES.has(String(route.name ?? "")),
);
</script>

<template>
  <RouterView v-slot="{ Component, route: matched }">
    <component :is="Component" v-if="matched.meta.bare" />
    <AppShell v-else :mode="shellMode">
      <DeckShell v-if="showsDeck" />
      <component :is="Component" v-else />
    </AppShell>
  </RouterView>
  <OsOverlay />
</template>
