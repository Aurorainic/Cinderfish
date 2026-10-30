<script setup lang="ts">
import { computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import TimelineFeed from "../components/TimelineFeed.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import { usePageTitle } from "../composables/use-page-title";
import { isLoggedIn, session } from "../stores/session";

type Source = "home" | "local" | "hybrid" | "global";

// Misskey channel names: home reads as 首页, hybrid as 社交. Deck columns use
// the same labels (stores/deck.ts), so both shells name a channel alike.
const CHANNELS: readonly Source[] = ["home", "local", "hybrid", "global"];
const LABELS: Record<Source, string> = {
  home: "首页",
  local: "本地",
  hybrid: "社交",
  global: "全球",
};

const AUTH_CHANNELS: readonly string[] = ["home", "hybrid"];

const route = useRoute();
const router = useRouter();

usePageTitle("时间线");

function isChannel(value: string): value is Source {
  return (CHANNELS as readonly string[]).includes(value);
}

function isAvailable(value: string): value is Source {
  return isChannel(value) && (isLoggedIn.value || !AUTH_CHANNELS.includes(value));
}

const tabs = computed(() =>
  CHANNELS.filter((key) => isAvailable(key)).map((key) => ({ key, label: LABELS[key] })),
);

const tab = computed<string>({
  get() {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    if (isAvailable(raw)) return raw;
    return isLoggedIn.value ? "home" : "local";
  },
  set(value) {
    if (route.query.tab === value) return;
    void router.push({ query: { ...route.query, tab: value } });
  },
});

const source = computed<Source>(() => (isChannel(tab.value) ? tab.value : "local"));

watch(
  [() => route.query.tab, () => session.ready, isLoggedIn],
  () => {
    if (!session.ready) return;
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    if (!raw || raw === tab.value) return;
    void router.replace({ query: { ...route.query, tab: tab.value } });
  },
  { immediate: true },
);
</script>

<template>
  <PageHeader v-model:tab="tab" title="时间线" :tabs="tabs" />
  <TimelineFeed :source="source" />
</template>
