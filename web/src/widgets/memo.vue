<!-- Ported from firefish@b425f61:packages/client/src/widgets/memo.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.memo')"
		icon="ph-bold ph-sticker"
		:show-header="widgetProps.showHeader === true"
	>
		<div class="memo">
			<textarea
				v-model="text"
				class="input"
				placeholder="输入便签内容"
				@input="onChange"
			></textarea>
			<button class="_buttonPrimary save" :disabled="!changed" @click="saveMemo">
				{{ t("save") }}
			</button>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// The memo is stored in `defaultStore.memo`, exactly as in Firefish, so the
// server-info/memo pair stays account-scoped. The debounced write is the same
// behaviour without `throttle-debounce` (contract §2).
import { ref, watch } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { t } from "@/i18n";
import { defaultStore } from "@/store";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "memo";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const text = ref<string | null>(defaultStore.state.memo);
const changed = ref(false);
let timeoutId: number | null = null;

function saveMemo(): void {
	defaultStore.set("memo", text.value);
	changed.value = false;
}

function onChange(): void {
	changed.value = true;
	if (timeoutId != null) window.clearTimeout(timeoutId);
	timeoutId = window.setTimeout(saveMemo, 1000);
}

watch(
	() => defaultStore.reactiveState.memo.value,
	(value) => {
		text.value = value;
	},
);

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.memo {
	position: relative;
	padding-bottom: 44px;

	> .input {
		display: block;
		width: 100%;
		min-height: 96px;
		max-height: 240px;
		padding: 16px;
		box-sizing: border-box;
		color: var(--fg);
		background: transparent;
		border: none;
		border-bottom: solid 0.5px var(--divider);
		border-radius: 0;
		font: inherit;
		font-size: 0.9em;
		resize: vertical;

		&:focus-visible {
			outline: none;
		}
	}

	> .save {
		position: absolute;
		right: 8px;
		bottom: 8px;
		height: 28px;
		padding: 0 10px;
		border-radius: 4px;

		&:disabled {
			opacity: 0.7;
		}
	}
}
</style>
