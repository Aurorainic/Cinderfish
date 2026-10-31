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
  <span v-else class="avatar fallback" :style="style">
    <span class="letter">{{ initial }}</span>
  </span>
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

/* The box's own layout must not depend on the parent. A parent that styles
   `.avatar` (the note header's link wrapper does) hands its `display` down to
   this element, because Vue gives a child component's root the parent's
   scope id. Naming the box as well as the state keeps this rule specific
   enough to win that, so the box stays a grid. */
.avatar.fallback {
  display: grid;
}

/* The letter fills the box and centres itself inside it, so it stays centred
   whatever the box's own `display` turns out to be: a leaked `display` on
   the box changes how the box lays the letter out, but not where the letter
   sits inside it. */
.letter {
  width: 100%;
  height: 100%;
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
