<script setup lang="ts">
import AppIcon from "../components/AppIcon.vue";
</script>

<template>
  <div class="auth">
    <section class="_panel auth-panel">
      <RouterLink to="/" class="brand">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">Cinderfish</span>
      </RouterLink>
      <slot />
    </section>
    <aside class="auth-aside">
      <span class="instance-logo"><AppIcon name="flame" /></span>
      <h1>在自己的实例上，安静地说话。</h1>
      <p class="intro">
        Cinderfish 是一个个人或极小规模的联邦社交实例。它说 Misskey 的 API，用 ActivityPub 和外面的世界通信。
      </p>
    </aside>
  </div>
</template>

<style scoped>
.auth {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 32px;
  min-block-size: 100dvh;
  padding: var(--margin);
  background: var(--bg);
}

.auth-panel {
  inline-size: 100%;
  max-inline-size: 420px;
  padding: 32px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-block-end: 32px;
  color: var(--fgHighlighted);
  font-weight: bold;
  text-decoration: none;
}

.brand-mark {
  display: grid;
  place-items: center;
  inline-size: 40px;
  block-size: 40px;
  border-radius: 50%;
  background: var(--accentedBg);
  color: var(--accent);
}

.brand-mark .icon {
  inline-size: 24px;
  block-size: 24px;
}

.brand-name {
  font-size: 1.2em;
}

.auth-aside {
  max-inline-size: 360px;
}

.instance-logo {
  display: grid;
  place-items: center;
  inline-size: 56px;
  block-size: 56px;
  margin-block-end: 16px;
  border-radius: 50%;
  background: var(--accentedBg);
  color: var(--accent);
}

.instance-logo .icon {
  inline-size: 32px;
  block-size: 32px;
}

.auth-aside h1 {
  margin-block-end: 12px;
  font-size: 1.5em;
  line-height: 1.35;
}

.intro {
  color: var(--fgTransparentWeak);
}

@media (max-width: 760px) {
  .auth {
    flex-direction: column;
    justify-content: flex-start;
    gap: 24px;
    padding-block: 32px;
  }

  .auth-panel {
    padding: 24px;
  }

  .auth-aside {
    order: 2;
  }
}
</style>
