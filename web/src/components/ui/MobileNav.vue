<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../AppIcon.vue";
import { openPostForm } from "../../os";
import { session } from "../../stores/session";
import { unreadNotifications } from "../../stores/streaming";

// The narrow-viewport navigation. It replaces the rail below 720px and is the
// only place the post entry lives on small screens; `openPostForm` is the
// frozen command-bus signature, so F9 swaps the panel without touching this.
const meRoute = computed(() => (session.me ? `/@${session.me.username}` : "/login"));
</script>

<template>
  <nav class="mobile-nav" aria-label="主导航">
    <RouterLink to="/" class="nav-item">
      <AppIcon name="home" />
      <span class="nav-label">时间线</span>
    </RouterLink>
    <RouterLink to="/notifications" class="nav-item">
      <span class="nav-icon">
        <AppIcon name="bell" />
        <span v-if="unreadNotifications" class="nav-badge">
          {{ unreadNotifications > 99 ? "99+" : unreadNotifications }}
        </span>
      </span>
      <span class="nav-label">通知</span>
    </RouterLink>
    <RouterLink to="/explore" class="nav-item">
      <AppIcon name="compass" />
      <span class="nav-label">探索</span>
    </RouterLink>
    <button class="nav-item nav-compose" type="button" @click="openPostForm()">
      <span class="compose-mark"><AppIcon name="plus" /></span>
      <span class="nav-label">发帖</span>
    </button>
    <RouterLink :to="meRoute" class="nav-item">
      <AppIcon name="user" />
      <span class="nav-label">我的</span>
    </RouterLink>
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
  padding: 6px 4px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font-size: 0.68rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
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
