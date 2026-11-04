<template>
	<fieldset class="novjtcto">
		<legend v-if="$slots.label" class="label"><slot name="label"></slot></legend>
		<div class="body"><slot></slot></div>
		<div v-if="$slots.caption" class="caption"><slot name="caption"></slot></div>
	</fieldset>
</template>

<script setup lang="ts">
import { computed, provide } from "vue";
import { RADIOS_KEY, type RadioValue } from "./radios-key";

const props = withDefaults(
	defineProps<{
		modelValue: RadioValue;
	}>(),
	{
		modelValue: null,
	},
);

const emit = defineEmits<{
	"update:modelValue": [value: RadioValue];
}>();

const model = computed<RadioValue>(() => props.modelValue);

function select(value: RadioValue): void {
	emit("update:modelValue", value);
}

// Children read the group model through injection; this mirrors Firefish's
// clone-vnode approach without needing to introspect slot VNodes.
provide(RADIOS_KEY, { model, select });
</script>

<style lang="scss" scoped>
.novjtcto {
	border: 0;
	padding: 0;
	margin: 0;

	> .label {
		font-size: 0.85em;
		padding: 0 0 8px;
		user-select: none;

		&:empty {
			display: none;
		}
	}

	> .body {
		display: flex;
		gap: 12px;
		flex-wrap: wrap;
	}

	> .caption {
		font-size: 0.85em;
		padding-block-start: 8px;
		color: var(--fgTransparentWeak);

		&:empty {
			display: none;
		}
	}
}
</style>
