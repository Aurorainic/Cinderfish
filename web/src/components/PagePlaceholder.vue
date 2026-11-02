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
  <div class="_fullinfo page-placeholder">
    <AppIcon class="badge" :name="icon" />
    <h2 class="title">{{ title }}</h2>
    <p v-if="description" class="description">{{ description }}</p>
    <div v-if="$slots.default" class="actions"><slot /></div>
  </div>
</template>

<style scoped>
/* Same shape as Firefish's _fullinfo empty states: centred badge, title and one
   line of explanation, with any action the caller passes below. */
.page-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--fgTransparentWeak);
}

.page-placeholder > .badge {
  inline-size: 64px;
  block-size: 64px;
  opacity: 0.45;
}

.page-placeholder > .title {
  margin: 0;
  font-size: 1.05em;
  color: var(--fg);
}

.page-placeholder > .description {
  margin: 0;
  font-size: 0.95em;
  line-height: 1.5;
}

.page-placeholder > .actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}
</style>
