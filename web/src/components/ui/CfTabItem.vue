<template>
  <span class="cf-tab-item"><slot></slot></span>
</template>

<script setup lang="ts">
// Marker component for CfTab's default slot. CfTab reads `value` / `disabled`
// off the vnode props; this component is never rendered on its own.
defineProps<{
  value: string | number;
  disabled?: boolean;
}>();
</script>
