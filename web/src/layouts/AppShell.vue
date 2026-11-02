<script setup lang="ts">
import { computed, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import MobileSidebar from "../components/layout/MobileSidebar.vue";
import Navbar from "../components/layout/Navbar.vue";
import WidgetHost from "../components/widgets/WidgetHost.vue";
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
  <!-- The Firefish universal composition (ui/universal.vue): navbar, the
       sticky contents column, the widget column. Deck mode keeps the navbar
       and hands the widget column's width to the horizontal strip, so
       WidgetHost only mounts in universal mode. -->
  <div class="shell" :data-shell="props.mode">
    <div
      v-if="busy"
      class="api-progress"
      role="progressbar"
      aria-label="正在加载"
      aria-valuetext="正在加载"
    ></div>

    <Navbar
      :deck-preferred="props.deckPreferred"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />

    <main class="contents">
      <!-- Every route except the three deck surfaces renders through the deck
           page window instead, so the slot is the routed view. -->
      <slot />
    </main>

    <div v-if="props.mode === 'universal'" class="widgets-container">
      <WidgetHost />
    </div>

    <!-- Below 1090px the widget column is hidden and the same stack moves into
         this drawer behind its floating fab; the fab hides itself above 1090px,
         so widget access survives every width. -->
    <WidgetsDrawer />

    <!-- Below 500px the navbar is gone and the drawer + bottom bar take over;
         Navbar hides itself at that breakpoint, so no second copy is needed. -->
    <MobileSidebar
      :deck-preferred="props.deckPreferred"
      @toggle-theme="toggleTheme"
      @toggle-mode="toggleShellMode"
      @signout="handleSignout"
    />
  </div>
</template>

<style scoped>
/* Firefish's universal.vue (dkgtipfy) is three flush columns divided by
   borders, not three cards floating on a gap: navbar (250px, 80px below
   1280px) | contents | widgets (300px, hidden below 1090px). Layout scalars
   are defined here rather than in main.css because only the shells use them. */
.shell {
  --nav-width: 250px;
  --nav-icon-only-width: 80px;
  --widgets-hide-threshold: 1090px;
  --widgets-width: 300px;

  position: relative;
  display: flex;
  box-sizing: border-box;
  min-height: 100dvh;
  margin: 0 auto;
}

/* Firefish's contents column (MkStickyContainer + main): the reading column
   owns a divider against the widgets side, keeps the header bleed padding, and
   is capped so a wide monitor does not stretch the text. */
.contents {
  flex: 1 1 auto;
  min-width: 0;
  box-sizing: border-box;
  overflow-x: clip;
  border-inline: 1px solid var(--divider);
  /* Matches --page-header-bleed in PageHeader.vue: the sticky header bleeds
     back over this padding, so the two numbers must stay equal or the header
     leaks across the column divider. */
  padding: 0 var(--space-24) var(--space-24);
}

@media (min-width: 1280px) {
  .contents {
    max-width: 750px;
  }
}

/* Firefish's widgets-container: sticky, its own scroll, 300px plus margin
   padding, hidden below 1090px. */
.widgets-container {
  position: sticky;
  top: 0;
  max-height: 100dvh;
  overflow-y: auto;
  overflow-x: clip;
  box-sizing: content-box;
  inline-size: var(--widgets-width);
  min-inline-size: max-content;
  padding-inline: var(--margin);
}

@media (max-width: 1090px) {
  .widgets-container {
    display: none;
  }
}

/* Firefish's mobile threshold: from here down the navbar is gone (both
   Navbar.vue and the mobile rules hide it) and the bottom bar plus drawer take
   over; the reserved padding keeps the page above the fixed bar, including the
   device safe area. */
@media (max-width: 500px) {
  .contents {
    border-inline: 0;
    padding: 0 var(--space-12) calc(var(--margin) + 4rem + env(safe-area-inset-bottom));
  }
}

/* Deck mode gives the widget column's width back to the horizontal column
   strip; the navbar keeps its own width so the icon state below 1280px still
   applies. Kept last so its higher specificity wins over the breakpoints. */
.shell[data-shell="deck"] .contents {
  max-width: none;
  display: flex;
  flex-direction: column;
  border-inline-end: 0;
  height: 100dvh;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.shell[data-shell="deck"] {
  height: 100dvh;
  min-height: 0;
  overflow: hidden;
}

/* On phones the shell stacks into a column for the deck strip; the bottom bar
   floats over the reserved padding, so the strip stops above it. */
@media (max-width: 500px) {
  .shell[data-shell="deck"] {
    display: flex;
    flex-direction: column;
    height: 100dvh;
    padding: 0 0 calc(var(--margin) + 4rem + env(safe-area-inset-bottom));
  }
}

/* The waiting bar sits on the viewport edge, above the columns, and never
   takes part in layout. */
.api-progress {
  position: fixed;
  inset: 0 0 auto 0;
  z-index: 1000;
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
  animation: api-progress-slide 1.1s ease infinite;
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
