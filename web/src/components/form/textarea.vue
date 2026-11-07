<!-- Ported from firefish@b425f61:packages/client/src/components/form/textarea.vue (AGPL-3.0-only) -->
<template>
	<CfTextarea
		:model-value="modelValue"
		:type="type"
		:required="required"
		:readonly="readonly"
		:disabled="disabled"
		:pattern="pattern"
		:placeholder="placeholder"
		:autofocus="autofocus"
		:autocomplete="autocomplete"
		:spellcheck="spellcheck"
		:code="code"
		:tall="tall"
		:pre="pre"
		:debounce="debounce"
		:manual-save="manualSave"
		@update:model-value="onUpdate"
		@change="onChange"
		@keydown="onKeydown"
		@enter="onEnter"
	>
		<template #label><slot name="label" /></template>
		<template #caption><slot name="caption" /></template>
	</CfTextarea>
</template>

<script setup lang="ts">
// Pass-through over `ui/CfTextarea.vue` (the wave-2 port of this same upstream
// file); see form/input.vue for why the form layer exists. `max` truncates the
// emitted string because the base component has no maxlength support.
import CfTextarea from "@/components/ui/CfTextarea.vue";

const props = withDefaults(
	defineProps<{
		modelValue: string;
		type?: string;
		required?: boolean;
		readonly?: boolean;
		disabled?: boolean;
		pattern?: string;
		placeholder?: string;
		autofocus?: boolean;
		autocomplete?: string;
		spellcheck?: boolean;
		code?: boolean;
		tall?: boolean;
		pre?: boolean;
		debounce?: boolean;
		manualSave?: boolean;
		/** Maximum length of the emitted string value. */
		max?: number;
	}>(),
	{
		type: undefined,
		required: false,
		readonly: false,
		disabled: false,
		pattern: undefined,
		placeholder: undefined,
		autofocus: false,
		autocomplete: undefined,
		spellcheck: true,
		code: false,
		tall: false,
		pre: false,
		debounce: false,
		manualSave: false,
		max: undefined,
	},
);

const emit = defineEmits<{
	"update:modelValue": [value: string];
	change: [event: Event];
	keydown: [event: KeyboardEvent];
	enter: [];
}>();

function onUpdate(value: string): void {
	emit("update:modelValue", props.max == null ? value : Array.from(value).slice(0, props.max).join(""));
}

function onChange(event: Event): void {
	emit("change", event);
}

function onKeydown(event: KeyboardEvent): void {
	emit("keydown", event);
}

function onEnter(): void {
	emit("enter");
}
</script>
