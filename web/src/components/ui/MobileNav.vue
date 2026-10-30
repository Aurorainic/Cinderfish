<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../AppIcon.vue";
import { NAV_ITEM_KEYS, navItemsFor, type NavIcon } from "../../nav";
import { openPostForm } from "../../os";
import { navPrefs } from "../../stores/nav";

// The narrow-viewport navigation. Items come from the shared registry and the
// user's mobile selection; the menu key (first) opens NavDrawer and the post
// key is fixed in the middle. The drawer's open state lives in AppShell, so
// this component only announces the request.
const emit = defineEmits<{ menu: [] }>();

interface BarSlot {
  key: string;
  kind: "menu" | "post" | "link";
  label: string;
  icon: NavIcon;
  to?: string;
  badge?: number;
  dot?: boolean;
}

// Badges for items the bar does not show would otherwise be invisible, so the
// menu key carries a dot when the drawer holds anything unread.
const drawerIndicated = computed(() => {
  const shown = new Set<string>(navPrefs.value.mobile);
  return navItemsFor(NAV_ITEM_KEYS).some((entry) => !shown.has(entry.key) && entry.badge > 0);
});

const slots = computed<BarSlot[]>(() => {
  const links: BarSlot[] = navItemsFor(navPrefs.value.mobile).map((entry) => ({
    key: entry.key,
    kind: "link",
    label: entry.def.label,
    icon: entry.def.icon,
    to: entry.def.to,
    badge: entry.badge,
  }));
  const split = Math.floor(links.length / 2);
  return [
    { key: "menu", kind: "menu", label: "菜单", icon: "more", dot: drawerIndicated.value },
    ...links.slice(0, split),
    { key: "post", kind: "post", label: "发帖", icon: "plus" },
    ...links.slice(split),
  ];
});
</script>

<template>
  <nav class="mobile-nav" aria-label="主导航">
    <template v-for="slot in slots" :key="slot.key">
      <RouterLink v-if="slot.to" :to="slot.to" class="nav-item">
        <span class="nav-icon">
          <AppIcon :name="slot.icon" />
          <span v-if="slot.badge" class="nav-badge">
            {{ slot.badge > 99 ? "99+" : slot.badge }}
          </span>
        </span>
        <span class="nav-label">{{ slot.label }}</span>
      </RouterLink>
      <button
        v-else
        class="nav-item"
        :class="{ 'nav-compose': slot.kind === 'post' }"
        type="button"
        :aria-label="slot.kind === 'menu' ? '打开导航菜单' : slot.label"
        :aria-haspopup="slot.kind === 'menu' ? 'dialog' : undefined"
        @click="slot.kind === 'post' ? openPostForm() : emit('menu')"
      >
        <span v-if="slot.kind === 'post'" class="compose-mark"><AppIcon name="plus" /></span>
        <span v-else-if="slot.kind === 'menu'" class="nav-icon">
          <AppIcon :name="slot.icon" />
          <span v-if="slot.dot" class="nav-dot" />
        </span>
        <AppIcon v-else :name="slot.icon" />
        <span class="nav-label">{{ slot.label }}</span>
      </button>
    </template>
  </nav>
</template>

<style scoped>
.mobile-nav {
  display: none;
}

@media (max-width: 720px) {
  .mobile-nav {
    position: fixed;
    inset: auto 0 0 0;
    z-index: 30;
    display: flex;
    align-items: stretch;
    justify-content: space-around;
    gap: 2px;
    padding: 6px 8px calc(6px + env(safe-area-inset-bottom));
    background: color-mix(in srgb, var(--bg-raised) 94%, transparent);
    border-top: 1px solid var(--border);
    backdrop-filter: blur(12px);
  }
}

.nav-item {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-width: 0;
  padding: 6px 2px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font-size: 0.68rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}

.nav-label {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nav-item .icon {
  width: 21px;
  height: 21px;
}

.nav-item.router-link-active {
  color: var(--accent);
}

.nav-item:hover {
  color: var(--text);
}

.nav-item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.nav-icon {
  position: relative;
  display: inline-flex;
}

.nav-badge {
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

.nav-dot {
  position: absolute;
  top: -2px;
  right: -4px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
}

.compose-mark {
  display: grid;
  place-items: center;
  width: 34px;
  height: 26px;
  margin: -2px 0;
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
}

.compose-mark .icon {
  width: 18px;
  height: 18px;
}

.nav-compose {
  color: var(--text);
}
</style>
