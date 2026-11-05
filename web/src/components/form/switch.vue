<!-- Ported from firefish@b425f61:packages/client/src/components/form/switch.vue (AGPL-3.0-only) -->
<template>
	<CfSwitch
		:model-value="modelValue"
		:disabled="disabled"
		@update:model-value="onUpdate"
	>
		<template #label><slot name="label" /></template>
		<template #caption><slot name="caption" /></template>
		<slot />
	</CfSwitch>
</template>

<script setup lang="ts">
// Pass-through over `ui/CfSwitch.vue` (same upstream file, ported in wave 2).
// The default slot stays alongside the labelled slot because upstream call
// sites still write `<FormSwitch v-model="x">label text</FormSwitch>`.
import CfSwitch from "@/components/ui/CfSwitch.vue";

defineProps<{
	modelValue: boolean;
	disabled?: boolean;
}>();

const emit = defineEmits<{
	"update:modelValue": [value: boolean];
}>();

function onUpdate(value: boolean): void {
	emit("update:modelValue", value);
}
</script>
