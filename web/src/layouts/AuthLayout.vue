<script setup lang="ts">
import AppIcon from "../components/AppIcon.vue";
</script>

<template>
  <div class="auth">
    <section class="auth-panel">
      <RouterLink to="/" class="brand" style="margin-bottom: 32px">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">Cinderfish</span>
      </RouterLink>
      <slot />
    </section>
    <aside class="auth-aside">
      <span class="instance-logo" style="width: 56px; height: 56px"><AppIcon name="flame" /></span>
      <h1>在自己的实例上，安静地说话。</h1>
      <p class="muted">
        Cinderfish 是一个个人或极小规模的联邦社交实例。它说 Misskey 的 API，用 ActivityPub 和外面的世界通信。
      </p>
    </aside>
  </div>
</template>
