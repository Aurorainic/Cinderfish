<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { Component } from "vue";
import { useRoute } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import SectionMenu from "../components/ui/SectionMenu.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import type { SectionMenuGroup } from "../components/ui/section-menu";
import { usePageTitle } from "../composables/use-page-title";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, isLoggedIn, session } from "../stores/session";
import AdminFiles from "./admin/AdminFiles.vue";
import AdminOverview from "./admin/AdminOverview.vue";
import AdminSettings from "./admin/AdminSettings.vue";
import AdminUsers from "./admin/AdminUsers.vue";
import { ADMIN_TABS, resolveAdminTab } from "./admin/tabs";
import type { AdminTabDef, AdminTabKey } from "./admin/tabs";

// Same threshold and the same shared SectionMenu as the settings page, so the
// two consoles read as one composition: icon menu on the left, one section on
// the right, chips above the content when the column does not fit.
const NARROW_THRESHOLD = 600;

// Taken from the tab declaration order instead of restating the group labels:
// adding a group in tabs.ts is enough for it to appear here.
const GROUPS: readonly AdminTabDef["group"][] = [...new Set(ADMIN_TABS.map((item) => item.group))];

const PAGES: Record<AdminTabKey, Component> = {
  overview: AdminOverview,
  users: AdminUsers,
  files: AdminFiles,
  settings: AdminSettings,
};

const route = useRoute();

const rootEl = ref<HTMLElement | null>(null);
const narrow = ref(false);

usePageTitle("管理");

// The tab lives in the query string rather than the route table, so
// /admin?tab=users is directly shareable and the back button works.
const tab = computed(() => resolveAdminTab(route.query.tab));

const menuDef = computed<SectionMenuGroup[]>(() =>
  GROUPS.map((title) => ({
    title,
    items: ADMIN_TABS.filter((item) => item.group === title).map((item) => ({
      key: item.key,
      label: item.label,
      icon: item.icon,
      to: { path: "/admin", query: { ...route.query, tab: item.key } },
    })),
  })).filter((group) => group.items.length > 0),
);

const page = computed(() => PAGES[tab.value]);

let observer: ResizeObserver | null = null;

function measure(): void {
  narrow.value = (rootEl.value?.offsetWidth ?? 0) < NARROW_THRESHOLD;
}

onMounted(() => {
  void loadInstance();
  measure();
  if (!rootEl.value) return;
  observer = new ResizeObserver(measure);
  observer.observe(rootEl.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});
</script>

<template>
  <div ref="rootEl" class="admin-page">
    <PageHeader title="管理" icon="shield" :subtitle="instance.meta?.name ?? '实例控制台'" />

    <div class="admin-body">
      <section v-if="!session.ready" class="card">
        <p class="muted small">正在确认登录状态…</p>
      </section>

      <section v-else-if="!isAdmin" class="card gate">
        <span class="gate-icon"><AppIcon name="shield" /></span>
        <h2>需要管理员权限</h2>
        <!-- The gate explains the state; it is not the access control. Every
             admin endpoint answers FORBIDDEN on the server for this account. -->
        <p class="muted gate-text">
          <template v-if="isLoggedIn">
            当前账号{{ session.me ? ` @${session.me.username}` : "" }}不是实例管理员，服务端也会拒绝管理接口的请求。
          </template>
          <template v-else>这个页面只对实例管理员开放。请使用管理员账号登录后再访问。</template>
        </p>
        <div class="gate-actions">
          <RouterLink v-if="!isLoggedIn" to="/login" class="button">登录</RouterLink>
          <RouterLink v-else to="/" class="button ghost">回到时间线</RouterLink>
        </div>
      </section>

      <div v-else class="admin-grid" :class="{ narrow }">
        <SectionMenu :def="menuDef" :active="tab" :narrow="narrow" label="管理分区" />
        <div class="admin-content">
          <component :is="page" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-page {
  min-width: 0;
}

.admin-body {
  padding-top: var(--space-20);
}

.admin-grid {
  display: grid;
  grid-template-columns: 164px minmax(0, 1fr);
  gap: var(--space-20);
  align-items: start;
}

.admin-grid.narrow {
  display: flex;
  flex-direction: column;
  /* The grid above aligns its two columns to the start; in the stacked flex
     form that would size each row to its max-content width and let the
     scrolling menu strip widen the page. Same fix as SettingsLayout. */
  align-items: stretch;
  gap: var(--space-14);
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
  max-width: 44ch;
}

.gate-actions {
  display: flex;
  gap: var(--space-8);
}

@media (max-width: 720px) {
  .admin-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-14);
  }
}
</style>
