<!-- Ported from firefish@b425f61:packages/client/src/components/MkInstanceTicker.vue (AGPL-3.0-only) -->
<template>
	<div class="hpaizdrt" :title="capitalize(info.softwareName)" :style="bg">
		<img class="icon" :src="icon" aria-hidden="true" />
		<span class="name">{{ info.name }}</span>
	</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { instance as Instance } from "@/instance";
import { resolveMediaUrl } from "@/api/client";

/**
 * Ported from Firefish. Two contract differences:
 *   - note.user.instance is always null (docs/research/api-contract/
 *     auth-users.md), so a remote user renders the local instance ticker; the
 *     ticker is hidden by the caller (MkNoteHeader) when the setting says so.
 *   - There is no media proxy (@/scripts/media-proxy) and no softwareName in
 *     /api/meta, so the instance icon is used as-is and the software name is
 *     the product name.
 */
const props = defineProps<{
	instance?: {
		faviconUrl?: string | null;
		iconUrl?: string | null;
		name: string;
		themeColor?: string | null;
		softwareName?: string | null;
	} | null;
}>();

const info = computed(() => {
	const given = props.instance;
	if (given != null) {
		return {
			iconUrl: given.iconUrl ?? given.faviconUrl ?? null,
			name: given.name,
			themeColor: given.themeColor ?? null,
			softwareName: given.softwareName ?? "Cinderfish",
		};
	}
	// No instance data given: this is for the local instance.
	return {
		iconUrl: Instance.iconUrl ?? null,
		name: Instance.name,
		themeColor: Instance.themeColor ?? null,
		softwareName: "Cinderfish",
	};
});

const icon = computed(
	() => resolveMediaUrl(info.value.iconUrl) || "/favicon.ico",
);

const capitalize = (s: string): string =>
	s ? s[0].toUpperCase() + s.slice(1) : s;

const bg = computed(() => {
	const fallback =
		typeof document === "undefined"
			? ""
			: getComputedStyle(document.documentElement).getPropertyValue("--bg");
	const themeColor = info.value.themeColor || fallback || "transparent";
	return {
		background: `linear-gradient(90deg, ${themeColor}, ${themeColor}55)`,
	};
});
</script>

<style lang="scss" scoped>
.hpaizdrt {
	display: flex;
	align-items: center;
	height: 1.1em;
	justify-self: flex-end;
	padding: 0.2em 0.4em;
	border-radius: 100px;
	font-size: 0.8em;
	text-shadow: 0 2px 2px var(--shadow);
	overflow: hidden;
	.header > .body & {
		width: max-content;
		max-width: 100%;
	}

	> .icon {
		height: 100%;
		border-radius: 0.3rem;
	}

	> .name {
		display: none;
		margin-left: 4px;
		font-size: 0.85em;
		vertical-align: top;
		font-weight: bold;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-shadow:
			-1px -1px 0 var(--bg),
			1px -1px 0 var(--bg),
			-1px 1px 0 var(--bg),
			1px 1px 0 var(--bg);
		.article > .main &,
		.header > .body & {
			display: unset;
		}
	}
}
</style>