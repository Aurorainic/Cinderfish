<!-- Ported from firefish@b425f61:packages/client/src/components/form/input.vue (AGPL-3.0-only) -->
<template>
	<CfInput
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
		:step="step"
		:datalist="datalist"
		:inline="inline"
		:debounce="debounce"
		:manual-save="manualSave"
		:small="small"
		:large="large"
		@update:model-value="onUpdate"
		@change="onChange"
		@keydown="onKeydown"
		@enter="onEnter"
	>
		<template #label><slot name="label" /></template>
		<template #prefix><slot name="prefix" /></template>
		<template #suffix><slot name="suffix" /></template>
		<template #caption><slot name="caption" /></template>
	</CfInput>
</template>

<script setup lang="ts">
// Wave-3 form layer. Firefish imports this file as `@/components/form/input.vue`
// from every settings page; the widget itself was already ported verbatim in
// wave 2 as `ui/CfInput.vue` (same upstream file), so this is a pass-through
// wrapper that keeps the upstream import paths alive instead of a second copy.
//
// One difference: `max` is a real prop here (upstream declares none and lets the
// attribute fall through to the wrapper <div>, where it does nothing). The base
// component has no `maxlength` plumbing, so the emitted value is truncated —
// which is what a settings field wants anyway.
import CfInput from "@/components/ui/CfInput.vue";

const props = withDefaults(
	defineProps<{
		modelValue: string | number | null;
		type?:
			| "text"
			| "number"
			| "password"
			| "email"
			| "url"
			| "date"
			| "time"
			| "search";
		required?: boolean;
		readonly?: boolean;
		disabled?: boolean;
		pattern?: string;
		placeholder?: string;
		autofocus?: boolean;
		autocomplete?: string;
		spellcheck?: boolean;
		step?: number | string;
		datalist?: string[];
		inline?: boolean;
		debounce?: boolean;
		manualSave?: boolean;
		small?: boolean;
		large?: boolean;
		/** Maximum length of the emitted string value. */
		max?: number;
	}>(),
	{
		type: "text",
		required: false,
		readonly: false,
		disabled: false,
		pattern: undefined,
		placeholder: undefined,
		autofocus: false,
		autocomplete: undefined,
		spellcheck: true,
		step: undefined,
		datalist: undefined,
		inline: false,
		debounce: false,
		manualSave: false,
		small: false,
		large: false,
		max: undefined,
	},
);

const emit = defineEmits<{
	"update:modelValue": [value: string | number | null];
	change: [event: Event];
	keydown: [event: KeyboardEvent];
	enter: [];
}>();

function clamp(value: string | number | null): string | number | null {
	if (props.max == null || typeof value !== "string") return value;
	return Array.from(value).slice(0, props.max).join("");
}

function onUpdate(value: string | number | null): void {
	emit("update:modelValue", clamp(value));
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
