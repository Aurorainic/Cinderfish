<script setup lang="ts">
import { ref } from "vue";
import AppIcon from "../components/AppIcon.vue";

const theme = ref(document.documentElement.dataset.theme ?? "dark");

function toggleTheme() {
  theme.value = theme.value === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = theme.value;
  localStorage.setItem("cinderfish.theme", theme.value);
}

const nav = [
  { to: "/", icon: "home", label: "时间线" },
  { to: "/notifications", icon: "bell", label: "通知" },
  { to: "/explore", icon: "compass", label: "探索" },
  { to: "/settings", icon: "settings", label: "设置" },
  { to: "/admin", icon: "shield", label: "管理" },
] as const;
</script>

<template>
  <div class="shell">
    <aside class="rail">
      <RouterLink to="/" class="brand" aria-label="Cinderfish 首页">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">Cinderfish</span>
      </RouterLink>

      <nav class="nav">
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="nav-link">
          <AppIcon :name="item.icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="rail-foot">
        <button class="icon-button" type="button" :title="theme === 'dark' ? '切换浅色' : '切换深色'" @click="toggleTheme">
          <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" />
        </button>
      </div>
    </aside>

    <main class="main">
      <slot />
    </main>

    <aside class="side">
      <section class="card instance-card">
        <div class="instance-head">
          <span class="instance-logo"><AppIcon name="flame" /></span>
          <div>
            <h2>Cinderfish</h2>
            <p class="muted">个人联邦实例</p>
          </div>
        </div>
        <p class="muted small">
          一个个人或极小规模的联邦社交实例。后端遵循 Misskey API，通信层使用 ActivityPub。
        </p>
        <div class="instance-actions">
          <RouterLink to="/login" class="button ghost small">登录</RouterLink>
          <RouterLink to="/signup" class="button small">注册</RouterLink>
        </div>
      </section>

      <section class="card">
        <h3 class="card-title">关于</h3>
        <ul class="link-list">
          <li><RouterLink to="/explore">探索时间线</RouterLink></li>
          <li><a href="/.well-known/nodeinfo" target="_blank" rel="noopener">NodeInfo</a></li>
          <li><a href="/.well-known/host-meta" target="_blank" rel="noopener">HostMeta</a></li>
        </ul>
      </section>

      <p class="side-foot muted small">Cinderfish · AGPL-3.0</p>
    </aside>
  </div>
</template>
