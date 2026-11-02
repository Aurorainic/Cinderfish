// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/ui/_common_/navbar.vue
// (adapted).
//
// The Firefish desktop navigation column: brand on top, the registry-driven
// nav list in the middle, the gradient post button at the bottom. Adaptations
// to Cinderfish:
//  - nav items come from web/src/nav.ts (the single registry) instead of
//    Firefish's navbarItemDef;
//  - the account menu, help menu and emoji-moderator entries are dropped —
//    Cinderfish has no account menu component yet; the account chip links to
//    the profile and the footer icon buttons carry theme/deck/signout;
//  - `iconOnly` follows the same rule as upstream (window width <= 1279px or
//    defaultStore.menuDisplay === "sideIcon") but as a media query listener
//    instead of MatchMedia onchange assignment;
//  - post opens through os.openPostForm, the shared global composer.
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import AppIcon from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";
import { defaultStore } from "../../store/firefish-store";
import { navItemsFor } from "../../nav";
import { openPostForm } from "../../os";
import { navPrefs } from "../../stores/nav";
import { session } from "../../stores/session";
import { theme } from "../../stores/theme";

defineProps<{ deckPreferred: boolean }>();

const emit = defineEmits<{
  "toggle-theme": [];
  "toggle-mode": [];
  signout: [];
}>();

const iconOnly = ref(false);
let stopMenuDisplay: (() => void) | null = null;

// Upstream: iconOnly = window.innerWidth <= 1279 || menuDisplay === "sideIcon".
const narrowQuery = window.matchMedia("(max-width: 1279px)");

function calcViewState(): void {
  iconOnly.value =
    narrowQuery.matches || defaultStore.state.menuDisplay === "sideIcon";
}

function onMenuDisplayChange(): void {
  calcViewState();
}

calcViewState();

onMounted(() => {
  narrowQuery.addEventListener("change", calcViewState);
  stopMenuDisplay = watch(
    () => defaultStore.reactiveState.menuDisplay.value,
    onMenuDisplayChange,
  );
});

onBeforeUnmount(() => {
  narrowQuery.removeEventListener("change", calcViewState);
  stopMenuDisplay?.();
});

const nav = computed(() => navItemsFor(navPrefs.value.desktop));
</script>

<template>
  <header class="navbar" :class="{ iconOnly }">
    <div class="body">
      <RouterLink to="/" class="brand" aria-label="Cinderfish 首页">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">{{ "Cinderfish" }}</span>
      </RouterLink>

      <nav class="middle" aria-label="主导航">
        <RouterLink
          v-for="item in nav"
          :key="item.key"
          :to="item.def.to"
          class="item"
          :title="item.def.label"
        >
          <span class="icon"><AppIcon :name="item.def.icon" /></span>
          <span class="text">{{ item.def.label }}</span>
          <span v-if="item.badge" class="indicator">
            {{ item.badge > 99 ? "99+" : item.badge }}
          </span>
        </RouterLink>
      </nav>

      <div class="bottom">
        <button
          class="item post"
          type="button"
          data-cy-open-post-form
          title="发帖"
          @click="openPostForm()"
        >
          <span class="icon"><AppIcon name="pencil" /></span>
          <span class="text">发帖</span>
        </button>

        <div class="bottom-row">
          <RouterLink
            v-if="session.me"
            :to="`/@${session.me.username}`"
            class="account"
            :title="session.me.username"
          >
            <UserAvatar :user="session.me" :size="34" />
            <span v-if="!iconOnly" class="account-name">{{ session.me.username }}</span>
          </RouterLink>
          <button
            class="icon-button mode-toggle"
            type="button"
            :aria-pressed="deckPreferred"
            :title="deckPreferred ? '切换到通用视图' : '切换到 Deck 视图'"
            @click="emit('toggle-mode')"
          >
            <AppIcon :name="deckPreferred ? 'display' : 'columns'" />
          </button>
          <button
            class="icon-button"
            type="button"
            :title="theme === 'dark' ? '切换浅色' : '切换深色'"
            @click="emit('toggle-theme')"
          >
            <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
          </button>
          <button
            v-if="session.me"
            class="icon-button"
            type="button"
            title="退出登录"
            @click="emit('signout')"
          >
            <AppIcon name="logout" />
          </button>
        </div>
      </div>
    </div>
  </header>
</template>

<style scoped>
/* Upstream .mvcprjjd: the column is 250px wide; below 1280px (or when
   menuDisplay is sideIcon) the body collapses to 80px and labels drop. The
   body is sticky and scrolls on its own. */
.navbar {
  flex: 0 0 250px;
  inline-size: 250px;
  box-sizing: border-box;
}

.body {
  position: sticky;
  inset-block-start: 0;
  inline-size: 250px;
  block-size: 100dvh;
  box-sizing: border-box;
  overflow-y: auto;
  overflow-x: clip;
  display: flex;
  flex-direction: column;
  background: var(--navBg);
  border-inline-end: 1px solid var(--divider);
  padding: var(--space-16) var(--space-12);
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  min-width: 0;
  padding: 6px 8px;
  border-radius: var(--radius-full);
  color: var(--fgHighlighted);
}

.brand:hover {
  color: var(--fgHighlighted);
  background: var(--buttonHoverBg);
}

.brand-mark {
  display: grid;
  place-items: center;
  inline-size: 32px;
  block-size: 32px;
  flex: none;
  border-radius: var(--radius-full);
  color: var(--fgOnAccent);
  background: var(--accent);
}

.brand-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 700;
}

.middle {
  display: flex;
  flex-direction: column;
  gap: 0;
  margin-block-start: var(--space-8);
}

/* Upstream .item: 2.85rem line height, 30px leading padding, 32px icon, and a
   pill backdrop drawn by ::before sized to calc(100% - 34px) so the rounded
   background never touches the column edges. No background at rest. */
.item {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--space-8);
  padding: 0 var(--space-12) 0 30px;
  line-height: 2.85rem;
  border-radius: var(--radius-full);
  color: var(--navFg);
  font-size: 0.9em;
  white-space: nowrap;
  transition: color 0.4s ease;
}

.item::before {
  content: "";
  position: absolute;
  inset: 0;
  width: calc(100% - 34px);
  margin: auto;
  border-radius: 999px;
  background: transparent;
  transition: background 0.4s ease;
}

.item:hover {
  color: var(--navHoverFg);
}

.item:hover::before,
.item.router-link-active::before {
  background: var(--accentedBg);
}

.item.router-link-active {
  color: var(--navActive);
}

.item .icon {
  position: relative;
  display: inline-flex;
  inline-size: 32px;
  block-size: 32px;
  flex: none;
}

.indicator {
  position: absolute;
  top: 50%;
  inset-inline-end: 8px;
  transform: translateY(-50%);
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--accent);
  color: var(--fgOnAccent);
  font-size: 0.65rem;
  font-weight: 800;
  line-height: 16px;
  text-align: center;
}

.bottom {
  margin-block-start: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  padding: 0;
}

/* Upstream .post: a 40px pill behind the row painted with the button
   gradient, accent-lightened on hover. */
.item.post {
  block-size: 40px;
  color: var(--fgOnAccent);
  font-weight: bold;
}

.item.post::before {
  background: linear-gradient(
    var(--gradient-to-inline-end),
    var(--buttonGradateA),
    var(--buttonGradateB)
  );
}

.item.post:hover,
.item.post:focus-visible {
  color: var(--fgOnAccent);
}

.item.post:hover::before,
.item.post:focus-visible::before {
  background: var(--accentLighten);
}

.bottom-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-8);
  padding: 0 2px;
}

.account {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  min-width: 0;
  flex: 1 1 100%;
  padding: 3px;
  border-radius: var(--radius-full);
}

.account-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--fg);
}

.icon-button {
  display: grid;
  place-items: center;
  inline-size: 34px;
  block-size: 34px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--fg);
  cursor: pointer;
}

.icon-button:hover {
  background: var(--buttonHoverBg);
  color: var(--accent);
}

.mode-toggle[aria-pressed="true"] {
  color: var(--accent);
  background: var(--accentedBg);
}

.brand:focus-visible,
.item:focus-visible,
.account:focus-visible,
.icon-button:focus-visible {
  outline: var(--focus);
  outline-offset: -2px;
}

/* Icon-only state: labels drop, everything centres in the 80px body. */
.navbar.iconOnly {
  flex: 0 0 80px;
  inline-size: 80px;
}

.navbar.iconOnly .body {
  inline-size: 80px;
  padding-inline: 0;
}

.navbar.iconOnly .brand {
  justify-content: center;
}

.navbar.iconOnly .brand-name,
.navbar.iconOnly .text,
.navbar.iconOnly .account-name {
  display: none;
}

.navbar.iconOnly .item {
  justify-content: center;
  padding: 0;
}

.navbar.iconOnly .item .icon {
  margin-inline-start: 0;
}

.navbar.iconOnly .indicator {
  inset-inline-end: 12px;
}

.navbar.iconOnly .bottom-row {
  flex-direction: column;
  align-items: center;
}

.navbar.iconOnly .account {
  flex: none;
}

/* Media-query fallback for the same state, so a window resized below 1280px
   collapses even before the listener fires. */
@media (max-width: 1279px) {
  .navbar {
    flex: 0 0 80px;
    inline-size: 80px;
  }

  .body {
    inline-size: 80px;
    padding-inline: 0;
  }

  .brand {
    justify-content: center;
  }

  .brand-name,
  .text,
  .account-name {
    display: none;
  }

  .item {
    justify-content: center;
    padding: 0;
  }

  .indicator {
    inset-inline-end: 12px;
  }

  .bottom-row {
    flex-direction: column;
    align-items: center;
  }

  .account {
    flex: none;
  }
}

/* At Firefish's mobile threshold the column leaves entirely; the bottom bar
   and drawer take over the same registry. */
@media (max-width: 500px) {
  .navbar {
    display: none;
  }
}
</style>
