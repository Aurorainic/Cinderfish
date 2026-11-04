<template>
	<div class="adhpbeos">
		<label>
			<span class="label"><slot name="label"></slot></span>
			<div class="input" :class="{ disabled, focused, tall, pre }">
				<textarea
					ref="inputEl"
					v-model="value"
					:class="{ code, _monospace: code }"
					:disabled="disabled"
					:required="required"
					:readonly="readonly"
					:placeholder="placeholder"
					:pattern="pattern"
					:autocomplete="autocomplete"
					:spellcheck="spellcheck"
					@focus="focused = true"
					@blur="focused = false"
					@keydown="onKeydown"
					@input="onInput"
				></textarea>
			</div>
			<div class="caption"><slot name="caption"></slot></div>
		</label>

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
	},
);

const emit = defineEmits<{
	change: [event: Event];
	keydown: [event: KeyboardEvent];
	enter: [];
	"update:modelValue": [value: string];
}>();

const value = ref(props.modelValue);
const focused = ref(false);
const changed = ref(false);
const inputEl = ref<HTMLTextAreaElement | null>(null);

let debounceTimer: number | null = null;

function focus(): void {
	inputEl.value?.focus();
}

function onInput(event: Event): void {
	changed.value = true;
	emit("change", event);
}

function onKeydown(event: KeyboardEvent): void {
	emit("keydown", event);
	if (event.code === "Enter") emit("enter");
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

watch(value, () => {
	if (props.manualSave) return;
	if (props.debounce) {
		if (debounceTimer != null) window.clearTimeout(debounceTimer);
		debounceTimer = window.setTimeout(commit, 1000);
	} else {
		commit();
	}
});

onMounted(() => {
	nextTick(() => {
		if (props.autofocus) focus();
	});
});

defineExpose({ focus });
</script>

<style lang="scss" scoped>
.adhpbeos {
	.label {
		display: block;
		font-size: 0.85em;
		padding-block-end: 8px;
		user-select: none;

		&:empty {
			display: none;
		}
	}

	.input {
		border: solid 1px var(--inputBorder);
		border-radius: 6px;
		background: var(--bg);
		overflow: hidden;
		transition: border-color 0.2s;

		&.focused {
			border-color: var(--accent);
		}

		&.disabled {
			opacity: 0.6;
		}

		&.tall textarea {
			min-block-size: 200px;
		}

		&.pre textarea {
			white-space: pre;
		}

		textarea {
			display: block;
			inline-size: 100%;
			min-block-size: 100px;
			padding: 9px 12px;
			background: none;
			border: none;
			outline: none;
			resize: vertical;
			color: var(--fg);
			font: inherit;
			line-height: 1.5;
		}
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
