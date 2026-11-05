<!-- Ported from firefish@b425f61:packages/client/src/ui/deck.vue (AGPL-3.0-only) -->
<template>
	<DeckLayout />
</template>

<script setup lang="ts">
// Firefish's `ui/deck.vue` *is* the deck shell: it renders the navbar rail
// (`XSidebar`) and the horizontal run of deck columns. Those two pieces live in
// `layouts/DeckLayout.vue` (the deck-shell task), so this entry only supplies
// what that layout expects from its parent.
//
// `DeckLayout` reads `useLayout()`, which `App.vue` deliberately does not
// provide (the root component stays free of store/account imports). Providing
// the context here keeps the deck shell working without touching App.vue.
import { provide } from "vue";
import { createLayoutContext, LAYOUT_KEY } from "@/router/layout";
import DeckLayout from "./layouts/DeckLayout.vue";

provide(LAYOUT_KEY, createLayoutContext());
</script>
