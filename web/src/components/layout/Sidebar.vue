<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";
import { navItemsFor } from "../../nav";
import { openPostForm } from "../../os";
import { instance } from "../../stores/instance";
import { navPrefs } from "../../stores/nav";
import { session } from "../../stores/session";
import type { ShellMode } from "../../stores/shell";
import { theme } from "../../stores/theme";

// The desktop navigation column, translated from fork/ui/_common_/navbar.vue:
// brand on top, the post button, the registry-driven nav list, and the account
// block at the bottom. Widths come from the layout tokens (250px full, 80px
// icon-only below 1280px, matching the fork's $nav-width pair). Navigation
// itself is delegated to RouterLink, so the only emitted events are shell
// actions; the store's show()/badge() are read inside the computed so an
// admin entry or an unread count shows up without this file knowing which
// item carries one.
defineProps<{ mode: ShellMode }>();

const emit = defineEmits<{
  "toggle-theme": [];
  "toggle-mode": [];
  signout: [];
}>();

const nav = computed(() => navItemsFor(navPrefs.value.desktop));
</script>

<template>
  <aside class="sidebar">
    <RouterLink to="/" class="brand" aria-label="Cinderfish 首页">
      <span class="brand-mark"><AppIcon name="flame" /></span>
      <span class="brand-name">{{ instance.meta?.name ?? "Cinderfish" }}</span>
    </RouterLink>

    <!-- The one post entry that survives in every layout: deck columns hide
         their pinned composer, so the sidebar carries the global one. -->
    <button v-if="session.me" class="compose" type="button" title="发帖" @click="openPostForm()">
      <AppIcon name="plus" />
      <span class="compose-label">发帖</span>
    </button>

    <nav class="nav" aria-label="主导航">
      <RouterLink
        v-for="item in nav"
        :key="item.key"
        :to="item.def.to"
        class="nav-link"
        :title="item.def.label"
      >
        <span class="nav-icon">
          <AppIcon :name="item.def.icon" />
          <span v-if="item.badge" class="nav-badge">
            {{ item.badge > 99 ? "99+" : item.badge }}
          </span>
        </span>
        <span class="nav-label">{{ item.def.label }}</span>
      </RouterLink>
    </nav>

    <div class="sidebar-foot">
      <RouterLink v-if="session.me" :to="`/@${session.me.username}`" class="account-chip" :title="session.me.username">
        <UserAvatar :user="session.me" :size="34" />
        <span class="account-name">{{ session.me.username }}</span>
      </RouterLink>
      <button
        class="icon-button mode-toggle"
        type="button"
        :aria-pressed="mode === 'deck'"
        :title="mode === 'deck' ? '切换到通用视图' : '切换到 Deck 视图'"
        @click="emit('toggle-mode')"
      >
        <AppIcon :name="mode === 'deck' ? 'display' : 'columns'" />
      </button>
      <button class="icon-button" type="button" :title="theme === 'dark' ? '切换浅色' : '切换深色'" @click="emit('toggle-theme')">
        <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
      </button>
      <button v-if="session.me" class="icon-button" type="button" title="退出登录" @click="emit('signout')">
        <AppIcon name="logout" />
      </button>
    </div>
  </aside>
</template>

<style scoped>
/* The sidebar scrolls on its own when the account block and the nav no longer
   fit the viewport, instead of pushing the document taller. */
.sidebar {
  position: sticky;
  top: var(--space-24);
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: var(--space-20);
  height: calc(100vh - var(--space-48));
  overflow-y: auto;
  overscroll-behavior: contain;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-14);
  padding: var(--space-10) var(--space-14);
  border-radius: var(--radius-md);
  color: var(--nav-fg);
  font-weight: var(--weight-semibold);
  transition: background var(--dur-base) ease, color var(--dur-base) ease;
}

.nav-link .icon {
  width: 22px;
  height: 22px;
}

.nav-link:hover {
  background: var(--surface-hover);
  color: var(--nav-fg-hover);
}

.nav-link.router-link-active {
  color: var(--nav-active);
  background: var(--surface-selected);
}

.sidebar-foot {
  margin-top: auto;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-8);
  padding: 0 var(--space-6);
}

/* The account chip needs a whole row: avatar, name, the mode switch and the two
   icon buttons do not fit on one line at 250px, and squeezing the name down to
   an ellipsis of one character is worse than a second row. Only applies to the
   full-width sidebar; the icon-only sidebar stacks its footer vertically. */
@media (min-width: 1280px) {
  .sidebar-foot > :not(.account-chip) {
    flex: none;
  }

  .sidebar-foot > .account-chip {
    flex: 1 1 100%;
  }
}

.compose {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border: 0;
  border-radius: var(--radius);
  background: var(--accent);
  color: var(--on-accent);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s ease;
}

.compose:hover {
  background: var(--accent-strong);
}

.compose:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.account-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 12px 3px 3px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  min-width: 0;
}

.account-chip:hover {
  border-color: var(--accent);
}

.account-name {
  font-size: 0.85rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* The deck/universal switch is an icon button like the ones next to it, so
   the sidebar footer keeps one shape. The pressed state carries the accent,
   which is what told the labelled button apart before. */
.mode-toggle[aria-pressed="true"] {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}

.mode-toggle:hover,
.mode-toggle[aria-pressed="true"] {
  color: var(--accent);
  border-color: var(--accent);
}

.mode-toggle:focus-visible {
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
  border-radius: var(--radius-full);
  background: var(--accent);
  color: var(--on-accent);
  font-size: 0.65rem;
  font-weight: 800;
  line-height: 16px;
  text-align: center;
}

/* Icon-only state below 1280px (fork navbar's 1279px threshold): the labels
   drop and everything centers in the 80px column. */
@media (max-width: 1279px) {
  .brand {
    justify-content: center;
    padding: 6px 0;
  }

  .brand-name,
  .nav-label {
    display: none;
  }

  .nav-link {
    justify-content: center;
    padding: 13px 0;
  }

  .sidebar-foot {
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 0;
  }

  .account-chip {
    flex: none;
    padding: 3px;
    border-radius: 50%;
  }

  .account-name {
    display: none;
  }

  .mode-toggle {
    width: 40px;
    height: 40px;
    padding: 0;
  }

  .compose {
    width: 40px;
    height: 40px;
    padding: 0;
    margin: 0 auto;
  }

  .compose-label {
    display: none;
  }
}

/* At the fork's mobile threshold the sidebar leaves entirely; the drawer and
   bottom bar in MobileSidebar take over the same registry. */
@media (max-width: 500px) {
  .sidebar {
    display: none;
  }
}
</style>
