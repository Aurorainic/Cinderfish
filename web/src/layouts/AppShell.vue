<script setup lang="ts">
import { computed, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import MobileSidebar from "../components/layout/MobileSidebar.vue";
import Sidebar from "../components/layout/Sidebar.vue";
import WidgetPanel from "../components/layout/WidgetPanel.vue";
import WidgetsDrawer from "../components/widgets/WidgetsDrawer.vue";
import { pendingApiRequestsCount } from "../api/client";
import { loadInstance } from "../stores/instance";
import { refreshSession, session, signout } from "../stores/session";
import { type ShellMode, setShellMode } from "../stores/shell";
import { streaming, setUnreadNotifications } from "../stores/streaming";
import { toggleTheme } from "../stores/theme";

// `mode` is the layout this render actually uses; `deckPreferred` is the stored
// preference. They differ whenever the preference is "deck" and the current
// route is not one the deck can show, which is why the switch below flips the
// preference while the layout stays universal.
const props = withDefaults(defineProps<{ mode?: ShellMode; deckPreferred?: boolean }>(), {
  mode: "universal",
  deckPreferred: false,
});

const router = useRouter();

// The global waiting indicator (runtime contract §3.3). Firefish's own wait
// overlay is commented out in its App and only the request counter is kept; a
// thin bar is enough to say "the application is doing something" without
// covering the page a second time.
const busy = computed(() => pendingApiRequestsCount.value > 0);

function handleSignout() {
  signout();
  router.push("/");
}

// The layout switch is a preference, not a route: deck mode keeps the same URL
// and only swaps what the main area renders.
function toggleShellMode() {
  setShellMode(props.deckPreferred ? "universal" : "deck");
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
    <div
      v-if="busy"
      class="api-progress"
      role="progressbar"
      aria-label="正在加载"
      aria-valuetext="正在加载"
    ></div>

    <Sidebar
      :deck-preferred="props.deckPreferred"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />

    <main class="main">
      <!-- Every route except the three deck surfaces renders through the deck
           page window instead, so the slot is the routed view. -->
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
      :deck-preferred="props.deckPreferred"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />
  </div>
</template>

<style scoped>
/* Firefish's shell is three flush columns divided by borders, not three cards
   floating on a gap: sidebar (250px, 80px below 1280px) | main | widgets
   (300px, hidden below 1090px by the panel itself). The tracks are centred and
   capped so a wide monitor does not stretch the reading column. */
.shell {
  position: relative;
  display: grid;
  grid-template-columns: var(--rail-width) minmax(0, 1fr) var(--side-width);
  gap: 0;
  max-width: var(--shell-max-width);
  margin: 0 auto;
  min-height: 100vh;
  border-inline: 1px solid var(--divider);
}

.main {
  min-width: 0;
  /* Matches --page-header-bleed in PageHeader.vue: the sticky header bleeds
     back over this padding, so the two numbers must stay equal or the header
     leaks across the column divider. */
  padding: 0 var(--space-24) var(--space-64);
}

@media (max-width: 1279px) {
  .shell {
    grid-template-columns: var(--rail-width-collapsed) minmax(0, 1fr) var(--side-width);
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
    border-inline: 0;
  }

  .main {
    padding: 0 var(--space-12) calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
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
    padding: 0 0 calc(var(--bottom-bar-height) + env(safe-area-inset-bottom));
  }
}

/* The waiting bar sits on the viewport edge, above the columns, and never
   takes part in layout. */
.api-progress {
  position: fixed;
  inset: 0 0 auto 0;
  z-index: var(--z-overlay);
  height: 2px;
  overflow: hidden;
  background: transparent;
}

.api-progress::after {
  content: "";
  position: absolute;
  inset: 0;
  width: 40%;
  background: var(--accent);
  animation: api-progress-slide 1.1s var(--ease-standard) infinite;
}

@keyframes api-progress-slide {
  from {
    transform: translateX(-100%);
  }
  to {
    transform: translateX(350%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .api-progress::after {
    width: 100%;
    animation: none;
    opacity: 0.6;
  }
}
</style>
