// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/form/input.vue
//
// Adaptations: the throttle-debounce dependency and the manualSave/useInterval
// machinery are dropped — Cinderfish forms always save on submit, and the
// prefix/suffix padding is measured on input/focus events instead of a polling
// interval. Styles are plain CSS over the Firefish token set.
<template>
	<div class="cinderfish-input">
		<label>
			<div class="label"><slot name="label"></slot></div>
			<div class="input" :class="{ inline, disabled, focused }">
				<div ref="prefixEl" class="prefix">
					<slot name="prefix"></slot>
				</div>
				<input
					ref="inputEl"
					v-model="v"
					:type="type"
					:disabled="disabled"
					:required="required"
					:readonly="readonly"
					:placeholder="placeholder"
					:pattern="pattern"
					:autocomplete="autocomplete"
					:spellcheck="spellcheck"
					:step="step"
					:autofocus="autofocus"
					@focus="focused = true"
					@blur="focused = false"
					@keydown="onKeydown($event)"
					@input="onInput"
				/>
				<div ref="suffixEl" class="suffix">
					<slot name="suffix"></slot>
				</div>
			</div>
			<div class="caption"><slot name="caption"></slot></div>
		</label>
	</div>
</template>

<script lang="ts" setup>
import { nextTick, onMounted, ref, toRefs, watch } from "vue";

const props = defineProps<{
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
	inline?: boolean;
	small?: boolean;
	large?: boolean;
}>();

const emit = defineEmits<{
	(ev: "change", _ev: Event): void;
	(ev: "keydown", _ev: KeyboardEvent): void;
	(ev: "enter"): void;
	(ev: "update:modelValue", value: string | number | null): void;
}>();

const { modelValue, autofocus } = toRefs(props);
const v = ref(modelValue.value);
const focused = ref(false);
const inputEl = ref<HTMLInputElement>();
const prefixEl = ref<HTMLElement>();
const suffixEl = ref<HTMLElement>();
const height = props.small ? 36 : props.large ? 40 : 38;

const focus = () => inputEl.value?.focus();

const onInput = (ev: Event) => {
	emit("change", ev);
};

const onKeydown = (ev: KeyboardEvent) => {
	emit("keydown", ev);

	if (ev.code === "Enter") {
		emit("enter");
	}
};

// Prefix/suffix slots overlay the input, so the input's padding must match
// their measured width. One pass on mount and again on focus covers the
// layouts this port is used in without Firefish's polling interval.
function measureOverlays() {
	if (prefixEl.value && prefixEl.value.offsetWidth) {
		inputEl.value!.style.paddingInlineStart = `${prefixEl.value.offsetWidth}px`;
	}
	if (suffixEl.value && suffixEl.value.offsetWidth) {
		inputEl.value!.style.paddingInlineEnd = `${suffixEl.value.offsetWidth}px`;
	}
}

watch(modelValue, (newValue) => {
	v.value = newValue;
});

watch(v, (value) => {
	emit("update:modelValue", value);
});

onMounted(() => {
	measureOverlays();
	nextTick(() => {
		if (autofocus.value) focus();
	});
});

defineExpose({
	focus,
});
</script>

<style scoped>
.cinderfish-input > label > .label {
	padding-block-end: 8px;
	font-size: 0.85em;
	user-select: none;
}

.cinderfish-input > label > .label:empty {
	display: none;
}

.cinderfish-input > label > .caption {
	padding-block-start: 8px;
	font-size: 0.85em;
	color: var(--fgTransparentWeak);
}

.cinderfish-input > label > .caption:empty {
	display: none;
}

.cinderfish-input > label > .input {
	position: relative;
}

.cinderfish-input > label > .input > input {
	appearance: none;
	-webkit-appearance: none;
	display: block;
	block-size: v-bind("height + 'px'");
	inline-size: 100%;
	margin: 0;
	padding-block: 0;
	padding-inline: 12px;
	font: inherit;
	font-weight: normal;
	font-size: 1em;
	color: var(--fg);
	background: var(--panel);
	border: solid 1px var(--panel);
	border-radius: 6px;
	outline: none;
	box-shadow: none;
	box-sizing: border-box;
	transition: border-color 0.1s ease-out;
}

.cinderfish-input > label > .input > input:hover {
	border-color: var(--panelHighlight);
}

.cinderfish-input > label > .input.focused > input {
	border-color: var(--accent);
}

.cinderfish-input > label > .input.disabled {
	opacity: 0.7;
}

.cinderfish-input > label > .input.disabled,
.cinderfish-input > label > .input.disabled * {
	cursor: not-allowed !important;
}

.cinderfish-input > label > .input > .prefix,
.cinderfish-input > label > .input > .suffix {
	display: flex;
	align-items: center;
	position: absolute;
	z-index: 1;
	inset-block-start: 0;
	padding-block: 0;
	padding-inline: 12px;
	font-size: 1em;
	block-size: v-bind("height + 'px'");
	pointer-events: none;
}

.cinderfish-input > label > .input > .prefix:empty,
.cinderfish-input > label > .input > .suffix:empty {
	display: none;
}

.cinderfish-input > label > .input > .prefix {
	inset-inline-start: 0;
	padding-inline-end: 6px;
}

.cinderfish-input > label > .input > .suffix {
	inset-inline-end: 0;
	padding-inline-start: 6px;
}

.cinderfish-input > label > .input.inline {
	display: inline-block;
	margin: 0;
}
</style>
