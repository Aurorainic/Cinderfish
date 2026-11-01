// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkA.vue
//
// Firefish builds its own router on top of vue-router and renders a plain
// <a>; here the standard RouterLink is used and only middle/ctrl-free plain
// clicks are intercepted for window behaviour. The right-click "open in
// window" context menu is dropped: TODO Cinderfish lacks the os.contextMenu
// / os.pageWindow runtime surface that menu depends on.
<template>
	<RouterLink
		:to="to"
		:class="active ? activeClass : undefined"
		@click="nav"
	>
		<slot></slot>
	</RouterLink>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { RouterLink, useRouter } from "vue-router";

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	copyLink: "复制链接",
};

const props = withDefaults(
	defineProps<{
		to: string;
		activeClass?: null | string;
		// "window" / "modalWindow" degrade to normal navigation:
		// TODO Cinderfish lacks the os.pageWindow windowing runtime.
		behavior?: null | "window" | "browser" | "modalWindow";
	}>(),
	{
		activeClass: null,
		behavior: null,
	},
);

const router = useRouter();

const active = computed(() => {
	if (props.activeClass == null) return false;
	const resolved = router.resolve(props.to);
	if (resolved == null) return false;
	if (resolved.path === router.currentRoute.value.path) return true;
	if (resolved.name == null) return false;
	if (router.currentRoute.value.name == null) return false;
	return resolved.name === router.currentRoute.value.name;
});

function nav(ev: MouseEvent) {
	if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.button !== 0) return;
	if (props.behavior === "browser") return;
	// RouterLink performs the navigation itself for plain clicks; shift-click
	// opens a browser tab natively. The "window" behaviors degrade to plain
	// navigation (see TODO above).
	if (props.behavior === "window" || props.behavior === "modalWindow") {
		ev.preventDefault();
		router.push(props.to);
	}
}
</script>
