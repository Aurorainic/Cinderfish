<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/statusbar-federation.vue (AGPL-3.0-only) -->
<template>
	<span class="cf-statusbar-federation">
		<span v-if="fetching" class="state">读取中…</span>
		<span v-else-if="instances.length === 0" class="state">
			还没有收录的远端实例
		</span>
		<span v-else class="track" :style="trackStyle">
			<span
				v-for="(instance, i) in loop"
				:key="`${instance.host}-${i}`"
				class="item"
				:class="{ colored }"
				:style="itemStyle(instance)"
			>
				<span class="host _monospace">{{ instance.host }}</span>
				<span class="divider"></span>
			</span>
		</span>
	</span>
</template>

<script lang="ts" setup>
// Firefish's federation status bar scrolled every known remote instance. The
// port keeps that shape on top of `federation/instances` (wave 32): the data is
// already local, so this polls nothing remote. Two differences, both forced by
// the API: there is no instance icon (the endpoint returns none) and no
// `oneByOne` display mode (upstream never implemented it either).
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { api } from "@/api/client";
import type { FederationInstance } from "@/api/types";

const props = withDefaults(
	defineProps<{
		display?: "marquee" | "oneByOne";
		colored?: boolean;
		marqueeDuration?: number;
		marqueeReverse?: boolean;
		refreshIntervalSec?: number;
	}>(),
	{
		display: "marquee",
		colored: false,
		marqueeDuration: 40,
		marqueeReverse: false,
		refreshIntervalSec: 300,
	},
);

const instances = ref<FederationInstance[]>([]);
const fetching = ref(true);

// The track is duplicated so the animation can wrap seamlessly.
const loop = computed<FederationInstance[]>(() => {
	if (instances.value.length === 0) return [];
	return [...instances.value, ...instances.value];
});

const trackStyle = computed(() => ({
	animationDuration: `${Math.max(5, props.marqueeDuration)}s`,
	animationDirection: props.marqueeReverse ? "reverse" : "normal",
}));

function itemStyle(instance: FederationInstance): Record<string, string> {
	if (!props.colored) return {};
	// The endpoint has no theme colour, so the tint is derived from the host
	// name: stable per instance, and it makes the bar readable at a glance.
	let hash = 0;
	for (let i = 0; i < instance.host.length; i++) {
		hash = (hash * 31 + instance.host.charCodeAt(i)) % 360;
	}
	return { background: `hsl(${hash} 55% 42% / 0.18)` };
}

async function tick(): Promise<void> {
	try {
		instances.value = await api("federation/instances", {
			sort: "+lastCommunicatedAt",
			limit: 30,
		});
	} catch {
		// A failing poll keeps the previous list; the bar must not flash empty.
	} finally {
		fetching.value = false;
	}
}

let timer: number | undefined;

function restart(): void {
	if (typeof window === "undefined") return;
	if (timer !== undefined) window.clearInterval(timer);
	timer = window.setInterval(
		() => void tick(),
		Math.max(5, props.refreshIntervalSec) * 1000,
	);
}

watch(() => props.refreshIntervalSec, restart);
restart();
void tick();

onBeforeUnmount(() => {
	if (typeof window !== "undefined" && timer !== undefined) {
		window.clearInterval(timer);
	}
});
</script>

<style lang="scss" scoped>
.cf-statusbar-federation {
	display: block;
	overflow: hidden;
	height: 100%;

	.state {
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
		padding-inline-start: 8px;
	}

	.track {
		display: inline-flex;
		align-items: center;
		height: 100%;
		white-space: nowrap;
		animation-name: cf-statusbar-scroll;
		animation-timing-function: linear;
		animation-iteration-count: infinite;
	}

	.item {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 0 10px;

		&.colored {
			border-radius: 4px;
		}

		.host {
			font-size: 0.85em;
		}

		.divider {
			width: 1px;
			height: 12px;
			background: var(--divider);
		}
	}
}

@keyframes cf-statusbar-scroll {
	from {
		transform: translateX(0);
	}
	to {
		transform: translateX(-50%);
	}
}

@media (prefers-reduced-motion: reduce) {
	.track {
		animation: none;
	}
}
</style>
