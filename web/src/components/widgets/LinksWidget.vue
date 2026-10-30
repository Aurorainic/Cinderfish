<script setup lang="ts">
import WidgetCard from "./WidgetCard.vue";

defineProps<{ widgetProps?: Record<string, unknown> }>();

// Static entry points only: the explore timeline and the two protocol
// discovery documents the Go server serves under /.well-known.
const links = [
  { label: "探索时间线", to: "/explore", external: false },
  { label: "NodeInfo", to: "/.well-known/nodeinfo", external: true },
  { label: "HostMeta", to: "/.well-known/host-meta", external: true },
];
</script>

<template>
  <WidgetCard title="关于" icon="globe">
    <ul class="links">
      <li v-for="link in links" :key="link.to">
        <a v-if="link.external" :href="link.to" target="_blank" rel="noopener">{{ link.label }}</a>
        <RouterLink v-else :to="link.to">{{ link.label }}</RouterLink>
      </li>
    </ul>
    <p class="license">Cinderfish · AGPL-3.0</p>
  </WidgetCard>
</template>

<style scoped>
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
