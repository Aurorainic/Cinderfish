<script setup lang="ts">
import { onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import MobileSidebar from "../components/layout/MobileSidebar.vue";
import Sidebar from "../components/layout/Sidebar.vue";
import WidgetPanel from "../components/layout/WidgetPanel.vue";
import WidgetsDrawer from "../components/widgets/WidgetsDrawer.vue";
import { loadInstance } from "../stores/instance";
import { refreshSession, session, signout } from "../stores/session";
import { type ShellMode, setShellMode } from "../stores/shell";
import { streaming, setUnreadNotifications } from "../stores/streaming";
import { toggleTheme } from "../stores/theme";

const props = withDefaults(defineProps<{ mode?: ShellMode }>(), { mode: "universal" });

const router = useRouter();

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

// The unread badge lives in the shared streaming store. Seed it from the
// session payload (/api/i carries unreadNotificationsCount and
// hasUnreadNotification) so a cold load lands on the right number instead of
// zero, and re-seed when the account changes. Live streaming events increment
// the store from there.
watch(
  () => session.me?.id ?? "",
  () => {
    const count = session.me?.unreadNotificationsCount;
    setUnreadNotifications(typeof count === "number" ? count : 0);
  },
  { immediate: true },
);
</script>

<template>
  <!-- The Iceshrimp universal composition (fork/ui/universal.vue): sidebar,
       main region, widget column. Deck mode keeps the sidebar and hands the
       widget column's width to the horizontal strip, so WidgetPanel only
       mounts in universal mode. -->
  <div class="shell" :data-shell="props.mode">
    <Sidebar
      :mode="props.mode"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />

    <main class="main">
      <slot />
    </main>

    <WidgetPanel v-if="props.mode === 'universal'" />

    <!-- Below 1090px the panel is hidden and the same stack moves into this
         drawer behind its floating fab; the fab hides itself above 1090px, so
         widget access survives every width. -->
    <WidgetsDrawer />

    <!-- Below 500px the sidebar is gone and the same registry renders through
         the drawer + bottom bar. -->
    <MobileSidebar
      :mode="props.mode"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />
  </div>
</template>

<style scoped>
/* Three-region grid: sidebar (250px, 80px below 1280px) | main | widgets
   (300px, hidden below 1090px by the panel itself). */
.shell {
  display: grid;
  grid-template-columns: var(--rail-width) minmax(0, 1fr) var(--side-width);
  gap: var(--space-28);
  max-width: var(--shell-max-width);
  margin: 0 auto;
  padding: var(--space-24);
  min-height: 100vh;
}

.main {
  min-width: 0;
  padding-bottom: var(--space-64);
}

@media (max-width: 1279px) {
  .shell {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr) var(--side-width);
    gap: var(--space-20);
  }
}

@media (max-width: 1090px) {
  .shell {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr);
  }
}

/* The fork's mobile threshold: from here down the sidebar disappears and the
   bottom bar plus drawer take over; the reserved padding keeps the page above
   the fixed bar, including the device safe area. */
@media (max-width: 500px) {
  .shell {
    display: block;
    padding: var(--space-12) var(--space-12)
      calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
  }

  .main {
    padding-bottom: 0;
  }
}

/* Deck mode gives the widget column's width back to the horizontal column
   strip; the sidebar keeps its own width so the icon state below 1280px still
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

/* The sidebar root belongs to the Sidebar component, so reach through the
   component boundary for the deck height. */
.shell[data-shell="deck"] :deep(.sidebar) {
  height: calc(100dvh - var(--space-48));
}

@media (max-width: 1279px) {
  .shell[data-shell="deck"] {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr);
  }
}

/* On phones the shell stacks into a column for the deck strip; the bottom bar
   floats over the reserved padding, so the strip stops above it. */
@media (max-width: 500px) {
  .shell[data-shell="deck"] {
    display: flex;
    flex-direction: column;
    grid-template-columns: none;
    height: 100dvh;
    padding: var(--space-12) var(--space-12)
      calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
  }
}
</style>
