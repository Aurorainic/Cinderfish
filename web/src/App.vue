<template>
  <component :is="shell" />
  <CfPopupHost />
</template>

<script setup lang="ts">
// Root component. The shell (universal / deck / zen / visitor) is injected by
// main.ts through ShellKey; App.vue itself stays free of store and account
// imports so the choice can be made once, at startup, and be lazy.
import { computed, defineAsyncComponent, inject, markRaw, ref } from "vue";
import CfPopupHost from "./os/CfPopupHost.vue";
import { ShellKey, type ShellMode } from "./ui/tokens";

const fallbackShell = defineAsyncComponent(() => import("./ui/UniversalShell.vue"));

const registry = inject(ShellKey, null);
const mode = registry?.mode ?? ref<ShellMode>("universal");

const shell = computed(() => {
  if (registry == null) return markRaw(fallbackShell);
  return markRaw(registry.load(mode.value));
});
</script>
