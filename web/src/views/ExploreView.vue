<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import TimelineFeed from "../components/TimelineFeed.vue";

const route = useRoute();
const scope = ref<"local" | "global">("local");
const tag = computed(() => (typeof route.query.tag === "string" ? route.query.tag : ""));
</script>

<template>
  <header class="page-title">
    <h1>探索</h1>
  </header>

  <p v-if="tag" class="notice info">筛选话题：#{{ tag }}</p>

  <nav class="tabs">
    <button type="button" class="tab" :class="{ active: scope === 'local' }" @click="scope = 'local'">本地</button>
    <button type="button" class="tab" :class="{ active: scope === 'global' }" @click="scope = 'global'">全球</button>
  </nav>

  <TimelineFeed :source="scope" :tag="tag" />
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 6px;
  padding: 5px;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel);
  width: fit-content;
}

.tab {
  padding: 7px 18px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  cursor: pointer;
}

.tab.active {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
