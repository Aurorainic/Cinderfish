<script setup lang="ts">
import AppIcon from "../AppIcon.vue";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ widgetProps?: Record<string, unknown> }>();

// Static entry points only: the two protocol discovery documents the Go server
// serves under /.well-known. The explore timeline is the card's header action,
// since that is the one entry a reader is likely to follow.
const links = [
  { label: "NodeInfo", to: "/.well-known/nodeinfo" },
  { label: "HostMeta", to: "/.well-known/host-meta" },
];
</script>

<template>
  <WidgetCard title="关于" icon="globe" hint="探索与协议入口">
    <template #actions>
      <RouterLink class="explore" to="/explore" title="打开探索时间线" aria-label="打开探索时间线">
        <AppIcon name="compass" />
      </RouterLink>
    </template>

    <ul class="links">
      <li v-for="link in links" :key="link.to">
        <a :href="link.to" target="_blank" rel="noopener">{{ link.label }}</a>
      </li>
    </ul>
    <p class="license">Cinderfish · AGPL-3.0</p>
  </WidgetCard>
</template>

<style scoped>
.explore {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--border);
  border-radius: 50%;
  color: var(--accent);
  transition: border-color 0.15s ease;
}

.explore:hover {
  border-color: var(--accent);
}

.explore:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

.explore .icon {
  width: 13px;
  height: 13px;
}

.links {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
  color: var(--muted);
  font-size: 0.88rem;
}

.links a:hover {
  color: var(--accent);
}

.license {
  margin: 14px 0 0;
  padding-top: 10px;
  border-top: 1px solid var(--divider);
  color: var(--muted);
  font-size: 0.76rem;
  text-align: center;
}
</style>
