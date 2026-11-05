<!-- Ported from firefish@b425f61:packages/client/src/components/form/select.vue (AGPL-3.0-only) -->
<template>
	<CfSelect
		:model-value="modelValue"
		:required="required"
		:readonly="readonly"
		:disabled="disabled"
		:placeholder="placeholder"
		:autofocus="autofocus"
		:inline="inline"
		:manual-save="manualSave"
		:small="small"
		:large="large"
		@update:model-value="onUpdate"
	>
		<template #label><slot name="label" /></template>
		<template #prefix><slot name="prefix" /></template>
		<template #suffix><slot name="suffix" /></template>
		<template #caption><slot name="caption" /></template>
		<slot />
	</CfSelect>
</template>

<script setup lang="ts">
// Pass-through over `ui/CfSelect.vue`. Upstream renders the option list in an
// os.popupMenu so the closed control can be styled freely; the wave-2 port keeps
// a native <select> (degraded, see docs/research/2026-11-05-firefish-port-wave3
// -contract.md §2), which is why the default slot carries <option> elements.
import CfSelect from "@/components/ui/CfSelect.vue";

defineProps<{
	modelValue: string | null;
	required?: boolean;
	readonly?: boolean;
	disabled?: boolean;
	placeholder?: string;
	autofocus?: boolean;
	inline?: boolean;
	manualSave?: boolean;
	small?: boolean;
	large?: boolean;
}>();

const emit = defineEmits<{
	"update:modelValue": [value: string | null];
}>();

function onUpdate(value: string | null): void {
	emit("update:modelValue", value);
}
</script>
