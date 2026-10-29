<script setup lang="ts">
import AppIcon from "./AppIcon.vue";

defineProps<{ icon: keyof typeof icons; title: string; description?: string }>();

const icons = {
  home: true,
  bell: true,
  compass: true,
  settings: true,
  shield: true,
  feather: true,
  user: true,
  search: true,
} as const;
</script>

<template>
  <div class="placeholder">
    <AppIcon :name="icon" />
    <h2>{{ title }}</h2>
    <p v-if="description" class="small">{{ description }}</p>
    <slot />
  </div>
</template>
