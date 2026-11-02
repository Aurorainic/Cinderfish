<script setup lang="ts">
import MkInfo from "../../components/ui/MkInfo.vue";

withDefaults(defineProps<{ title?: string; tone?: "danger" | "warn" }>(), {
  title: "",
  tone: "danger",
});
</script>

<template>
  <MkInfo :warn="tone === 'danger'" :icon="tone === 'warn' ? 'ph-warning' : 'ph-info'" class="callout">
    <div class="callout-body">
      <p v-if="title" class="callout-title">{{ title }}</p>
      <slot />
    </div>
  </MkInfo>
</template>

<style scoped>
.callout-body {
  flex: 1;
  min-inline-size: 0;
}

.callout-title {
  margin-block-end: 4px;
  font-weight: bold;
}

.callout-body :deep(p + p) {
  margin-block-start: 6px;
}

.callout-body :deep(.callout-actions) {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-block-start: 10px;
}
</style>
