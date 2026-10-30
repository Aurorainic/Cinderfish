<script setup lang="ts">
import { computed } from "vue";
import type { UserLite } from "../api/types";

const props = withDefaults(
  defineProps<{
    user: Pick<UserLite, "username" | "name" | "avatarUrl">;
    size?: number | "inherit";
    radius?: "50%" | "8px";
  }>(),
  { size: 40, radius: "50%" },
);

const initial = computed(() => (props.user.name || props.user.username || "?").slice(0, 1).toUpperCase());

const length = computed(() => (props.size === "inherit" ? "var(--avatar-size, 40px)" : `${props.size}px`));

const style = computed(() => ({
  width: length.value,
  height: length.value,
  borderRadius: props.radius,
}));
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
  /* Drawn inside the box so a radius change never shifts the layout around it. */
  outline: 1px solid var(--border);
  outline-offset: -1px;
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
