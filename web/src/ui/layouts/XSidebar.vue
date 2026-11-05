<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/navbar.vue (AGPL-3.0-only) -->
<template>
	<MkNavbar :icon-only="collapsed" />
</template>

<script setup lang="ts">
// XSidebar is the shell layer around MkNavbar: it decides whether the navbar
// renders full width (Firefish's `sideFull`) or as the 80px icon rail
// (`sideIcon`, which Firefish also forces below 1279px). The rail *is*
// MkNavbar's `iconOnly` state — upstream's `ui/_common_/navbar.vue` renders both
// shapes from one component, so there is no separate rail component here.
//
// DEGRADED: Firefish's `classic` UI (centred column, transparent nav, wallpaper
// mask) is not ported; `defaultStore.menuDisplay === "top"` has no Cinderfish
// implementation to switch to, so only sideFull/sideIcon are honoured.
import { computed, onUnmounted, ref, type Ref } from "vue";
import { store } from "@/store/base";
import MkNavbar from "@/components/MkNavbar.vue";

const props = withDefaults(
	defineProps<{
		/** "always" pins the rail (the deck shell uses it). */
		iconOnly?: "auto" | "always";
	}>(),
	{
		iconOnly: "auto",
	},
);

const NARROW_THRESHOLD = 1279;

const narrow: Ref<boolean> = ref(
	typeof window === "undefined"
		? false
		: window.innerWidth <= NARROW_THRESHOLD,
);

if (typeof window !== "undefined") {
	const media = window.matchMedia(`(max-width: ${NARROW_THRESHOLD}px)`);
	const onChange = (event: MediaQueryListEvent): void => {
		narrow.value = event.matches;
	};
	media.addEventListener("change", onChange);
	onUnmounted(() => media.removeEventListener("change", onChange));
}

const collapsed = computed(
	() =>
		props.iconOnly === "always" ||
		narrow.value ||
		store.state.menuDisplay === "sideIcon",
);
</script>
