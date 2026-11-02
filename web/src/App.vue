<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import AppShell from "./layouts/AppShell.vue";
import DeckShell from "./layouts/DeckShell.vue";
import { OsOverlay } from "./os";
import ToastHost from "./components/DialogHost.vue";
import { shellLayout, shellMode } from "./stores/shell";

// The deck preference is not the layout. Only the three surfaces the deck
// renders as columns carry `meta.deck`; every other route keeps the scrolling
// universal layout even when the preference is "deck", so a deep link to
// /notes/:id, /@handle or /settings/* is never put inside the fixed-height deck
// box. Deck columns themselves open notes and profiles in the page window
// instead of navigating, so a reader can stay on the deck.
const route = useRoute();
const deckPreferred = computed(() => shellMode.value === "deck");
const deckRoute = computed(() => route.meta.deck === true);
const showsDeck = computed(() => shellLayout(deckPreferred.value, deckRoute.value) === "deck");
</script>

<template>
  <RouterView v-slot="{ Component, route: matched }">
    <component :is="Component" v-if="matched.meta.bare" />
    <AppShell v-else :mode="showsDeck ? 'deck' : 'universal'" :deck-preferred="deckPreferred">
      <DeckShell v-if="showsDeck" />
      <component :is="Component" v-else />
    </AppShell>
  </RouterView>
  <OsOverlay />
  <!-- lib/os.ts toasts; dialogs, menus and the composer render in OsOverlay. -->
  <ToastHost />
</template>
