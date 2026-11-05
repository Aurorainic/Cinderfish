<!-- Ported from firefish@b425f61:packages/client/src/widgets/post-form.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.postForm')"
		icon="ph-bold ph-pencil-simple"
		:show-header="widgetProps.showHeader === true"
	>
		<MkPostForm class="post-form" :fixed="true" :autofocus="false" />
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish renders XPostForm bare with `fixed = true` (so the form shows its own
// close button) and no MFM cheat sheet toggle. The card header is added here so
// the widget reads like the other cards in the column; the form itself is the
// composer-core component.
import WidgetCard from "./WidgetCard.vue";
import MkPostForm from "@/components/MkPostForm.vue";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "postForm";

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

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.post-form {
	padding: 12px;
}
</style>
