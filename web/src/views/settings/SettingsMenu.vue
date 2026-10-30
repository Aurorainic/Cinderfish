<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import { isLoggedIn } from "../../stores/session";
import { SETTINGS_GROUPS, SETTINGS_TABS } from "./menu";

// Wide: a sticky vertical menu under the page header. Narrow: the same items
// collapse into one horizontally scrolling row above the content, which is how
// Iceshrimp and Firefish hand the menu over to small screens.
defineProps<{ active: string; narrow: boolean }>();

const groups = computed(() =>
  SETTINGS_GROUPS.map((title) => ({
    title,
    tabs: SETTINGS_TABS.filter((tab) => tab.group === title),
  })),
);
</script>

<template>
  <nav class="settings-menu" :class="{ narrow }" aria-label="设置分类">
    <div v-for="group in groups" :key="group.title" class="group">
      <p v-if="!narrow" class="group-title">{{ group.title }}</p>
      <ul class="items">
        <li v-for="tab in group.tabs" :key="tab.key">
          <RouterLink
            :to="`/settings/${tab.key}`"
            class="item"
            :class="{ active: tab.key === active }"
            :aria-current="tab.key === active ? 'page' : undefined"
          >
            <AppIcon class="item-icon" :name="tab.icon" />
            <span class="item-label">{{ tab.label }}</span>
            <span v-if="tab.requiresAuth && !isLoggedIn" class="item-note">需登录</span>
          </RouterLink>
        </li>
      </ul>
    </div>
  </nav>
</template>

<style scoped>
.settings-menu {
  position: sticky;
  top: calc(52px + var(--space-16));
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  min-width: 0;
}

.group-title {
  margin: 0 0 var(--space-6);
  padding-left: var(--space-10);
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
  letter-spacing: 0.06em;
}

.items {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-8) var(--space-10);
  border-radius: var(--radius-md);
  color: var(--text-weak);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  transition: background var(--dur-base) ease, color var(--dur-base) ease;
}

.item:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.item:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.item.active {
  background: var(--surface-selected);
  color: var(--accent);
}

.item .item-icon {
  flex: none;
  width: 18px;
  height: 18px;
}

.item-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-note {
  margin-left: auto;
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-medium);
}

/* Narrow: one scrolling row. The groups keep their markup but dissolve into
   the row, so the items read as a single strip of chips. */
.settings-menu.narrow {
  position: static;
  flex-direction: row;
  align-items: center;
  gap: 0;
  padding-bottom: var(--space-2);
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
}

.settings-menu.narrow::-webkit-scrollbar {
  display: none;
}

.narrow .group {
  display: contents;
}

.narrow .items {
  flex-direction: row;
  gap: var(--space-6);
}

.narrow .item {
  flex: none;
  gap: var(--space-6);
  padding: var(--space-6) var(--space-12);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  background: var(--panel);
  white-space: nowrap;
}

.narrow .item.active {
  border-color: var(--accent-border);
  background: var(--accent-soft);
}

.narrow .item .item-icon {
  width: 16px;
  height: 16px;
}

.narrow .item-note {
  margin-left: var(--space-2);
}
</style>
