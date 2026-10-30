<script setup lang="ts">
import { computed, onMounted } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import {
  setThemePreference,
  systemTheme,
  theme,
  themePreference,
  toggleTheme,
  type ThemePreference,
} from "../../stores/theme";
import SettingsPage from "./SettingsPage.vue";
import { applyFontScale, FONT_SCALES, fontScale, setFontScale } from "./font-scale";

const themes: { id: ThemePreference; label: string; icon: "display" | "sun" | "moon" }[] = [
  { id: "auto", label: "跟随系统", icon: "display" },
  { id: "light", label: "浅色", icon: "sun" },
  { id: "dark", label: "深色", icon: "moon" },
];

const systemLabel = computed(() => (systemTheme() === "dark" ? "深色" : "浅色"));

// The scale is applied on demand: the preference belongs to the device, and
// the appearance page is where a reader expects to see it take effect.
onMounted(() => {
  applyFontScale();
});
</script>

<template>
  <SettingsPage>
    <section class="card">
      <h2 class="card-title">主题</h2>
      <p class="muted small">
        选择界面主题，选择会保存在这台设备上。跟随系统时，系统切换深浅色会立即生效。
      </p>
      <div class="theme-options">
        <button
          v-for="option in themes"
          :key="option.id"
          type="button"
          class="theme-option"
          :class="{ active: themePreference === option.id }"
          :aria-pressed="themePreference === option.id"
          @click="setThemePreference(option.id)"
        >
          <AppIcon :name="option.icon" />
          <span>{{ option.label }}</span>
          <AppIcon v-if="themePreference === option.id" name="check" class="tick" />
        </button>
        <button type="button" class="theme-option switch" @click="toggleTheme()">
          <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
          <span>切换为{{ theme === "dark" ? "浅色" : "深色" }}</span>
        </button>
      </div>
      <p v-if="themePreference === 'auto'" class="muted small hint">
        当前系统为{{ systemLabel }}，界面正在跟随。
      </p>
    </section>

    <section class="card">
      <h2 class="card-title">字体大小</h2>
      <p class="muted small">调整界面文字的基准大小，同样只保存在这台设备上。</p>
      <div class="theme-options">
        <button
          v-for="option in FONT_SCALES"
          :key="option.id"
          type="button"
          class="theme-option"
          :class="{ active: fontScale === option.id }"
          :aria-pressed="fontScale === option.id"
          @click="setFontScale(option.id)"
        >
          <span class="sample" :style="{ fontSize: option.size }">{{ option.sample }}</span>
          <AppIcon v-if="fontScale === option.id" name="check" class="tick" />
        </button>
      </div>

      <p class="preview">
        这条示例文字使用当前的字体大小显示。昵称、正文与按钮会随之变化，图标和间距保持不变。
      </p>
    </section>
  </SettingsPage>
</template>

<style scoped>
.theme-options {
  display: flex;
  gap: var(--space-10);
  margin-top: var(--space-12);
  flex-wrap: wrap;
}

.theme-option {
  display: inline-flex;
  align-items: center;
  gap: var(--space-8);
  padding: var(--space-8) var(--space-16);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--panel-2);
  color: var(--text-weak);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: border-color var(--dur-base) ease, color var(--dur-base) ease,
    background var(--dur-base) ease;
}

.theme-option:hover {
  border-color: var(--accent);
  color: var(--text);
}

.theme-option:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.theme-option.active {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent);
}

.theme-option .icon {
  width: 16px;
  height: 16px;
}

.theme-option .tick {
  width: 15px;
  height: 15px;
}

.theme-option.switch {
  border-style: dashed;
}

.hint {
  margin-top: var(--space-10);
}

.sample {
  min-width: 2ch;
  line-height: 1.2;
  text-align: center;
}

.preview {
  margin-top: var(--space-16);
  padding: var(--space-14);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
  color: var(--text-weak);
  font-size: var(--fs-md);
  line-height: var(--leading-relaxed);
}
</style>
