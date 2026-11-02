<script setup lang="ts">
import MkFolder from "../../components/ui/MkFolder.vue";

// One settings group. Firefish renders the settings tabs as a stack of
// MkFolder sections (2fa.vue is the clearest example); the header supplies the
// section title and the body keeps the 16px inner padding the column sets up.
withDefaults(
  defineProps<{
    title: string;
    persistKey?: string;
    expanded?: boolean;
  }>(),
  { persistKey: undefined, expanded: true },
);
</script>

<template>
  <MkFolder :persist-key="persistKey" :expanded="expanded">
    <template #header>{{ title }}</template>
    <div class="body"><slot /></div>
  </MkFolder>
</template>

<style scoped>
.body {
  display: flex;
  flex-direction: column;
  gap: var(--space-10);
  padding-block-end: var(--margin);
}
</style>
