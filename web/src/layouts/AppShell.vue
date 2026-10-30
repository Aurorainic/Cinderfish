<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import MobileNav from "../components/ui/MobileNav.vue";
import NavDrawer from "../components/ui/NavDrawer.vue";
import UserAvatar from "../components/UserAvatar.vue";
import WidgetsDrawer from "../components/widgets/WidgetsDrawer.vue";
import WidgetStack from "../components/widgets/WidgetStack.vue";
import { navItemsFor } from "../nav";
import { openPostForm } from "../os";
import { instance, loadInstance } from "../stores/instance";
import { navPrefs } from "../stores/nav";
import { refreshSession, session, signout } from "../stores/session";
import { type ShellMode, setShellMode } from "../stores/shell";
import { streaming } from "../stores/streaming";
import { theme, toggleTheme } from "../stores/theme";

const props = withDefaults(defineProps<{ mode?: ShellMode }>(), { mode: "universal" });

const router = useRouter();
const drawerOpen = ref(false);

// Reads the registry's show()/badge() inside the computed, so an admin item or
// an unread count appears without the rail knowing which item carries one.
const nav = computed(() => navItemsFor(navPrefs.value.desktop));

function handleSignout() {
  drawerOpen.value = false;
  signout();
  router.push("/");
}

// The layout switch is a preference, not a route: deck mode keeps the same
// URL and only swaps what the main area renders.
function toggleShellMode() {
  drawerOpen.value = false;
  setShellMode(props.mode === "deck" ? "universal" : "deck");
}

onMounted(() => {
  refreshSession();
  loadInstance();
  streaming.start();
});

// A new token changes who the stream speaks for, so reconnect on it.
watch(
  () => session.token,
  () => streaming.restart(),
);
</script>

<template>
  <div class="shell" :data-shell="props.mode">
    <aside class="rail">
      <RouterLink to="/" class="brand" aria-label="Cinderfish 首页">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">{{ instance.meta?.name ?? "Cinderfish" }}</span>
      </RouterLink>

      <!-- The one post entry that survives in every layout: deck columns hide
           their pinned composer, so the rail carries the global one. -->
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

      <div class="rail-foot">
        <RouterLink v-if="session.me" :to="`/@${session.me.username}`" class="account-chip" :title="session.me.username">
          <UserAvatar :user="session.me" :size="34" />
          <span class="account-name">{{ session.me.username }}</span>
        </RouterLink>
        <button
          class="rail-mode"
          type="button"
          :aria-pressed="props.mode === 'deck'"
          :title="props.mode === 'deck' ? '切换到通用视图' : '切换到 Deck 视图'"
          @click="toggleShellMode"
        >
          {{ props.mode === "deck" ? "通用" : "Deck" }}
        </button>
        <button class="icon-button" type="button" :title="theme === 'dark' ? '切换浅色' : '切换深色'" @click="toggleTheme">
          <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
        </button>
        <button v-if="session.me" class="icon-button" type="button" title="退出登录" @click="handleSignout">
          <AppIcon name="logout" />
        </button>
      </div>
    </aside>

    <main class="main">
      <slot />
    </main>

    <aside class="widgets">
      <slot name="widgets"><WidgetStack /></slot>
    </aside>

    <WidgetsDrawer />
    <MobileNav @menu="drawerOpen = true" />

    <NavDrawer
      v-model:open="drawerOpen"
      :me="session.me"
      :mode="props.mode"
      :theme="theme"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />
  </div>
</template>

<style scoped>
/* Rail collapse is a pure media-query state: the full rail down to 1280px,
   an icon-only rail below it. No resize listener, nothing to keep in sync
   with the viewport in JS. */
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

/* Deck/universal switch. Labelled rather than icon-only so the two layouts
   stay distinguishable once the rail collapses; the label is short enough for
   the icon-only rail. */
.rail-mode {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  color: var(--muted);
  font: inherit;
  font-size: 0.78rem;
  font-weight: 700;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.rail-mode:hover,
.rail-mode[aria-pressed="true"] {
  color: var(--accent);
  border-color: var(--accent);
}

.rail-mode:focus-visible {
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

/* The rail scrolls on its own when the account block and the nav no longer fit
   the viewport, instead of pushing the document taller. */
.rail {
  overflow-y: auto;
  overscroll-behavior: contain;
}

/* Widget column container; the default content is the widget stack. It scrolls
   independently of the page so a tall stack never stretches the shell. */
.widgets {
  position: sticky;
  top: var(--space-24);
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  max-height: calc(100dvh - var(--space-48));
  overflow-y: auto;
  overscroll-behavior: contain;
}

@media (max-width: 1279px) {
  .shell {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr) var(--side-width);
    gap: var(--space-20);
  }

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

  .rail-foot {
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 0;
  }

  .account-chip {
    padding: 3px;
    border-radius: 50%;
  }

  .account-name {
    display: none;
  }

  .rail-mode {
    width: 40px;
    padding: 6px 0;
    font-size: 0.7rem;
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

@media (max-width: 1099px) {
  .shell {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr);
  }

  .widgets {
    display: none;
  }
}

@media (max-width: 720px) {
  .shell {
    display: block;
    padding: var(--space-12) var(--space-12)
      calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
  }

  .rail {
    display: none;
  }

  .main {
    padding-bottom: 0;
  }
}

/* Deck mode gives the widget column's width back to the horizontal column
   strip; the rail keeps its own width so the icon state below 1280px still
   applies. Kept last so its higher specificity wins over the breakpoints. */
.shell[data-shell="deck"] {
  grid-template-columns: var(--rail-width) minmax(0, 1fr);
  height: 100dvh;
  min-height: 0;
  overflow: hidden;
}

/* The deck strip owns the scrolling, so the main area stops being a scrolling
   document: it becomes a flex column whose child grows into the remaining
   height. This is the whole height chain — nothing below divides the viewport
   by hand. */
.shell[data-shell="deck"] .main {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.shell[data-shell="deck"] .rail {
  height: calc(100dvh - var(--space-48));
}

.shell[data-shell="deck"] .widgets {
  display: none;
}

@media (max-width: 1279px) {
  .shell[data-shell="deck"] {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr);
  }
}

/* On phones the shell stacks into a column for the deck strip; the bottom bar
   floats over the reserved padding, so the strip stops above it. */
@media (max-width: 720px) {
  .shell[data-shell="deck"] {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    padding: var(--space-12) var(--space-12)
      calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
  }
}
</style>
