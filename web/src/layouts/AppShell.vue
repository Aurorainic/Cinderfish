<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import UserAvatar from "../components/UserAvatar.vue";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, refreshSession, session, signout } from "../stores/session";

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

onMounted(() => {
  refreshSession();
  loadInstance();
});
</script>

<template>
  <div class="shell">
    <aside class="rail">
      <RouterLink to="/" class="brand" aria-label="Cinderfish 首页">
        <span class="brand-mark"><AppIcon name="flame" /></span>
        <span class="brand-name">{{ instance.meta?.name ?? "Cinderfish" }}</span>
      </RouterLink>

      <nav class="nav">
        <RouterLink v-for="item in nav" :key="item.to" :to="item.to" class="nav-link">
          <AppIcon :name="item.icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="rail-foot">
        <RouterLink v-if="session.me" :to="`/@${session.me.username}`" class="account-chip" :title="session.me.username">
          <UserAvatar :user="session.me" :size="34" />
          <span class="account-name">{{ session.me.username }}</span>
        </RouterLink>
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

    <aside class="side">
      <section class="card instance-card">
        <div class="instance-head">
          <span class="instance-logo"><AppIcon name="flame" /></span>
          <div>
            <h2>{{ instance.meta?.name ?? "Cinderfish" }}</h2>
            <p class="muted small">个人联邦实例</p>
          </div>
        </div>
        <p class="muted small">
          {{ instance.meta?.description || "一个个人或极小规模的联邦社交实例。后端遵循 Misskey API，通信层使用 ActivityPub。" }}
        </p>
        <div v-if="!session.me" class="instance-actions">
          <RouterLink to="/login" class="button ghost small">登录</RouterLink>
          <RouterLink to="/signup" class="button small">注册</RouterLink>
        </div>
        <div v-else class="instance-actions">
          <RouterLink :to="`/@${session.me.username}`" class="button ghost small">我的主页</RouterLink>
          <RouterLink to="/settings" class="button small">设置</RouterLink>
        </div>
      </section>

      <section class="card">
        <h3 class="card-title">关于</h3>
        <ul class="link-list">
          <li><RouterLink to="/explore">探索时间线</RouterLink></li>
          <li><a :href="`/.well-known/nodeinfo`" target="_blank" rel="noopener">NodeInfo</a></li>
          <li><a :href="`/.well-known/host-meta`" target="_blank" rel="noopener">HostMeta</a></li>
        </ul>
      </section>

      <p class="side-foot muted small">Cinderfish · AGPL-3.0</p>
    </aside>
  </div>
</template>

<style scoped>
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

@media (max-width: 1080px) {
  .account-name {
    display: none;
  }
  .account-chip {
    padding: 3px;
    border-radius: 50%;
  }
}
</style>
