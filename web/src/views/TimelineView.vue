<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import TimelineFeed from "../components/TimelineFeed.vue";
import { isLoggedIn, session } from "../stores/session";

type Source = "home" | "local" | "hybrid" | "global";

const source = ref<Source>("local");
const chosen = ref(false);

// Misskey channel names: home reads as 首页, hybrid as 社交. Deck columns use
// the same labels (stores/deck.ts), so both shells name a channel alike.
const tabs: { id: Source; label: string; auth?: boolean }[] = [
  { id: "home", label: "首页", auth: true },
  { id: "local", label: "本地" },
  { id: "hybrid", label: "社交", auth: true },
  { id: "global", label: "全球" },
];

function pick(id: Source) {
  source.value = id;
  chosen.value = true;
}

// Logged-in users start on the home timeline, guests on local. The session
// resolves asynchronously, so wait for it unless the user already picked.
watch(
  () => session.ready,
  (ready) => {
    if (ready && !chosen.value) source.value = isLoggedIn.value ? "home" : "local";
  },
  { immediate: true },
);

onMounted(() => {
  if (session.ready && !chosen.value) source.value = isLoggedIn.value ? "home" : "local";
});
</script>

<template>
  <header class="page-title">
    <h1>时间线</h1>
  </header>

  <nav class="tabs">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      type="button"
      class="tab"
      :class="{ active: source === tab.id }"
      @click="pick(tab.id)"
    >
      {{ tab.label }}
      <span v-if="tab.auth && !isLoggedIn" class="lock" title="需要登录">·</span>
    </button>
  </nav>

  <TimelineFeed :source="source" />
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 6px;
  padding: 5px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  width: fit-content;
}

.tab {
  padding: 7px 18px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.tab:hover {
  color: var(--text);
}

.tab.active {
  background: var(--accent);
  color: var(--on-accent);
}

.lock {
  opacity: 0.7;
}
</style>
