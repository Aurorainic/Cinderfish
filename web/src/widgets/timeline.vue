<!-- Ported from firefish@b425f61:packages/client/src/widgets/timeline.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="title"
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
		<MkTimeline
			:key="timelineKey"
			:src="src"
			:antenna-id="antennaId ?? undefined"
		/>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish also offers list sources and loads them with `users/lists/list`;
// that menu entry is not ported. Antennas are: `antennas/list` exists and the
// widget stores the chosen antenna id/name alongside the source, so the title
// can show the antenna name without a second request.
import { computed } from "vue";
import WidgetCard from "./WidgetCard.vue";
import MkTimeline from "@/components/MkTimeline.vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { popupMenu, type MenuItem } from "@/os";
import { t } from "@/i18n";
import type { TimelineSource } from "@/store/base";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "timeline";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

/** Persisted sources plus the non-persisted antenna source. */
type WidgetSource = TimelineSource | "antenna";

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

const src = computed<WidgetSource>(() => {
	const value = widgetProps.src;
	if (value === "antenna" && antennaId.value != null) return "antenna";
	return sources.includes(value as TimelineSource)
		? (value as TimelineSource)
		: "home";
});

const antennaId = computed<string | null>(() => {
	const value = widgetProps.antennaId;
	return typeof value === "string" && value !== "" ? value : null;
});

const antennaName = computed<string | null>(() => {
	const value = widgetProps.antennaName;
	return typeof value === "string" && value !== "" ? value : null;
});

const title = computed(() =>
	src.value === "antenna"
		? (antennaName.value ?? t("antennas"))
		: t(`_timelines.${src.value}`),
);

const timelineKey = computed(() =>
	src.value === "antenna" ? `antenna:${antennaId.value}` : src.value,
);

const height = computed(() => {
	const value = widgetProps.height;
	return typeof value === "number" ? value : 300;
});

function iconFor(source: WidgetSource): string {
	switch (source) {
		case "local":
			return "ph-bold ph-users";
		case "social":
			return "ph-bold ph-handshake";
		case "global":
			return "ph-bold ph-planet";
		case "antenna":
			return "ph-bold ph-flying-saucer";
		case "home":
		default:
			return "ph-bold ph-house";
	}
}

function setSrc(next: WidgetSource): void {
	widgetProps.src = next;
	save();
}

async function choose(event: MouseEvent): Promise<void> {
	const items: MenuItem[] = sources.map((source) => ({
		text: t(`_timelines.${source}`),
		icon: iconFor(source),
		active: source === src.value,
		action: () => setSrc(source),
	}));

	try {
		const antennas = await api(ENDPOINTS.antennasList, {});
		if (antennas.length > 0) {
			items.push({ type: "label", text: t("antennas") });
			for (const antenna of antennas) {
				items.push({
					text: antenna.name,
					icon: "ph-bold ph-flying-saucer",
					action: () => {
						widgetProps.antennaId = antenna.id;
						widgetProps.antennaName = antenna.name;
						setSrc("antenna");
					},
				});
			}
		}
	} catch {
		items.push({ type: "label", text: t("somethingHappened") });
	}

	void popupMenu(items, event);
}

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>
