<script setup lang="ts">
import PagePlaceholder from "../components/PagePlaceholder.vue";
</script>

<template>
  <PagePlaceholder icon="search" title="页面不存在" description="这个地址没有对应的页面。">
    <RouterLink to="/" class="button small" style="margin-top: 10px">回到时间线</RouterLink>
  </PagePlaceholder>
</template>
