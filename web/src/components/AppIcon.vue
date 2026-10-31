<script setup lang="ts">
defineProps<{ name: keyof typeof paths }>();

// A small stroked icon set, in the spirit of the Firefish/Misskey UI.
const paths = {
  home: "M3 10.5 12 3l9 7.5M5 9.5V21h5v-6h4v6h5V9.5",
  bell: "M6 8a6 6 0 1 1 12 0c0 5 2 6 2 6H4s2-1 2-6Zm4 9a2 2 0 0 0 4 0",
  compass: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm3.5-12.5-2 5-5 2 2-5 5-2Z",
  settings:
    "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Zm8-3a8 8 0 0 0-.2-1.8l2-1.5-2-3.4-2.3 1a8 8 0 0 0-3-1.7L14 2h-4l-.5 2.6a8 8 0 0 0-3 1.7l-2.3-1-2 3.4 2 1.5A8 8 0 0 0 4 12c0 .6.1 1.2.2 1.8l-2 1.5 2 3.4 2.3-1a8 8 0 0 0 3 1.7L10 22h4l.5-2.6a8 8 0 0 0 3-1.7l2.3 1 2-3.4-2-1.5c.1-.6.2-1.2.2-1.8Z",
  shield: "M12 3 4.5 6v6c0 4.5 3.2 7.8 7.5 9 4.3-1.2 7.5-4.5 7.5-9V6L12 3Z",
  display: "M3 4h18v12H3zM8 20h8M12 16v4",
  flame: "M12 22c4 0 7-2.8 7-6.6 0-3.1-1.9-5.2-3.6-7.2C13.8 6.3 12.7 4.2 13 2c-2.6 1-4.3 3.3-4.6 6.1-.9-.5-1.6-1.3-2-2.3C4.9 7.3 5 9.9 5 12.2 5 18 8 22 12 22Z",
  user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-8 9a8 8 0 0 1 16 0",
  sun: "M12 17a5 5 0 1 0 0-10 5 5 0 0 0 0 10ZM12 1v2m0 18v2M4.2 4.2l1.4 1.4m12.8 12.8 1.4 1.4M1 12h2m18 0h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4",
  moon: "M21 13.2A8.5 8.5 0 1 1 10.8 3a6.8 6.8 0 0 0 10.2 10.2Z",
  feather: "M20 4c-6 0-11 3-13 9l-3 7 7-3c6-2 9-7 9-13ZM8 16l4-4",
  logout: "M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3",
  plus: "M12 5v14M5 12h14",
  close: "M6 6l12 12M18 6 6 18",
  reply: "M21 11.5a8.4 8.4 0 0 1-8.5 8.5 8.5 8.5 0 0 1-3.5-.8L3 21l1.8-6A8.5 8.5 0 1 1 21 11.5Z",
  renote: "M17 1l4 4-4 4M3 11V9a4 4 0 0 1 4-4h14M7 23l-4-4 4-4M21 13v2a4 4 0 0 1-4 4H3",
  heart: "M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1-1.1a5.5 5.5 0 0 0-7.8 7.8L12 21l8.8-8.6a5.5 5.5 0 0 0 0-7.8Z",
  image: "M3 4h18v16H3zM3 16l5-5 4 4 3-3 6 6",
  more: "M5 12h.01M12 12h.01M19 12h.01",
  search: "M21 21l-5-5m-3 2a7 7 0 1 1 0-14 7 7 0 0 1 0 14Z",
  arrowLeft: "M19 12H5m6-6-6 6 6 6",
  check: "M4 12l5 5L20 6",
  globe: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18ZM3 12h18M12 3c2.5 2.5 3.8 5.5 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.5-3.8-9S9.5 5.5 12 3Z",
  quote: "M9 7a3.5 3.5 0 1 0 0 7c.4 0 .8-.1 1.2-.2-.4 2-1.8 3.4-4.2 4.4l.8 1.8c3.9-1.4 6.2-4.2 6.2-8.5V10A3.5 3.5 0 0 0 9 7Zm9 0a3.5 3.5 0 1 0 0 7c.4 0 .8-.1 1.2-.2-.4 2-1.8 3.4-4.2 4.4l.8 1.8c3.9-1.4 6.2-4.2 6.2-8.5V10A3.5 3.5 0 0 0 18 7Z",
  lock: "M6 11h12v10H6zM8.5 11V7.5a3.5 3.5 0 0 1 7 0V11",
} as const;
</script>

<template>
  <svg
    class="icon"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="paths[name]" />
  </svg>
</template>

<style scoped>
.icon {
  width: 1.2em;
  height: 1.2em;
  display: block;
}
</style>
