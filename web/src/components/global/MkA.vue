<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkA.vue (AGPL-3.0-only) -->
<template>
	<a
		:href="href"
		:class="active ? activeClass : null"
		:target="target ?? (isExternal ? '_blank' : undefined)"
		:rel="rel ?? (isExternal ? 'noopener' : undefined)"
		@contextmenu.prevent.stop="onContextmenu"
		@click.stop="nav"
	>
		<slot></slot>
		<i
			v-if="showIcon && isExternal"
			class="ph-arrow-square-out ph-bold ph-lg icon"
		></i>
	</a>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { contextMenu, success } from "../../os";
import { t } from "../../i18n";
import { url as local } from "../../config";

// Firefish's MkA is the router-aware link used everywhere in note bodies and
// page chrome. Cinderfish merges the "internal route" and "external URL" cases
// into one component ("to" for router paths, "url" for absolute links) so the
// MFM renderer, MkUrl/MkLink and the note tree share one implementation.
const props = withDefaults(
	defineProps<{
		/** Router path (internal link). */
		to?: string | null;
		/** Absolute URL (external link). */
		url?: string | null;
		target?: string | null;
		rel?: string | null;
		behavior?: "window" | "browser" | "modalWindow" | null;
		showIcon?: boolean;
		activeClass?: string | null;
	}>(),
	{
		to: null,
		url: null,
		target: null,
		rel: null,
		behavior: null,
		showIcon: false,
		activeClass: null,
	},
);

const router = useRouter();
const route = useRoute();

const isExternal = computed(
	() => props.url != null && (props.to == null || props.url !== ""),
);

const href = computed<string>(() =>
	isExternal.value ? (props.url ?? "") : (props.to ?? ""),
);

const active = computed<boolean>(() => {
	if (props.activeClass == null) return false;
	const to = props.to;
	if (to == null) return false;
	const resolved = router.resolve(to);
	if (resolved == null) return false;
	if (resolved.path === route.path) return true;
	if (resolved.name == null || route.name == null) return false;
	return resolved.name === route.name;
});

function copyLink(): void {
	const text = isExternal.value ? (props.url ?? "") : local + (props.to ?? "");
	void navigator.clipboard.writeText(text).then(
		() => success(),
		() => {
			// Clipboard permission denied; keep the URL reachable in the console.
			console.info(text);
		},
	);
}

function openWindow(): void {
	window.open(href.value, "_blank", "noopener");
}

function onContextmenu(ev: MouseEvent): void {
	const items: {
		type?: "label";
		icon?: string;
		text?: string;
		action?: () => void;
	}[] = [];
	if (!isExternal.value) {
		items.push({
			icon: "ph-bold ph-browser",
			text: t("openInWindow"),
			action: openWindow,
		});
		items.push({
			icon: "ph-bold ph-arrows-out-simple",
			text: t("showInPage"),
			action: () => {
				void router.push(props.to ?? "/");
			},
		});
	}
	items.push({
		icon: "ph-bold ph-arrow-square-out",
		text: t("openInNewTab"),
		action: openWindow,
	});
	items.push({
		icon: "ph-bold ph-link-simple",
		text: t("copyLink"),
		action: copyLink,
	});
	void contextMenu(items, ev);
}

function nav(ev: MouseEvent): void {
	if (isExternal.value) return;
	if (props.behavior === "browser") return;
	// Modified clicks (new tab / new window) are left to the browser.
	if (ev.ctrlKey || ev.metaKey || ev.button !== 0) return;
	ev.preventDefault();
	if (props.behavior === "window" || props.behavior === "modalWindow") {
		openWindow();
		return;
	}
	if (ev.shiftKey) {
		openWindow();
		return;
	}
	void router.push(props.to ?? "/");
}
</script>

<style lang="scss" scoped>
.icon {
	padding-left: 2px;
	font-size: 0.9em;
}
</style>
