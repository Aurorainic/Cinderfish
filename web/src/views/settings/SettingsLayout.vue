<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import SectionMenu from "../../components/ui/SectionMenu.vue";
import PageHeader from "../../components/ui/PageHeader.vue";
import { usePageTitle } from "../../composables/use-page-title";
import { isLoggedIn, session } from "../../stores/session";
import { findSettingsTab, settingsMenuDef } from "./menu";

// Firefish collapses its settings menu at 600px; the same threshold is applied
// to this column, which is already narrower than the viewport inside the shell.
const NARROW_THRESHOLD = 600;

const route = useRoute();

const rootEl = ref<HTMLElement | null>(null);
const narrow = ref(false);

// Child routes are named `settings-<key>`, so the active tab comes from the
// route name and an unknown path has already been redirected by the router.
const activeTab = computed(() => {
  const name = String(route.name ?? "").replace(/^settings-/, "");
  return findSettingsTab(name);
});

const menuDef = computed(() => settingsMenuDef(isLoggedIn.value));

usePageTitle(() => activeTab.value.label);

let observer: ResizeObserver | null = null;

function measure(): void {
  narrow.value = (rootEl.value?.offsetWidth ?? 0) < NARROW_THRESHOLD;
}

onMounted(() => {
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
  <div ref="rootEl" class="settings" :class="{ narrow }">
    <PageHeader
      title="设置"
      :subtitle="activeTab.label"
      icon="settings"
      back
      back-to="/"
    />

    <div class="body">
      <SectionMenu :def="menuDef" :active="activeTab.key" :narrow="narrow" label="设置分类" />
      <section class="content">
        <p v-if="!session.ready" class="settings-hint">正在读取会话…</p>
        <RouterView v-else />
      </section>
    </div>
  </div>
</template>

<style scoped>
.settings {
  min-width: 0;
}

.body {
  display: grid;
  grid-template-columns: 196px minmax(0, 1fr);
  gap: var(--space-16);
  align-items: start;
  margin-top: var(--margin);
  padding-bottom: var(--space-16);
  /* MkFolder headers are sticky; they stop below the sticky PageHeader rather
     than sliding underneath it. */
  --stickyTop: 52px;
}

.content {
  min-width: 0;
  /* Firefish's MkSpacer gutter: every tab body and folder title lines up on
     the same 16px inner edge. */
  padding-inline: var(--margin);
}

.narrow .body {
  display: flex;
  flex-direction: column;
  /* The grid used align-items: start for the two columns; in the stacked flex
     form that would size each row to its max-content width instead of the
     column, and the scrolling menu strip would push the page wider. */
  align-items: stretch;
  gap: var(--space-12);
  margin-top: var(--space-12);
}

@media (max-width: 720px) {
  .body {
    grid-template-columns: 168px minmax(0, 1fr);
    gap: var(--space-16);
  }
}
</style>
