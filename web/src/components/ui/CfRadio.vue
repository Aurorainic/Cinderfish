<template>
	<label class="novjtctn" :class="{ disabled: isDisabled, checked }">
		<input type="radio" :disabled="isDisabled" :checked="checked" @change="toggle" />
		<span class="button"><span></span></span>
		<span class="label"><slot></slot></span>
	</label>
</template>

<script setup lang="ts">
import { computed, inject } from "vue";
import { RADIOS_KEY, type RadioValue } from "./radios-key";

const props = withDefaults(
	defineProps<{
		modelValue?: RadioValue;
		value: RadioValue;
		disabled?: boolean;
	}>(),
	{
		modelValue: undefined,
		disabled: false,
	},
);

const emit = defineEmits<{
	"update:modelValue": [value: RadioValue];
}>();

const group = inject(RADIOS_KEY, null);

const current = computed<RadioValue>(() =>
	group != null ? group.model.value : (props.modelValue ?? null),
);

const checked = computed(() => current.value === props.value);
const isDisabled = computed(() => props.disabled || (group?.disabled.value ?? false));

function toggle(): void {
	if (isDisabled.value) return;
	if (group != null) group.select(props.value);
	else emit("update:modelValue", props.value);
}
</script>

<style lang="scss" scoped>
.novjtctn {
	display: flex;
	align-items: flex-start;
	gap: 8px;
	cursor: pointer;
	user-select: none;

	> input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}

	> .button {
		position: relative;
		display: inline-flex;
		flex-shrink: 0;
		inline-size: 18px;
		block-size: 18px;
		margin-block-start: 2px;
		border: solid 2px var(--inputBorder);
		border-radius: 50%;
		transition: border-color 0.2s;

		> span {
			position: absolute;
			inset: 3px;
			border-radius: 50%;
			background: var(--accent);
			transform: scale(0);
			transition: transform 0.15s ease;
		}
	}

	> .label {
		font-size: 0.95em;
		line-height: 1.5;
	}

	&.checked > .button {
		border-color: var(--accent);

		> span {
			transform: scale(1);
		}
	}

	&.disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	&:hover:not(.disabled) > .button {
		border-color: var(--inputBorderHover);
	}
}
</style>
