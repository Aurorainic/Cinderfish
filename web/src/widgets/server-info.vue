<!-- Ported from firefish@b425f61:packages/client/src/widgets/server-info.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard :show-header="false" :bare="true">
		<div class="server" :style="bannerStyle">
			<div class="icon">
				<img :src="iconUrl" alt="Instance icon" />
			</div>
			<div class="body">
				<MkA class="name" to="/about">{{ instance.name }}</MkA>
				<div class="host">{{ host }}</div>
			</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish reads `$instance.bannerUrl` / `iconUrl` / `name` and links to /about.
// Cinderfish's meta has `backgroundImageUrl` instead of a banner, and the icon
// falls back to the favicon, so the two fields are resolved here.
import { computed } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { host } from "@/config";
import { instance } from "@/instance";
import { resolveMediaUrl } from "@/api/client";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "serverInfo";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [];

const { configure } = useWidgetPropsManager(name, fields, props.widget, (data) =>
	emit("updateProps", data),
);

const bannerStyle = computed<Record<string, string>>(() => {
	const url = instance.backgroundImageUrl || instance.bannerUrl;
	const style: Record<string, string> = {};
	if (url !== "") style.backgroundImage = `url(${resolveMediaUrl(url)})`;
	return style;
});

const iconUrl = computed(
	() =>
		resolveMediaUrl(instance.iconUrl || instance.faviconUrl) || "/favicon.ico",
);

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.server {
	display: flex;
	background-size: cover;
	background-position: center;

	> .icon {
		padding: 16px;

		> img {
			width: 60px;
			height: 60px;
			border-radius: 8px;
			box-sizing: border-box;
			border: solid 3px var(--panelBorder, var(--divider));
		}
	}

	> .body {
		display: flex;
		flex-direction: column;
		justify-content: center;
		min-width: 0;
		padding: 0 16px 0 0;

		> .name {
			font-weight: bold;
			color: var(--fg);
			text-shadow:
				-1px -1px 0 var(--bg),
				1px -1px 0 var(--bg),
				-1px 1px 0 var(--bg),
				1px 1px 0 var(--bg);
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}

		> .host {
			font-size: 0.85em;
			color: var(--fg);
			opacity: 0.8;
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}
	}
}
</style>
