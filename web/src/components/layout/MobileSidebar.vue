<script setup lang="ts">
import { ref } from "vue";
import MobileNav from "../ui/MobileNav.vue";
import NavDrawer from "../ui/NavDrawer.vue";
import { session } from "../../stores/session";
import type { ShellMode } from "../../stores/shell";
import { theme } from "../../stores/theme";

// The narrow-viewport navigation, the counterpart of
// fork/ui/_common_/navbar-for-mobile.vue plus universal.vue's bottom button
// bar: both render from the same nav registry, so an item appears (or carries
// its badge) everywhere at once. The drawer and bottom bar already satisfied
// the contract — Teleport/transition, focus trap, Escape/overlay close and
// close-on-route-change — so this component composes the existing NavDrawer
// and MobileNav instead of rewriting them; it owns the drawer open state that
// used to sit in AppShell and forwards the shell actions up.
defineProps<{ mode: ShellMode }>();

const emit = defineEmits<{
  "toggle-theme": [];
  "toggle-mode": [];
  signout: [];
}>();

const drawerOpen = ref(false);
</script>

<template>
  <MobileNav @menu="drawerOpen = true" />
  <NavDrawer
    v-model:open="drawerOpen"
    :me="session.me"
    :mode="mode"
    :theme="theme"
    @toggle-theme="emit('toggle-theme')"
    @toggle-mode="emit('toggle-mode')"
    @signout="emit('signout')"
  />
</template>
