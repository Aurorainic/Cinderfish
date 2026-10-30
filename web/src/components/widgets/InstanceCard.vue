<script setup lang="ts">
import InstanceWidget from "./InstanceWidget.vue";
import LinksWidget from "./LinksWidget.vue";

// Kept for the shell's current `#widgets` fallback; the stack replaces it once
// the layout renders WidgetStack. Both cards now exist as registry widgets.
</script>

<template>
  <InstanceWidget />
  <LinksWidget />
</template>
