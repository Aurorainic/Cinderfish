<template>
  <div class="cf-shell-zen">
    <RouterView />
  </div>
</template>

<script setup lang="ts">
// Ported from Firefish's ui/zen.vue: no chrome at all.
import { RouterView } from "vue-router";
</script>

<style lang="scss" scoped>
.cf-shell-zen {
  min-height: 100vh;
  max-width: 700px;
  margin: 0 auto;
  padding: var(--margin);
  box-sizing: border-box;
}
</style>
