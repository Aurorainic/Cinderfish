<template>
	<div class="vblkjoeq">
		<div class="label"><slot name="label"></slot></div>
		<div class="input" :class="{ inline, disabled, focused }">
			<div v-if="$slots.prefix" class="prefix"><slot name="prefix"></slot></div>
			<select
				ref="inputEl"
				v-model="value"
				class="select"
				:disabled="disabled"
				:required="required"
				@focus="focused = true"
				@blur="focused = false"
				@change="onChange"
			>
				<slot></slot>
			</select>
			<div class="suffix">
				<slot name="suffix"></slot>
				<i class="ph-bold ph-caret-down"></i>
			</div>
		</div>
		<div class="caption"><slot name="caption"></slot></div>

		<CfButton v-if="manualSave && changed" primary class="save" @click="commit">
			{{ t("save") }}
		</CfButton>
	</div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from "vue";
import CfButton from "./CfButton.vue";
import { t } from "../../i18n";

const props = withDefaults(
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
	}>(),
	{
		required: false,
		readonly: false,
		disabled: false,
		placeholder: undefined,
		autofocus: false,
		inline: false,
		manualSave: false,
		small: false,
		large: false,
	},
);

const emit = defineEmits<{
	change: [event: Event];
	"update:modelValue": [value: string | null];
}>();

const value = ref<string | null>(props.modelValue);
const focused = ref(false);
const changed = ref(false);
const inputEl = ref<HTMLSelectElement | null>(null);

function onChange(event: Event): void {
	changed.value = true;
	emit("change", event);
	if (!props.manualSave) {
		emit("update:modelValue", value.value);
	}
}

function commit(): void {
	changed.value = false;
	emit("update:modelValue", value.value);
}

watch(
	() => props.modelValue,
	(next) => {
		value.value = next;
	},
);

onMounted(() => {
	nextTick(() => {
		if (props.autofocus) inputEl.value?.focus();
	});
});
</script>

<style lang="scss" scoped>
.vblkjoeq {
	.label {
		font-size: 0.85em;
		padding-block-end: 8px;
		user-select: none;

		&:empty {
			display: none;
		}
	}

	.input {
		position: relative;
		display: flex;
		align-items: center;
		border: solid 1px var(--inputBorder);
		border-radius: 6px;
		background: var(--bg);
		transition: border-color 0.2s;

		&.focused {
			border-color: var(--accent);
		}

		&.disabled {
			opacity: 0.6;
		}

		&.inline {
			display: inline-flex;
		}
	}

	.select {
		flex: 1;
		min-inline-size: 0;
		padding: 9px 36px 9px 12px;
		background: none;
		border: none;
		outline: none;
		color: var(--fg);
		font: inherit;
		appearance: none;
		cursor: pointer;
	}

	.prefix {
		padding-inline-start: 10px;
		color: var(--fgTransparentWeak);
	}

	.suffix {
		position: absolute;
		inset-inline-end: 10px;
		display: flex;
		align-items: center;
		color: var(--fgTransparentWeak);
		pointer-events: none;
	}

	.caption {
		padding-block-start: 6px;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);

		&:empty {
			display: none;
		}
	}

	.save {
		margin-block-start: 8px;
	}
}
</style>
