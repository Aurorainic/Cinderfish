// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkInstanceTicker.vue
//
// The pill naming the instance a remote user posts from. Cinderfish's
// UserLite.instance carries only `name` and `faviconUrl`: no themeColor, no
// softwareName/softwareVersion, so the tooltip degrades to the instance name
// and the gradient falls back to the theme's --bg.
<template>
	<div v-if="show" class="ticker" :style="bg">
		<img class="icon" :src="iconUrl" aria-hidden="true" @error="onIconError" />
		<span class="name">{{ instance.name }}</span>
	</div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { getInstanceInfo } from "../../instance";

const props = defineProps<{
	instance?: { name?: string; faviconUrl?: string } | null;
}>();

// A local user carries no instance object: like the original's assumption,
// that renders the local instance's ticker.
const show = computed(() => !!props.instance && !!props.instance.name);

const instance = computed(() => ({
	name: props.instance?.name ?? getInstanceInfo().name ?? "",
	faviconUrl: props.instance?.faviconUrl ?? getInstanceInfo().iconUrl ?? "/favicon.ico",
}));

const iconUrl = computed(() => instance.value.faviconUrl);

const bg = computed(() => {
	const themeColor =
		getComputedStyle(document.documentElement).getPropertyValue("--bg") || "transparent";
	return {
		background: `linear-gradient(90deg, ${themeColor}, ${themeColor}55)`,
	};
});

function onIconError(event: Event) {
	const target = event.target as HTMLImageElement;
	target.src = "/favicon.ico";
}
</script>

<style scoped>
.ticker {
	display: inline-flex;
	align-items: center;
	block-size: 1.1em;
	padding-block: 0.2em;
	padding-inline: 0.4em;
	border-radius: 100px;
	font-size: 0.8em;
	overflow: hidden;
	max-inline-size: 100%;
}

.icon {
	block-size: 100%;
	border-radius: 0.3rem;
}

.name {
	display: none;
	margin-inline-start: 4px;
	font-size: 0.85em;
	font-weight: bold;
	text-overflow: ellipsis;
	white-space: nowrap;
	overflow: hidden;
	text-shadow: -1px -1px 0 var(--bg), 1px -1px 0 var(--bg), -1px 1px 0 var(--bg), 1px 1px 0 var(--bg);
}

/* Inside the note body the name shows, as in the original's .article rule. */
.article .name,
.note-detail .name {
	display: unset;
}
</style>
