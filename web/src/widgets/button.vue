<!-- Ported from firefish@b425f61:packages/client/src/widgets/button.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="wrap">
			<button
				class="btn"
				:class="{ colored: widgetProps.colored !== false }"
				type="button"
				@click="run"
			>
				{{ widgetProps.label || "BUTTON" }}
			</button>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish's button widget runs an AiScript program on click. AiScript is not a
// dependency of this port, so the widget is a link button instead: `url` opens
// the address in a new tab. The label and colour options keep their upstream
// meaning, and the AiScript-only `script` field is not offered.
import WidgetCard from "./WidgetCard.vue";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "button";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "label", label: "按钮文字", type: "text", default: "BUTTON" },
	{ key: "url", label: "链接（留空则不跳转）", type: "text", default: "" },
	{ key: "colored", label: "使用主题色", type: "boolean", default: true },
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

function run(): void {
	const url = String(widgetProps.url ?? "").trim();
	if (url === "") return;
	// Only http(s) links are followed, so a stored javascript: URL cannot
	// execute in this origin.
	const parsed = new URL(url, location.href);
	if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return;
	window.open(parsed.href, "_blank", "noopener,noreferrer");
}

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.wrap {
	padding: 8px;
}

.btn {
	display: block;
	width: 100%;
	padding: 10px 12px;
	border: none;
	border-radius: var(--radius);
	background: var(--panel);
	color: var(--fg);
	font: inherit;
	cursor: pointer;

	&.colored {
		background: var(--accent, #86b300);
		color: var(--fg-on-accent, #fff);
	}

	&:focus-visible {
		outline: 2px solid var(--accent, #86b300);
		outline-offset: 2px;
	}
}
</style>
