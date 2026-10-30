<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import {
  MAX_MOBILE_NAV,
  NAV_ITEM_DEFS,
  NAV_ITEM_KEYS,
  isNavItemVisible,
  type NavKey,
} from "../../nav";
import {
  navPrefs,
  resetNav,
  setDesktopNav,
  setMobileNav,
  toggleDesktopItem,
  toggleMobileItem,
} from "../../stores/nav";
import SettingsPage from "./SettingsPage.vue";

// Two lists share one editor: the rail on wide screens and the bottom bar on
// narrow ones. Order is explicit up/down rather than drag, the same choice the
// widget editor makes.
type Target = "desktop" | "mobile";

const descriptions: Record<NavKey, string> = {
  timeline: "首页信息流",
  notifications: "回复、反应与关注",
  explore: "本地与全球时间线",
  settings: "设置入口",
  admin: "仅管理员可见",
};

const desktopVisible = computed(() => navPrefs.value.desktop.filter(isNavItemVisible));
const mobileVisible = computed(() => navPrefs.value.mobile.filter(isNavItemVisible));

const desktopHidden = computed(() =>
  NAV_ITEM_KEYS.filter((key) => isNavItemVisible(key) && !navPrefs.value.desktop.includes(key)),
);
const mobileHidden = computed(() =>
  NAV_ITEM_KEYS.filter((key) => isNavItemVisible(key) && !navPrefs.value.mobile.includes(key)),
);

// A hide that empties the list would leave the rail or the bar blank, and the
// mobile bar is capped, so both limits are counted over the items this account
// can actually see.
const desktopLocked = computed(() => desktopVisible.value.length <= 1);
const mobileLocked = computed(() => mobileVisible.value.length <= 1);
const mobileFull = computed(() => mobileVisible.value.length >= MAX_MOBILE_NAV);

function listOf(target: Target): NavKey[] {
  return target === "desktop" ? navPrefs.value.desktop : navPrefs.value.mobile;
}

// Writes only what is rendered, so an admin-only entry kept by a regular
// user's stored preference cannot hold a slot the mobile bar would count.
function apply(target: Target, keys: readonly NavKey[]): void {
  const visible = keys.filter(isNavItemVisible);
  if (target === "desktop") setDesktopNav(visible);
  else setMobileNav(visible);
}

// Moves past entries the current account cannot see, so an admin-only item
// hidden from a regular user is neither rendered nor reordered.
function move(target: Target, key: NavKey, delta: number): void {
  const list = [...listOf(target)];
  const from = list.indexOf(key);
  if (from < 0) return;
  const step = delta < 0 ? -1 : 1;
  let to = from + step;
  while (to >= 0 && to < list.length && !isNavItemVisible(list[to])) to += step;
  if (to < 0 || to >= list.length) return;
  list.splice(from, 1);
  list.splice(to, 0, key);
  apply(target, list);
}

function toggle(target: Target, key: NavKey): void {
  if (target === "desktop") toggleDesktopItem(key);
  else toggleMobileItem(key);
}
</script>

<template>
  <SettingsPage>
    <section class="card">
      <h2 class="card-title">导航</h2>
      <p class="muted small">
        选择侧栏与底部栏里出现哪些入口、以什么顺序出现。设置只保存在这台设备上。
      </p>

      <div class="section">
        <h3 class="section-title">桌面侧栏</h3>
        <ul class="rows">
          <li v-for="(key, index) in desktopVisible" :key="key" class="row">
            <span class="row-icon"><AppIcon :name="NAV_ITEM_DEFS[key].icon" /></span>
            <span class="row-text">
              <span class="row-label">{{ NAV_ITEM_DEFS[key].label }}</span>
              <span class="row-desc">{{ descriptions[key] }}</span>
            </span>
            <span class="row-actions">
              <button
                type="button"
                class="row-button"
                title="上移"
                aria-label="上移"
                :disabled="index === 0"
                @click="move('desktop', key, -1)"
              >
                <span class="arrow up"><AppIcon name="arrowLeft" /></span>
              </button>
              <button
                type="button"
                class="row-button"
                title="下移"
                aria-label="下移"
                :disabled="index >= desktopVisible.length - 1"
                @click="move('desktop', key, 1)"
              >
                <span class="arrow down"><AppIcon name="arrowLeft" /></span>
              </button>
              <button
                type="button"
                class="row-button danger"
                title="隐藏"
                aria-label="隐藏"
                :disabled="desktopLocked"
                @click="toggle('desktop', key)"
              >
                <AppIcon name="close" />
              </button>
            </span>
          </li>
        </ul>

        <div v-if="desktopHidden.length" class="hidden-row">
          <span class="hidden-label">已隐藏</span>
          <button
            v-for="key in desktopHidden"
            :key="key"
            type="button"
            class="chip"
            :title="`显示${NAV_ITEM_DEFS[key].label}`"
            @click="toggle('desktop', key)"
          >
            <AppIcon name="plus" />
            {{ NAV_ITEM_DEFS[key].label }}
          </button>
        </div>
      </div>

      <div class="section">
        <h3 class="section-title">移动底部栏</h3>
        <p class="muted small">
          窄屏下只显示这些入口，最多 {{ MAX_MOBILE_NAV }} 项。
        </p>
        <ul class="rows">
          <li v-for="(key, index) in mobileVisible" :key="key" class="row">
            <span class="row-icon"><AppIcon :name="NAV_ITEM_DEFS[key].icon" /></span>
            <span class="row-text">
              <span class="row-label">{{ NAV_ITEM_DEFS[key].label }}</span>
              <span class="row-desc">{{ descriptions[key] }}</span>
            </span>
            <span class="row-actions">
              <button
                type="button"
                class="row-button"
                title="上移"
                aria-label="上移"
                :disabled="index === 0"
                @click="move('mobile', key, -1)"
              >
                <span class="arrow up"><AppIcon name="arrowLeft" /></span>
              </button>
              <button
                type="button"
                class="row-button"
                title="下移"
                aria-label="下移"
                :disabled="index >= mobileVisible.length - 1"
                @click="move('mobile', key, 1)"
              >
                <span class="arrow down"><AppIcon name="arrowLeft" /></span>
              </button>
              <button
                type="button"
                class="row-button danger"
                title="隐藏"
                aria-label="隐藏"
                :disabled="mobileLocked"
                @click="toggle('mobile', key)"
              >
                <AppIcon name="close" />
              </button>
            </span>
          </li>
        </ul>

        <div v-if="mobileHidden.length" class="hidden-row">
          <span class="hidden-label">已隐藏</span>
          <button
            v-for="key in mobileHidden"
            :key="key"
            type="button"
            class="chip"
            :disabled="mobileFull"
            :title="mobileFull ? `最多 ${MAX_MOBILE_NAV} 项` : `显示${NAV_ITEM_DEFS[key].label}`"
            @click="toggle('mobile', key)"
          >
            <AppIcon name="plus" />
            {{ NAV_ITEM_DEFS[key].label }}
          </button>
        </div>
      </div>

      <button type="button" class="button ghost small reset" @click="resetNav">恢复默认</button>
    </section>
  </SettingsPage>
</template>

<style scoped>
.section {
  margin-top: var(--space-16);
  padding-top: var(--space-14);
  border-top: 1px solid var(--divider);
}

.section-title {
  margin-bottom: var(--space-8);
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--weight-bold);
}

.rows {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  margin: var(--space-10) 0 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-8) var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel-2);
}

.row-icon {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--accent);
}

.row-icon .icon {
  width: 16px;
  height: 16px;
}

.row-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.row-label {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  line-height: 1.35;
}

.row-desc {
  overflow: hidden;
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  flex: none;
  gap: var(--space-2);
}

.row-button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-weak);
  cursor: pointer;
  transition: color var(--dur-base) ease, border-color var(--dur-base) ease,
    opacity var(--dur-base) ease;
}

.row-button:hover:not(:disabled) {
  border-color: var(--border);
  color: var(--accent);
}

.row-button.danger:hover:not(:disabled) {
  color: var(--danger);
}

.row-button:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.row-button:disabled {
  opacity: 0.35;
  cursor: default;
}

.row-button .icon {
  width: 13px;
  height: 13px;
}

.arrow {
  display: grid;
  place-items: center;
}

.arrow.up {
  transform: rotate(90deg);
}

.arrow.down {
  transform: rotate(-90deg);
}

.hidden-row {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  flex-wrap: wrap;
  margin-top: var(--space-10);
}

.hidden-label {
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-semibold);
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-10);
  border: 1px dashed var(--border);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-weak);
  font-size: var(--fs-xs);
  font-weight: var(--weight-semibold);
  cursor: pointer;
  transition: color var(--dur-base) ease, border-color var(--dur-base) ease;
}

.chip:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
}

.chip:disabled {
  opacity: 0.4;
  cursor: default;
}

.chip .icon {
  width: 12px;
  height: 12px;
}

.reset {
  margin-top: var(--space-16);
}
</style>
