<script setup lang="ts">
import { computed } from "vue";
import type { NoteVisibility } from "../../api/types";
import AppIcon from "../AppIcon.vue";

const props = defineProps<{ visibility: NoteVisibility; localOnly?: boolean }>();

const label = computed(
  () => ({ public: "公开", home: "首页", followers: "关注者", specified: "指定" })[props.visibility] ?? props.visibility,
);

const title = computed(() => (props.localOnly ? `${label.value} · 仅本地` : label.value));
</script>

<template>
  <span class="vis" :title="title" :aria-label="title" role="img">
    <svg
      v-if="visibility === 'public'"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.8"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3c2.5 2.5 3.8 5.5 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.5-3.8-9S9.5 5.5 12 3Z" />
    </svg>
    <svg
      v-else-if="visibility === 'home'"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.8"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="M3 10.5 12 3l9 7.5M5 9.5V21h14V9.5" />
    </svg>
    <svg
      v-else-if="visibility === 'followers'"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.8"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <rect x="4" y="10" width="16" height="11" rx="2" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3" />
    </svg>
    <svg
      v-else
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.8"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="3.5" />
      <path d="M20.5 12a8.5 8.5 0 0 1-8.5 8.5 8.5 8.5 0 0 1-6.9-3.4M3.5 12A8.5 8.5 0 0 1 12 3.5c2.8 0 5.3 1.4 6.9 3.4" />
    </svg>
    <AppIcon v-if="localOnly" name="user" class="local" />
  </span>
</template>

<style scoped>
.vis {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--muted);
  flex: none;
}

.vis svg {
  width: 0.92rem;
  height: 0.92rem;
  display: block;
}

.local {
  width: 0.85rem;
  height: 0.85rem;
  opacity: 0.7;
}
</style>
