<template>
  <div class="settings-page">
    <slot />
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: var(--margin);
  min-width: 0;
}
</style>

<style>
/* Shared helpers for the settings tab bodies. The pages all render inside
   SettingsPage, so these names stay inside the settings tree in practice. */
.settings-hint {
  margin: 0;
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  line-height: 1.5;
}

.settings-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
