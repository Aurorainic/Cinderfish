<script setup lang="ts">
import { computed } from "vue";
import type { UserLite } from "../api/types";

const props = defineProps<{ user: Pick<UserLite, "username" | "name" | "avatarUrl">; size?: number }>();

const initial = computed(() => (props.user.name || props.user.username || "?").slice(0, 1).toUpperCase());
const style = computed(() => ({ width: `${props.size ?? 40}px`, height: `${props.size ?? 40}px` }));
</script>

<template>
  <img v-if="user.avatarUrl" class="avatar" :src="user.avatarUrl" :alt="user.username" :style="style" />
  <span v-else class="avatar fallback" :style="style">{{ initial }}</span>
</template>

<style scoped>
.avatar {
  flex: none;
  border-radius: 50%;
  object-fit: cover;
  background: var(--panel-2);
  border: 1px solid var(--border);
}

.fallback {
  display: grid;
  place-items: center;
  /* Parents often set line-height: 0 under an avatar to kill inline gaps; the
     fallback letter must keep its own line box or it drifts to the top edge. */
  line-height: 1;
  font-weight: 700;
  font-size: 0.85em;
  color: var(--text-strong);
}
</style>
