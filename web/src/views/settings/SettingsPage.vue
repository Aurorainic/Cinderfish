<template>
  <div class="settings-page">
    <slot />
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  min-width: 0;
}
</style>
