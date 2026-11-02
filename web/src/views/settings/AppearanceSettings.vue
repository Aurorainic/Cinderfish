// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/settings/theme.vue
// (theme mode toggle) and pages/settings/general.vue (appearance + font-size
// sections).
//
// Adaptations: Firefish's boolean darkMode toggle becomes Cinderfish's
// auto/light/dark themeMode slice, written through stores/theme.ts so both
// the store and the legacy paint path stay in step. Theme packages, custom
// CSS, wallpapers and sounds are dropped: Cinderfish ships a single built-in
// token set and has no theme editor or install pipeline.
<script setup lang="ts">
import { computed, onMounted } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import MkSwitch from "./MkSwitch.vue";
import {
  setThemePreference,
  systemTheme,
  theme,
  themePreference,
  toggleTheme,
  type ThemePreference,
} from "../../stores/theme";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";
import { applyFontScale, FONT_SCALES, fontScale, setFontScale } from "./font-scale";
import { defaultStore } from "../../store/firefish-store";

type IconName = InstanceType<typeof AppIcon>["$props"]["name"];

const themes: { id: ThemePreference; label: string; icon: IconName }[] = [
  { id: "auto", label: "自动", icon: "display" },
  { id: "light", label: "浅色", icon: "sun" },
  { id: "dark", label: "深色", icon: "moon" },
];

const systemLabel = computed(() => (systemTheme() === "dark" ? "深色" : "浅色"));

// The scale is applied on demand: the preference belongs to the device, and
// the appearance page is where a reader expects to see it take effect.
onMounted(() => {
  applyFontScale();
});

// Display toggles from general.vue's appearance section, backed by the
// slices the ported note list and avatar actually read. squareAvatars is
// wired but the ported MkAvatar pins rounded corners, so the switch is kept
// out until that TODO lands.
const showGapBetweenNotesInTimeline = defaultStore.makeGetterSetter(
  "showGapBetweenNotesInTimeline",
);
const seperateRenoteQuote = defaultStore.makeGetterSetter("seperateRenoteQuote");
const showFixedPostForm = defaultStore.makeGetterSetter("showFixedPostForm");
const showBigPostButton = defaultStore.makeGetterSetter("showBigPostButton");
const loadRawImages = defaultStore.makeGetterSetter("loadRawImages");
const useBlurEffect = defaultStore.makeGetterSetter("useBlurEffect");
</script>

<template>
  <SettingsPage>
    <SettingsSection title="主题" persist-key="settings-appearance-theme">
      <p class="settings-hint">
        选择界面主题，选择会保存在这台设备上。自动模式会跟随系统，系统切换深浅色时立即生效。
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
      <p v-if="themePreference === 'auto'" class="settings-hint">
        当前系统为{{ systemLabel }}，界面正在跟随。
      </p>
    </SettingsSection>

    <SettingsSection title="显示" persist-key="settings-appearance-display">
      <div class="switch-list">
        <MkSwitch v-model="showGapBetweenNotesInTimeline">
          在时间线的帖子之间留出间隔
        </MkSwitch>
        <MkSwitch v-model="seperateRenoteQuote">分开显示转发与引用</MkSwitch>
        <MkSwitch v-model="showFixedPostForm">固定发帖框</MkSwitch>
        <MkSwitch v-model="showBigPostButton">显示大号发帖按钮</MkSwitch>
        <MkSwitch v-model="loadRawImages">直接加载原始图片</MkSwitch>
        <MkSwitch v-model="useBlurEffect">使用背景模糊效果</MkSwitch>
      </div>
    </SettingsSection>

    <SettingsSection title="字体大小" persist-key="settings-appearance-font">
      <p class="settings-hint">调整界面文字的基准大小，同样只保存在这台设备上。</p>
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
    </SettingsSection>
  </SettingsPage>
</template>

<style scoped>
.theme-options {
  display: flex;
  gap: var(--space-10);
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
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
}

.theme-option:hover {
  border-color: var(--accent);
  color: var(--text);
}

.theme-option:focus-visible {
  outline: 2px solid var(--focus);
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

.switch-list {
  display: flex;
  flex-direction: column;
}

.preview {
  margin: 0;
  padding: var(--space-16);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
  color: var(--text-weak);
  font-size: var(--fs-md);
  line-height: 1.6;
}

.sample {
  min-width: 2ch;
  line-height: 1.2;
  text-align: center;
}
</style>
