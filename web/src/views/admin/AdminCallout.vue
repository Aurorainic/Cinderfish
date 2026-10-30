<script setup lang="ts">
withDefaults(defineProps<{ title?: string; tone?: "danger" | "warn" }>(), {
  title: "",
  tone: "danger",
});
</script>

<template>
  <section class="callout" :class="tone">
    <h3 v-if="title" class="callout-title">{{ title }}</h3>
    <div class="callout-body"><slot /></div>
  </section>
</template>

<style scoped>
.callout {
  padding: var(--space-14) var(--space-16);
  border: 1px solid var(--danger-border);
  border-radius: var(--radius-lg);
  background: var(--danger-soft);
  color: var(--text);
}

.callout.warn {
  border-color: color-mix(in srgb, var(--warn) 45%, var(--border));
  background: var(--warn-bg);
}

.callout-title {
  margin-bottom: var(--space-6);
  font-size: var(--fs-sm);
  color: var(--danger);
}

.callout.warn .callout-title {
  color: var(--warn-fg);
}

.callout-body {
  font-size: var(--fs-sm);
  color: var(--text-weak);
}

.callout-body :deep(p + p) {
  margin-top: var(--space-6);
}

.callout-body :deep(.callout-actions) {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
  margin-top: var(--space-10);
}
</style>
