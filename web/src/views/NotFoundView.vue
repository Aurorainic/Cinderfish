<script setup lang="ts">
import PagePlaceholder from "../components/PagePlaceholder.vue";
import { usePageTitle } from "../composables/use-page-title";

usePageTitle("页面不存在");
</script>

<template>
  <PagePlaceholder icon="search" title="页面不存在" description="这个地址没有对应的页面。">
    <div class="actions">
      <RouterLink to="/" class="button small">回到时间线</RouterLink>
      <RouterLink to="/login" class="button ghost small">登录</RouterLink>
    </div>
  </PagePlaceholder>
</template>

<style scoped>
.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-8);
  margin-top: var(--space-6);
}
</style>
