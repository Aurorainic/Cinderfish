<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { Component } from "vue";
import { useRoute } from "vue-router";
import AppIcon from "../components/AppIcon.vue";
import MkButton from "../components/ui/MkButton.vue";
import MkLoading from "../components/ui/MkLoading.vue";
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
      <section v-if="!session.ready" class="_panel">
        <MkLoading />
      </section>

      <section v-else-if="!isAdmin" class="_panel gate">
        <span class="gate-glyph"><AppIcon name="shield" /></span>
        <h2>需要管理员权限</h2>
        <!-- The gate explains the state; it is not the access control. Every
             admin endpoint answers FORBIDDEN on the server for this account. -->
        <p class="gate-text">
          <template v-if="isLoggedIn">
            当前账号{{ session.me ? ` @${session.me.username}` : "" }}不是实例管理员，服务端也会拒绝管理接口的请求。
          </template>
          <template v-else>这个页面只对实例管理员开放。请使用管理员账号登录后再访问。</template>
        </p>
        <div class="gate-actions">
          <MkButton v-if="!isLoggedIn" link to="/login" primary>登录</MkButton>
          <MkButton v-else link to="/">回到时间线</MkButton>
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
  padding-block-start: var(--margin);
}

.admin-grid {
  display: grid;
  grid-template-columns: 164px minmax(0, 1fr);
  gap: var(--margin);
  align-items: start;
}

.admin-grid.narrow {
  display: flex;
  flex-direction: column;
  /* The grid above aligns its two columns to the start; in the stacked flex
     form that would size each row to its max-content width and let the
     scrolling menu strip widen the page. Same fix as SettingsLayout. */
  align-items: stretch;
  gap: 14px;
}

.admin-content {
  display: flex;
  flex-direction: column;
  gap: var(--margin);
  min-width: 0;
}

/* Firefish's full-page state (MkError / _fullinfo): a centred glyph, a short
   explanation and the action that gets the reader out of the dead end. */
.gate {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding-block: 48px;
  padding-inline: 24px;
  text-align: center;
}

.gate-glyph {
  display: grid;
  place-items: center;
  inline-size: 52px;
  block-size: 52px;
  border-radius: 50%;
  background: var(--accentedBg);
  color: var(--accent);
}

.gate-glyph .icon {
  inline-size: 24px;
  block-size: 24px;
}

.gate h2 {
  font-size: 1.15em;
}

.gate-text {
  max-inline-size: 44ch;
  color: var(--fgTransparentWeak);
}

.gate-actions {
  display: flex;
  gap: 8px;
}

@media (max-width: 720px) {
  .admin-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 14px;
  }
}
</style>
