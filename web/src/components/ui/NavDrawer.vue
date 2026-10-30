<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import AppIcon from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";
import { DRAWER_FOOTER_KEYS, NAV_ITEM_KEYS, navItemsFor } from "../../nav";
import { openPostForm } from "../../os";
import type { MeDetailed } from "../../api/types";
import type { ShellMode } from "../../stores/shell";

// The full-height navigation drawer behind the mobile bar's first key and the
// rail's collapsed state. AppShell owns the open flag, the account, the shell
// mode and the theme; this component renders and emits, and traps focus while
// open. Mount it once with `v-model:open` (or `:open` + `@update:open`).
const props = withDefaults(
  defineProps<{
    open: boolean;
    me?: MeDetailed | null;
    mode?: ShellMode;
    theme?: string;
  }>(),
  { mode: "universal", theme: "dark" },
);

const emit = defineEmits<{
  "update:open": [value: boolean];
  "toggle-theme": [];
  "toggle-mode": [];
  signout: [];
}>();

const route = useRoute();
const panel = ref<HTMLElement | null>(null);

const mainEntries = computed(() =>
  navItemsFor(NAV_ITEM_KEYS.filter((key) => !DRAWER_FOOTER_KEYS.includes(key))),
);
const footerEntries = computed(() => navItemsFor(DRAWER_FOOTER_KEYS));
const profileTo = computed(() => (props.me ? `/@${props.me.username}` : ""));
const isDark = computed(() => props.theme !== "light");

function close(): void {
  emit("update:open", false);
}

function startPost(): void {
  close();
  openPostForm();
}

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function focusableElements(): HTMLElement[] {
  if (!panel.value) return [];
  return Array.from(panel.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") {
    event.preventDefault();
    close();
    return;
  }
  if (event.key !== "Tab") return;
  const list = focusableElements();
  if (!list.length) {
    event.preventDefault();
    panel.value?.focus();
    return;
  }
  const first = list[0];
  const last = list[list.length - 1];
  const active = document.activeElement;
  const outside = !panel.value || !panel.value.contains(active);
  if (event.shiftKey) {
    if (outside || active === first) {
      event.preventDefault();
      last.focus();
    }
    return;
  }
  if (outside || active === last) {
    event.preventDefault();
    first.focus();
  }
}

let restoreFocus: HTMLElement | null = null;
let previousOverflow = "";

function teardown(): void {
  document.removeEventListener("keydown", onKeydown, true);
  document.body.style.overflow = previousOverflow;
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previousOverflow = document.body.style.overflow;
      restoreFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      document.addEventListener("keydown", onKeydown, true);
      document.body.style.overflow = "hidden";
      await nextTick();
      panel.value?.focus();
      return;
    }
    teardown();
    restoreFocus?.focus();
    restoreFocus = null;
  },
  { immediate: true },
);

watch(
  () => route.fullPath,
  () => {
    if (props.open) close();
  },
);

onBeforeUnmount(teardown);
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="drawer-layer">
        <div class="drawer-scrim" @click="close" />
        <aside
          ref="panel"
          class="drawer"
          role="dialog"
          aria-modal="true"
          aria-label="导航"
          tabindex="-1"
        >
          <header class="drawer-head">
            <RouterLink v-if="me" :to="profileTo" class="account">
              <UserAvatar :user="me" :size="40" />
              <span class="account-text">
                <span class="account-name">{{ me.name || me.username }}</span>
                <span class="account-handle">@{{ me.username }}</span>
              </span>
            </RouterLink>
            <div v-else class="account">
              <span class="account-text">
                <span class="account-name">未登录</span>
                <span class="account-handle">登录后查看时间线</span>
              </span>
            </div>
            <button class="close-button" type="button" aria-label="关闭导航" @click="close">
              <AppIcon name="close" />
            </button>
          </header>

          <nav class="drawer-nav" aria-label="导航">
            <div v-if="!me" class="drawer-guest">
              <RouterLink to="/login" class="guest-button">登录</RouterLink>
              <RouterLink to="/signup" class="guest-button guest-button--primary">注册</RouterLink>
            </div>
            <button v-if="me" class="drawer-item drawer-compose" type="button" @click="startPost">
              <span class="drawer-mark"><AppIcon name="plus" /></span>
              <span class="drawer-label">发帖</span>
            </button>
            <RouterLink
              v-for="entry in mainEntries"
              :key="entry.key"
              :to="entry.def.to"
              class="drawer-item"
            >
              <span class="drawer-icon">
                <AppIcon :name="entry.def.icon" />
                <span v-if="entry.badge" class="drawer-badge">
                  {{ entry.badge > 99 ? "99+" : entry.badge }}
                </span>
              </span>
              <span class="drawer-label">{{ entry.def.label }}</span>
            </RouterLink>
          </nav>

          <div class="drawer-foot">
            <RouterLink
              v-for="entry in footerEntries"
              :key="entry.key"
              :to="entry.def.to"
              class="drawer-item"
            >
              <span class="drawer-icon"><AppIcon :name="entry.def.icon" /></span>
              <span class="drawer-label">{{ entry.def.label }}</span>
            </RouterLink>
            <button class="drawer-item" type="button" @click="emit('toggle-theme')">
              <span class="drawer-icon"><AppIcon :name="isDark ? 'sun' : 'moon'" /></span>
              <span class="drawer-label">{{ isDark ? "切换到浅色" : "切换到深色" }}</span>
            </button>
            <button class="drawer-item" type="button" @click="emit('toggle-mode')">
              <span class="drawer-label drawer-label--single">
                {{ mode === "deck" ? "切换到通用视图" : "切换到 Deck 视图" }}
              </span>
              <span class="drawer-pill">{{ mode === "deck" ? "Deck" : "通用" }}</span>
            </button>
            <button
              v-if="me"
              class="drawer-item drawer-signout"
              type="button"
              @click="emit('signout')"
            >
              <span class="drawer-icon"><AppIcon name="logout" /></span>
              <span class="drawer-label">退出登录</span>
            </button>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-layer {
  position: fixed;
  inset: 0;
  z-index: 950;
  display: flex;
}

.drawer-scrim {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}

.drawer {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(340px, 86vw);
  height: 100%;
  background: var(--bg-raised);
  border-right: 1px solid var(--border);
  box-shadow: var(--shadow-sm);
  outline: none;
}

.drawer-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 12px 12px 14px;
  border-bottom: 1px solid var(--divider);
}

.account {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 4px 6px;
  border-radius: var(--radius-sm);
  color: var(--text);
}

a.account:hover {
  background: var(--panel-2);
}

.account-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.3;
}

.account-name {
  font-size: 0.92rem;
  font-weight: 700;
  color: var(--text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-handle {
  font-size: 0.78rem;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.close-button {
  display: grid;
  flex: none;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}

.close-button:hover {
  background: var(--panel-2);
  color: var(--text);
}

.close-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.drawer-nav {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  padding: 10px 8px;
  overflow-y: auto;
}

.drawer-guest {
  display: flex;
  gap: 8px;
  padding: 4px 4px 10px;
}

.guest-button {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  padding: 9px 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel);
  color: var(--text);
  font-size: 0.9rem;
  font-weight: 700;
}

.guest-button:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.guest-button--primary {
  border-color: var(--accent);
  background: var(--accent);
  color: var(--on-accent);
}

.guest-button--primary:hover {
  background: var(--accent-strong);
  color: var(--on-accent);
}

.drawer-foot {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 8px calc(10px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--divider);
}

.drawer-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 42px;
  padding: 8px 12px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 0.92rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}

.drawer-item:hover {
  background: var(--panel-2);
  color: var(--text);
}

.drawer-item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.drawer-item.router-link-active {
  background: var(--accent-soft);
  color: var(--accent);
}

.drawer-icon {
  position: relative;
  display: inline-flex;
  flex: none;
  color: inherit;
}

.drawer-icon .icon {
  width: 20px;
  height: 20px;
}

.drawer-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.drawer-label--single {
  flex: 1;
  min-width: 0;
}

.drawer-pill {
  flex: none;
  padding: 2px 9px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--muted);
}

.drawer-badge {
  position: absolute;
  top: -6px;
  right: -10px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 0.62rem;
  font-weight: 800;
  line-height: 16px;
  text-align: center;
}

.drawer-compose {
  margin-bottom: 6px;
  color: var(--text);
}

.drawer-mark {
  display: grid;
  flex: none;
  place-items: center;
  width: 26px;
  height: 26px;
  margin-left: -3px;
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
}

.drawer-mark .icon {
  width: 17px;
  height: 17px;
}

.drawer-signout:hover {
  color: var(--danger);
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.2s ease;
}

.drawer-enter-active .drawer,
.drawer-leave-active .drawer {
  transition: transform 0.22s ease;
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .drawer,
.drawer-leave-to .drawer {
  transform: translateX(-100%);
}

@media (prefers-reduced-motion: reduce) {
  .drawer-enter-active,
  .drawer-leave-active,
  .drawer-enter-active .drawer,
  .drawer-leave-active .drawer {
    transition: none;
  }
}
</style>
