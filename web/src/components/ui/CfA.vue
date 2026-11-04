<template>
	<a
		:href="to"
		:class="isActive ? activeClass : undefined"
		@contextmenu.prevent.stop="onContextmenu"
		@click.stop="nav"
	>
		<slot></slot>
	</a>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { contextMenu, success } from "../../os";
import { t } from "../../i18n";

const props = withDefaults(
	defineProps<{
		to: string;
		activeClass?: string | null;
		behavior?: "window" | "browser" | "modalWindow" | null;
	}>(),
	{
		activeClass: null,
		behavior: null,
	},
);

const router = useRouter();
const route = useRoute();

const isActive = computed<boolean>(() => {
	if (props.activeClass == null) return false;
	return route.path === props.to || route.fullPath === props.to;
});

function openWindow(): void {
	window.open(props.to, "_blank", "noopener");
}

function nav(event: MouseEvent): void {
	if (props.behavior === "browser") return;
	if (event.ctrlKey || event.metaKey || event.shiftKey) {
		event.preventDefault();
		openWindow();
		return;
	}
	event.preventDefault();
	if (props.behavior === "window" || props.behavior === "modalWindow") {
		openWindow();
		return;
	}
	void router.push(props.to);
}

async function copyLink(): Promise<void> {
	const url = `${location.origin}${props.to}`;
	try {
		await navigator.clipboard.writeText(url);
		success();
	} catch {
		// Clipboard permission denied; leave the URL in the console.
		console.info(url);
	}
}

function onContextmenu(event: MouseEvent): void {
	void contextMenu(
		[
			{ type: "label", text: props.to },
			{
				icon: "ph-bold ph-browser",
				text: t("openInWindow"),
				action: openWindow,
			},
			{
				icon: "ph-bold ph-arrow-square-out",
				text: t("openInNewTab"),
				action: openWindow,
			},
			{
				icon: "ph-bold ph-link-simple",
				text: t("copyLink"),
				action: () => void copyLink(),
			},
		],
		event,
	);
}
</script>
