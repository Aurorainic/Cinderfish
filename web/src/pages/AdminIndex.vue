<template>
  <div class="cf-page-placeholder">
    <p>TODO: port AdminIndex</p>
    <RouterView />
  </div>
</template>

<script setup lang="ts">
import { RouterView } from "vue-router";

defineOptions({ name: "CfPageAdminIndex" });
</script>

<style lang="scss" scoped>
.cf-page-placeholder {
  padding: 32px;
  text-align: center;
  opacity: 0.7;
}
</style>
