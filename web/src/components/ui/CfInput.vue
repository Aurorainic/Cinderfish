<template>
	<div class="matxzzsk">
		<label>
			<div class="label"><slot name="label"></slot></div>
			<div v-adaptive-border class="input" :class="{ inline, disabled, focused }">
				<div ref="prefixEl" class="prefix"><slot name="prefix"></slot></div>
				<input
					ref="inputEl"
					v-model="value"
					:type="type"
					:disabled="disabled"
					:required="required"
					:readonly="readonly"
					:placeholder="placeholder"
					:pattern="pattern"
					:autocomplete="autocomplete"
					:spellcheck="spellcheck"
					:step="step"
					:list="listId"
					@focus="focused = true"
					@blur="focused = false"
					@keydown="onKeydown"
					@input="onInput"
				/>
				<datalist v-if="datalist" :id="listId">
					<option v-for="item in datalist" :key="item" :value="item" />
				</datalist>
				<div ref="suffixEl" class="suffix"><slot name="suffix"></slot></div>
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
import { useInterval } from "../../composables/use-interval";
import { t } from "../../i18n";

const props = withDefaults(
	defineProps<{
		modelValue: string | number | null;
		type?: "text" | "number" | "password" | "email" | "url" | "date" | "time" | "search";
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
	},
);

const emit = defineEmits<{
	change: [event: Event];
	keydown: [event: KeyboardEvent];
	enter: [];
	"update:modelValue": [value: string | number | null];
}>();

const value = ref<string | number | null>(props.modelValue);
const listId = `cf-input-${Math.random().toString(36).slice(2)}`;
const focused = ref(false);
const changed = ref(false);
const invalid = ref(false);
const inputEl = ref<HTMLInputElement | null>(null);
const prefixEl = ref<HTMLElement | null>(null);
const suffixEl = ref<HTMLElement | null>(null);

let debounceTimer: number | null = null;

function focus(): void {
	inputEl.value?.focus();
}

function selectRange(start: number, end: number): void {
	inputEl.value?.setSelectionRange(start, end);
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
	if (props.type === "number") {
		emit("update:modelValue", Number.parseFloat(String(value.value)));
	} else {
		emit("update:modelValue", value.value);
	}
}

function scheduleCommit(): void {
	if (props.manualSave) return;
	if (props.debounce) {
		if (debounceTimer != null) window.clearTimeout(debounceTimer);
		debounceTimer = window.setTimeout(commit, 1000);
	} else {
		commit();
	}
}

watch(
	() => props.modelValue,
	(next) => {
		value.value = next;
	},
);

watch(value, () => {
	scheduleCommit();
	invalid.value = inputEl.value?.validity.badInput ?? false;
});

useInterval(
	() => {
		if (prefixEl.value != null && prefixEl.value.offsetWidth > 0) {
			inputEl.value?.style.setProperty(
				"padding-inline-start",
				`${prefixEl.value.offsetWidth}px`,
			);
		}
		if (suffixEl.value != null && suffixEl.value.offsetWidth > 0) {
			inputEl.value?.style.setProperty(
				"padding-inline-end",
				`${suffixEl.value.offsetWidth}px`,
			);
		}
	},
	100,
	{ immediate: true, afterMounted: true },
);

onMounted(() => {
	nextTick(() => {
		if (props.autofocus) focus();
	});
});

defineExpose({ focus, selectRange });
</script>

<style lang="scss" scoped>
.matxzzsk {
	position: relative;

	> label {
		display: block;
	}

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
		// `v-adaptive-border` picks a visible edge on flush surfaces and the
		// field's own fill on contrasting ones; the token is the fallback.
		border-color: var(--cf-adaptive-border-color, var(--inputBorder));
		border-radius: 6px;
		background: var(--bg);
		transition: border-color 0.2s;
		overflow: hidden;

		&.focused {
			border-color: var(--accent);
		}

		&.disabled {
			opacity: 0.6;
		}

		&.inline {
			display: inline-flex;
		}

		input {
			flex: 1;
			min-inline-size: 0;
			padding: 9px 12px;
			background: none;
			border: none;
			outline: none;
			color: var(--fg);
			font: inherit;
		}
	}

	.prefix,
	.suffix {
		position: absolute;
		inset-block: 0;
		display: flex;
		align-items: center;
		padding-inline: 10px;
		color: var(--fgTransparentWeak);
		pointer-events: none;
	}

	.prefix {
		inset-inline-start: 0;
	}

	.suffix {
		inset-inline-end: 0;
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
