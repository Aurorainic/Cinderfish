<script setup lang="ts">
import { computed, onMounted } from "vue";
import type { Component } from "vue";
import { useRoute } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import { usePageTitle } from "../composables/use-page-title";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, session } from "../stores/session";
import AdminFiles from "./admin/AdminFiles.vue";
import AdminOverview from "./admin/AdminOverview.vue";
import AdminSettings from "./admin/AdminSettings.vue";
import AdminUsers from "./admin/AdminUsers.vue";
import { ADMIN_TABS, resolveAdminTab } from "./admin/tabs";
import type { AdminTabKey } from "./admin/tabs";

const TABS = ADMIN_TABS;

const GROUPS = [
  { label: "实例", items: TABS.filter((item) => item.group === "实例") },
  { label: "配置", items: TABS.filter((item) => item.group === "配置") },
];

const PAGES: Record<AdminTabKey, Component> = {
  overview: AdminOverview,
  users: AdminUsers,
  files: AdminFiles,
  settings: AdminSettings,
};

const route = useRoute();

usePageTitle("管理");

// The tab lives in the query string rather than the route table, so
// /admin?tab=users is directly shareable and the back button works.
const tab = computed(() => resolveAdminTab(route.query.tab));

const page = computed(() => PAGES[tab.value]);

onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <PageHeader
    title="管理"
    icon="shield"
    :subtitle="instance.meta?.name ?? '实例控制台'"
  />

  <div class="admin-body">
    <section v-if="!session.ready" class="card">
      <p class="muted small">正在确认登录状态…</p>
    </section>

    <section v-else-if="!isAdmin" class="card gate">
      <span class="gate-icon"><AppIcon name="shield" /></span>
      <h2>需要管理员权限</h2>
      <p class="muted gate-text">
        这个页面只对实例管理员开放。请使用管理员账号登录后再访问。
      </p>
      <RouterLink to="/login" class="button">登录</RouterLink>
    </section>

    <div v-else class="admin-grid">
      <nav class="admin-nav" aria-label="管理分区">
        <div v-for="group in GROUPS" :key="group.label" class="nav-group">
          <p class="nav-group-title">{{ group.label }}</p>
          <RouterLink
            v-for="item in group.items"
            :key="item.key"
            :to="{ path: '/admin', query: { ...route.query, tab: item.key } }"
            class="admin-nav-link"
            :class="{ active: item.key === tab }"
            :aria-current="item.key === tab ? 'page' : undefined"
          >
            <AppIcon :name="item.icon" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </div>
      </nav>

      <div class="admin-content">
        <component :is="page" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-body {
  padding-top: var(--space-20);
}

.admin-grid {
  display: grid;
  grid-template-columns: 164px minmax(0, 1fr);
  gap: var(--space-20);
  align-items: start;
}

.admin-nav {
  position: sticky;
  top: calc(52px + var(--space-16));
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
}

.nav-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.nav-group-title {
  margin-bottom: var(--space-6);
  padding: 0 var(--space-12);
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.admin-nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-8) var(--space-12);
  border-radius: var(--radius-md);
  color: var(--nav-fg);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  transition: background var(--dur-base) ease, color var(--dur-base) ease;
}

.admin-nav-link .icon {
  flex: none;
  width: 18px;
  height: 18px;
}

.admin-nav-link:hover {
  background: var(--surface-hover);
  color: var(--nav-fg-hover);
}

.admin-nav-link.active {
  background: var(--surface-selected);
  color: var(--accent);
}

.admin-nav-link:focus-visible {
  outline: var(--focus-outline);
  outline-offset: -2px;
}

.admin-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  min-width: 0;
}

.gate {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-48) var(--space-24);
  text-align: center;
}

.gate-icon {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--accent);
}

.gate-icon .icon {
  width: 24px;
  height: 24px;
}

.gate h2 {
  font-size: var(--fs-lg);
}

.gate-text {
  max-width: 40ch;
  margin-bottom: var(--space-4);
}

@media (max-width: 680px) {
  .admin-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-12);
  }

  .admin-nav {
    position: static;
    flex-direction: row;
    align-items: center;
    gap: var(--space-4);
    padding-bottom: var(--space-4);
    overflow-x: auto;
    scrollbar-width: none;
  }

  .admin-nav::-webkit-scrollbar {
    display: none;
  }

  .nav-group {
    flex-direction: row;
    align-items: center;
    gap: var(--space-4);
  }

  .nav-group-title {
    display: none;
  }

  .admin-nav-link {
    padding: var(--space-8) var(--space-12);
    white-space: nowrap;
  }
}
</style>
