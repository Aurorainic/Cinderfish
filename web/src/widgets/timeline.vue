<!-- Ported from firefish@b425f61:packages/client/src/widgets/timeline.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_timelines.' + src)"
		:icon="iconFor(src)"
		:show-header="widgetProps.showHeader === true"
		:height="height"
		:scrollable="true"
	>
		<template #actions>
			<button class="_button" :aria-label="t('settings')" @click="choose">
				<i class="ph-bold ph-caret-down"></i>
			</button>
		</template>
		<MkTimeline :key="src" :src="src" />
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish also offers antenna and list sources and loads them with
// `antennas/list` / `users/lists/list`. Neither endpoint exists (contract §2),
// so the menu is limited to the four real timelines.
import { computed } from "vue";
import WidgetCard from "./WidgetCard.vue";
import MkTimeline from "@/components/MkTimeline.vue";
import { popupMenu, type MenuItem } from "@/os";
import { t } from "@/i18n";
import type { TimelineSource } from "@/store/base";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "timeline";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const sources: TimelineSource[] = ["home", "local", "social", "global"];

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
	{ key: "height", label: "高度 (px)", type: "number", default: 300, min: 120, max: 800, step: 10 },
	{
		key: "src",
		label: "时间线",
		type: "enum",
		default: "home",
		options: sources.map((source) => ({
			value: source,
			label: t(`_timelines.${source}`),
		})),
	},
];

const { widgetProps, configure, save } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const src = computed<TimelineSource>(() => {
	const value = widgetProps.src;
	return sources.includes(value as TimelineSource) ? (value as TimelineSource) : "home";
});

const height = computed(() => {
	const value = widgetProps.height;
	return typeof value === "number" ? value : 300;
});

function iconFor(source: TimelineSource): string {
	switch (source) {
		case "local":
			return "ph-bold ph-users";
		case "social":
			return "ph-bold ph-handshake";
		case "global":
			return "ph-bold ph-planet";
		case "home":
		default:
			return "ph-bold ph-house";
	}
}

function setSrc(next: TimelineSource): void {
	widgetProps.src = next;
	save();
}

function choose(event: MouseEvent): void {
	const items: MenuItem[] = sources.map((source) => ({
		text: t(`_timelines.${source}`),
		icon: iconFor(source),
		active: source === src.value,
		action: () => setSrc(source),
	}));
	void popupMenu(items, event);
}

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>
