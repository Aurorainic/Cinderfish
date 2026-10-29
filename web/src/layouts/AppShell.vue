<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import MobileNav from "../components/ui/MobileNav.vue";
import UserAvatar from "../components/UserAvatar.vue";
import InstanceCard from "../components/widgets/InstanceCard.vue";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, refreshSession, session, signout } from "../stores/session";
import { type ShellMode, setShellMode } from "../stores/shell";
import { streaming, unreadNotifications } from "../stores/streaming";

const props = withDefaults(defineProps<{ mode?: ShellMode }>(), { mode: "universal" });

const router = useRouter();
const theme = ref(document.documentElement.dataset.theme ?? "dark");

function toggleTheme() {
  theme.value = theme.value === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = theme.value;
  localStorage.setItem("cinderfish.theme", theme.value);
}

const nav = computed(() => {
  const items = [
    { to: "/", icon: "home", label: "时间线" },
    { to: "/notifications", icon: "bell", label: "通知" },
    { to: "/explore", icon: "compass", label: "探索" },
    { to: "/settings", icon: "settings", label: "设置" },
  ] as const;
  return isAdmin.value ? [...items, { to: "/admin", icon: "shield", label: "管理" } as const] : items;
});

function handleSignout() {
  signout();
  router.push("/");
}

// The layout switch is a preference, not a route: deck mode keeps the same
// URL and only swaps what the main area renders.
function toggleShellMode() {
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

      <nav class="nav">
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="nav-link" :title="item.label">
          <span class="nav-icon">
            <AppIcon :name="item.icon" />
            <span v-if="item.to === '/notifications' && unreadNotifications" class="nav-badge">
              {{ unreadNotifications > 99 ? "99+" : unreadNotifications }}
            </span>
          </span>
          <span class="nav-label">{{ item.label }}</span>
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
      <slot name="widgets"><InstanceCard /></slot>
    </aside>

    <MobileNav />
  </div>
</template>

<style scoped>
/* Rail collapse is a pure media-query state: the full rail down to 1280px,
   an icon-only rail below it. No resize listener, nothing to keep in sync
   with the viewport in JS. */
.account-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 3px 12px 3px 3px;
  border: 1px solid var(--border);
  border-radius: 999px;
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
  border-radius: 999px;
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
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 0.65rem;
  font-weight: 800;
  line-height: 16px;
  text-align: center;
}

/* Widget column container; the default content is the InstanceCard stack. */
.widgets {
  position: sticky;
  top: 24px;
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

@media (max-width: 1279px) {
  .shell {
    grid-template-columns: 76px minmax(0, 1fr) var(--side-width);
    gap: 20px;
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
}

@media (max-width: 1099px) {
  .shell {
    grid-template-columns: 76px minmax(0, 1fr);
  }

  .widgets {
    display: none;
  }
}

@media (max-width: 720px) {
  .shell {
    display: block;
    padding: 12px 12px calc(72px + env(safe-area-inset-bottom));
  }

  .rail {
    display: none;
  }

  .main {
    padding-bottom: 0;
  }
}
</style>
