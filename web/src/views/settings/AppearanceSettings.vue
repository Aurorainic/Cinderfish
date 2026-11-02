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
import { applyFontScale, FONT_SCALES, fontScale, setFontScale } from "./font-scale";
import { defaultStore } from "../../store/firefish-store";

type IconName = InstanceType<typeof AppIcon>["$props"]["name"];

const themes: { id: ThemePreference; label: string; icon: IconName }[] = [
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
      <h2 class="card-title">显示</h2>
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

.switch-list {
  display: flex;
  flex-direction: column;
  margin-top: var(--space-8);
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

.sample {
  min-width: 2ch;
  line-height: 1.2;
  text-align: center;
}
</style>
