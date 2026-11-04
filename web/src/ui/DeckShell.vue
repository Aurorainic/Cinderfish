<template>
  <div class="cf-shell-deck">
    <aside class="rail">
      <CfA to="/" class="rail-item" active-class="active" aria-label="Timeline">
        <i class="ph-house ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/my/notifications" class="rail-item" active-class="active" aria-label="Notifications">
        <i class="ph-bell ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/explore" class="rail-item" active-class="active" aria-label="Explore">
        <i class="ph-compass ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/my/drive" class="rail-item" active-class="active" aria-label="Drive">
        <i class="ph-cloud ph-bold ph-lg"></i>
      </CfA>
      <CfA to="/settings" class="rail-item" active-class="active" aria-label="Settings">
        <i class="ph-gear ph-bold ph-lg"></i>
      </CfA>
      <div class="rail-bottom">
        <CfAccountButton @click="openAccount" @login="goLogin" />
      </div>
    </aside>

    <main class="column main">
      <RouterView />
    </main>

    <aside class="column side">
      <slot name="widgets"></slot>
    </aside>
  </div>
</template>

<script setup lang="ts">
// Structure ported from Firefish's ui/deck.vue: a narrow icon rail plus a
// horizontally scrolling set of columns. Deck column configuration lives in
// the store (`deck`, `widgets`); this skeleton renders the main column and an
// optional widgets column until the deck column port lands.
import { useRouter } from "vue-router";
import CfA from "../components/ui/CfA.vue";
import { CfAccountButton } from "../account/components";
import { me } from "../account";
import { userPage } from "../components/ui/icon";

const router = useRouter();

function openAccount(): void {
  if (me.value != null) void router.push(userPage(me.value));
}

function goLogin(): void {
  void router.push({ path: "/", query: { login: "1" } });
}
</script>

<style lang="scss" scoped>
.cf-shell-deck {
  display: flex;
  height: 100vh;
  overflow: hidden;
  gap: var(--margin);
  padding: var(--margin);
  box-sizing: border-box;
}

.rail {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
  width: 56px;
  padding: 8px 4px;
  border-radius: var(--radius);
  background: var(--panel);

  > .rail-item {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 44px;
    border-radius: 10px;
    color: var(--navFg);

    &:hover {
      background: var(--buttonHoverBg);
    }

    &.active {
      color: var(--navActive);
      background: var(--accentedBg);
    }
  }

  > .rail-bottom {
    margin-top: auto;

    :deep(.cf-account-button) {
      padding: 4px;

      :deep(.names),
      :deep(i) {
        display: none;
      }
    }
  }
}

.column {
  min-width: 0;
  height: 100%;
  overflow: auto;

  &.main {
    flex: 1 1 380px;
  }

  &.side {
    flex: 0 0 300px;

    &:empty {
      display: none;
    }
  }
}
</style>
