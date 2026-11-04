<template>
  <div class="cf-shell-universal">
    <aside class="sidebar">
      <CfA to="/" class="logo" aria-label="Cinderfish">
        <span class="mark">C</span>
        <span class="wordmark">Cinderfish</span>
      </CfA>

      <nav class="nav">
        <CfA
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          active-class="active"
        >
          <i :class="item.icon"></i>
          <span class="label">{{ t(item.label) }}</span>
        </CfA>
      </nav>

      <div class="bottom">
        <CfAccountButton @click="openAccount" @login="goLogin" />
      </div>
    </aside>

    <main id="maincontent" class="contents">
      <RouterView />
    </main>

    <aside class="widgets">
      <slot name="widgets"></slot>
    </aside>

    <nav class="mobile-nav">
      <CfA to="/" class="nav-item" active-class="active">
        <i class="ph-house ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/my/notifications" class="nav-item" active-class="active">
        <i class="ph-bell ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/explore" class="nav-item" active-class="active">
        <i class="ph-compass ph-bold ph-lg"></i>
      </CfA>
      <button class="nav-item _button" @click="goLogin">
        <i class="ph-user ph-bold ph-lg"></i>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
// Structure ported from Firefish's ui/universal.vue: fixed left sidebar,
// scrollable main column and an optional widgets column on wide screens.
// The drawer menu, status bars and widget implementations are later ports;
// this shell only fixes the grid and the account chip.
import { useRouter } from "vue-router";
import CfA from "../components/ui/CfA.vue";
import { CfAccountButton } from "../account/components";
import { useLocale } from "../i18n";
import { me } from "../account";
import { userPage } from "../components/ui/icon";

interface NavItem {
  to: string;
  icon: string;
  label: string;
}

const router = useRouter();
const { t } = useLocale();

const navItems: NavItem[] = [
  { to: "/", icon: "ph-house ph-bold ph-lg", label: "timeline" },
  { to: "/my/notifications", icon: "ph-bell ph-bold ph-lg", label: "notifications" },
  { to: "/explore", icon: "ph-compass ph-bold ph-lg", label: "explore" },
  { to: "/my/drive", icon: "ph-cloud ph-bold ph-lg", label: "drive" },
  { to: "/settings", icon: "ph-gear ph-bold ph-lg", label: "settings" },
];

function openAccount(): void {
  if (me.value != null) void router.push(userPage(me.value));
}

function goLogin(): void {
  void router.push({ path: "/", query: { login: "1" } });
}
</script>

<style lang="scss" scoped>
.cf-shell-universal {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr) 300px;
  gap: var(--margin);
  max-width: 1300px;
  margin: 0 auto;
  padding: var(--margin);

  @media (max-width: 1100px) {
    grid-template-columns: 220px minmax(0, 1fr);
  }

  @media (max-width: 700px) {
    grid-template-columns: minmax(0, 1fr);
    padding-bottom: 72px;
  }
}

.sidebar {
  position: sticky;
  top: var(--margin);
  align-self: start;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: calc(100vh - var(--margin) * 2);
  padding: 16px 8px;
  box-sizing: border-box;

  @media (max-width: 700px) {
    display: none;
  }
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 10px;
  font-weight: 800;
  font-size: 1.1em;

  > .mark {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: 10px;
    color: var(--fgOnAccent);
    background: linear-gradient(135deg, var(--accent), var(--renote));
  }
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 999px;
  color: var(--navFg);
  text-decoration: none;
  transition:
    background 0.15s ease,
    color 0.15s ease;

  > i {
    font-size: 1.1em;
  }

  &:hover {
    background: var(--buttonHoverBg);
    color: var(--navHoverFg);
  }

  &.active {
    color: var(--navActive);
    background: var(--accentedBg);
  }
}

.bottom {
  margin-top: auto;
}

.contents {
  min-width: 0;
  min-height: 100vh;
}

.widgets {
  position: sticky;
  top: var(--margin);
  align-self: start;
  max-height: calc(100vh - var(--margin) * 2);
  overflow: auto;

  &:empty {
    display: none;
  }

  @media (max-width: 1100px) {
    display: none;
  }
}

.mobile-nav {
  display: none;

  @media (max-width: 700px) {
    position: fixed;
    z-index: 100;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    justify-content: space-around;
    padding: 6px 8px calc(6px + env(safe-area-inset-bottom));
    background: var(--header);
    -webkit-backdrop-filter: var(--blur, blur(15px));
    backdrop-filter: var(--blur, blur(15px));
    border-top: solid 0.5px var(--divider);

    > .nav-item {
      justify-content: center;
      width: 52px;
      height: 44px;
      padding: 0;

      > .label {
        display: none;
      }
    }
  }
}
</style>
