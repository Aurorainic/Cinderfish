<script lang="ts">
// A small stroked icon set, in the spirit of the Firefish/Misskey UI.
// Ported Firefish markup uses Phosphor class names (`<i class="ph-house">`);
// the port maps each of those to one of these names instead of shipping the
// icon font. Several Phosphor variants share a name (ph-heart / ph-heart-fill
// both map to `heart`).
//
// The paths live in a normal script block so their names can be exported:
// `AppIconName` keeps the `name` prop a literal union (a typo fails typecheck),
// and `iconPaths` lets the menu translate an incoming icon name at runtime.
export const iconPaths = {
  home: "M3 10.5 12 3l9 7.5M5 9.5V21h5v-6h4v6h5V9.5",
  bell: "M6 8a6 6 0 1 1 12 0c0 5 2 6 2 6H4s2-1 2-6Zm4 9a2 2 0 0 0 4 0",
  compass: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm3.5-12.5-2 5-5 2 2-5 5-2Z",
  settings:
    "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Zm8-3a8 8 0 0 0-.2-1.8l2-1.5-2-3.4-2.3 1a8 8 0 0 0-3-1.7L14 2h-4l-.5 2.6a8 8 0 0 0-3 1.7l-2.3-1-2 3.4 2 1.5A8 8 0 0 0 4 12c0 .6.1 1.2.2 1.8l-2 1.5 2 3.4 2.3-1a8 8 0 0 0 3 1.7L10 22h4l.5-2.6a8 8 0 0 0 3-1.7l2.3 1 2-3.4-2-1.5c.1-.6.2-1.2.2-1.8Z",
  shield: "M12 3 4.5 6v6c0 4.5 3.2 7.8 7.5 9 4.3-1.2 7.5-4.5 7.5-9V6L12 3Z",
  display: "M3 4h18v12H3zM8 20h8M12 16v4",
  flame: "M12 22c4 0 7-2.8 7-6.6 0-3.1-1.9-5.2-3.6-7.2C13.8 6.3 12.7 4.2 13 2c-2.6 1-4.3 3.3-4.6 6.1-.9-.5-1.6-1.3-2-2.3C4.9 7.3 5 9.9 5 12.2 5 18 8 22 12 22Z",
  user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-8 9a8 8 0 0 1 16 0",
  users:
    "M9 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Zm-7 8a7 7 0 0 1 14 0M16 4.5a3.5 3.5 0 0 1 0 7M17.6 12.4a5.5 5.5 0 0 1 3.4 5.1",
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
  columns: "M4 4h7v16H4zM13 4h7v16h-7z",
  eye: "M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7Zm10 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z",
  eyeSlash: "M3 3l18 18M10.6 5.2A9.8 9.8 0 0 1 12 5c5 0 9 4.5 9 7 0 .9-.6 2-1.7 3.1M6.4 6.4C4.2 7.8 3 9.8 3 12c0 2.5 4 7 9 7 1.6 0 3-.4 4.3-1.1M9.9 9.9a3 3 0 0 0 4.2 4.2",
  trash: "M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6",
  pencil: "m4 20 1-5L16 4l4 4L9 19l-5 1Z",
  warning: "M12 3 2 21h20L12 3Zm0 6v6m0 3h.01",
  info: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm0-9v5m0-8h.01",
  question: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm-2-12a2 2 0 1 1 3 1.7c-.7.5-1 1-1 1.8m0 3h.01",
  download: "M12 3v12m-5-5 5 5 5-5M4 21h16",
  upload: "M12 21V9m-5 5 5-5 5 5M4 3h16",
  link: "M10.5 13.5a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1.6 1.6m-.4 5.1a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1.6-1.6",
  externalLink: "M14 3h7v7M21 3l-9 9M19 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h5",
  smiley: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18ZM8.5 14a4 4 0 0 0 7 0M9 9.5h.01M15 9.5h.01",
  clock: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm0-13v5l3 2",
  star: "m12 3 2.8 5.7 6.2.9-4.5 4.4 1 6.1L12 17.2 6.5 20.1l1-6.1L3 9.6l6.2-.9L12 3Z",
  save: "M4 4h11l5 5v11H4V4Zm4 0v6h7V4M8 20v-6h8v6",
  hash: "M9 3 7 21M17 3l-2 18M4 9h16M3 15h16",
  spinner: "M12 3a9 9 0 1 1-9 9",
  refresh: "M20 12a8 8 0 1 1-2.3-5.7M20 4v5h-5",
  list: "M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01",
  menu: "M4 6h16M4 12h16M4 18h16",
  filter: "M3 5h18l-7 8v6l-4 2v-8L3 5Z",
  pin: "M9 3h6l-1 7 4 4H6l4-4-1-7Zm3 11v7",
  code: "m9 8-4 4 4 4m6-8 4 4-4 4",
  cloud: "M7 18a4 4 0 0 1 0-8 5.5 5.5 0 0 1 10.5-1.5A3.8 3.8 0 0 1 18 18H7Z",
  caretDown: "m6 9 6 6 6-6",
  caretUp: "m6 15 6-6 6 6",
  caretRight: "m9 6 6 6-6 6",
} as const;

export type AppIconName = keyof typeof iconPaths;
</script>

<script setup lang="ts">
defineProps<{ name: AppIconName }>();
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
    <path :d="iconPaths[name]" />
  </svg>
</template>

<style scoped>
.icon {
  width: 1.2em;
  height: 1.2em;
  display: block;
}
</style>
